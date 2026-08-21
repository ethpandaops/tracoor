package indexer

import (
	"context"
	"math"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/ethpandaops/tracoor/pkg/server/ethereum"
	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	ServiceType      = "tracoor.indexer"
	metricsNamespace = "tracoor_indexer"

	KeyNode                    = "node"
	KeyBlockRoot               = "block_root"
	KeyBlockHash               = "block_hash"
	KeyBlockNumber             = "block_number"
	KeyExecutionImplementation = "execution_implementation"
	KeyNetwork                 = "network"
	KeySlot                    = "slot"
	KeyEpoch                   = "epoch"
	KeyStateRoot               = "state_root"
	KeyContentEncoding         = "content_encoding"
	KeyNodeVersion             = "node_version"
	KeyLocation                = "location"
	KeyFetchedAt               = "fetched_at"
	KeyBeaconImplementation    = "beacon_implementation"
	KeyID                      = "id"
	KeyIndex                   = "index"
	KeyLockKey                 = "lock_key"
	KeyKind                    = "kind"
	KeyDedupKey                = "dedup_key"

	OrderFetchedAtDesc = "fetched_at DESC"
	OrderFetchedAtAsc  = "fetched_at ASC"
)

type Indexer struct {
	indexer.IndexerServer

	log logrus.FieldLogger

	store store.Store

	db *persistence.Indexer

	config *Config

	ethereumConfig *ethereum.Config

	permanentStore *PermanentStore

	metrics *Metrics

	// reaper carries object deletes that the store refused, between retention passes.
	reaper *objectReaper

	// disagreements keeps the root-disagreement report to one per distinct observation.
	disagreements *disagreementReporter

	// blobs carries the collection failure counts that decide when a payload the collector
	// cannot evaluate stops being retried.
	blobs *blobQuarantine

	// archiveBudget bounds how long one page of blocks may spend waiting on the permanent
	// store before the rest of the page is left for the next cycle.
	archiveBudget time.Duration
}

func NewIndexer(ctx context.Context, log logrus.FieldLogger, conf *Config, db *persistence.Indexer, st store.Store, ethereumConfig *ethereum.Config) (*Indexer, error) {
	// Generate a unique node ID for this instance
	nodeID := uuid.New().String()

	permanentStore, err := NewPermanentStore(log, st, db, nodeID, &conf.PermanentStore)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create permanent store")
	}

	i := &Indexer{
		log:            log.WithField("server/module", ServiceType),
		db:             db,
		store:          st,
		config:         conf,
		ethereumConfig: ethereumConfig,
		permanentStore: permanentStore,
		metrics:        NewMetrics(metricsNamespace),
		reaper:         newObjectReaper(),
		disagreements:  newDisagreementReporter(),
		blobs:          newBlobQuarantine(),
		archiveBudget:  permanentStoreArchiveBudget,
	}

	return i, nil
}

func (i *Indexer) Start(ctx context.Context, grpcServer *grpc.Server) error {
	i.log.Info("Starting module")

	if err := i.store.Healthy(ctx); err != nil {
		return errors.Wrap(err, "failed to connect to store")
	}

	if err := i.permanentStore.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start permanent store")
	}

	indexer.RegisterIndexerServer(grpcServer, i)

	go i.startRetentionWatchers(ctx)

	return nil
}

func (i *Indexer) Stop(ctx context.Context) error {
	i.log.Info("Stopping module")

	if err := i.permanentStore.Stop(ctx); err != nil {
		i.log.WithError(err).Error("Failed to stop permanent store")
	}

	// Wait for all requests to finish?

	return nil
}

func (i *Indexer) Store() store.Store {
	return i.store
}

func (i *Indexer) GetConfig(ctx context.Context, req *indexer.GetConfigRequest) (*indexer.GetConfigResponse, error) {
	return &indexer.GetConfigResponse{
		Config: &indexer.Config{
			Ethereum: &indexer.EthereumConfig{
				Config: &indexer.EthereumNetworkConfig{
					Repository: wrapperspb.String(i.ethereumConfig.Config.Repository),
					Branch:     wrapperspb.String(i.ethereumConfig.Config.Branch),
					Path:       wrapperspb.String(i.ethereumConfig.Config.Path),
				},
				Tools: &indexer.ToolsConfig{
					Ncli: &indexer.GitRepositoryConfig{
						Repository: wrapperspb.String(i.ethereumConfig.Tools.Ncli.Repository),
						Branch:     wrapperspb.String(i.ethereumConfig.Tools.Ncli.Branch),
					},
					Lcli: &indexer.GitRepositoryConfig{
						Repository: wrapperspb.String(i.ethereumConfig.Tools.Lcli.Repository),
						Branch:     wrapperspb.String(i.ethereumConfig.Tools.Lcli.Branch),
					},
					Zcli: &indexer.ZcliConfig{
						Fork: wrapperspb.String(i.ethereumConfig.Tools.Zcli.Fork),
					},
				},
			},
		},
	}, nil
}

// gateOnStore confirms the object is really there before an index entry claims it exists.
//
// The check is skipped for exactly one case: the write is taking a reference on a ready blob
// stored at the same location. That blob row was written by an agent that had just uploaded
// the object and it is a stronger, cheaper statement than a HEAD request, which every node
// observing the same payload would otherwise repeat.
func (i *Indexer) gateOnStore(ctx context.Context, p *persistence.InsertArtifactParams) error {
	if p.DedupKey != "" && p.ContentHash != "" {
		blob, err := i.db.GetBlob(ctx, p.Kind, p.Network, p.DedupKey)

		switch {
		case err == nil && blob.Location == p.Location:
			return nil
		case err != nil && !errors.Is(err, persistence.ErrBlobNotFound):
			return status.Error(codes.Internal, err.Error())
		}
	}

	exists, err := i.store.Exists(ctx, p.Location)
	if err != nil {
		i.log.
			WithError(err).
			WithField(KeyLocation, p.Location).
			WithField(KeyKind, p.Kind).
			Error("Failed to index an artifact because the store could not be reached. Check that the agent and server are pointed at the same storage backend.")

		return status.Error(codes.Internal, err.Error())
	}

	if !exists {
		return status.Error(codes.FailedPrecondition, "object not present in store")
	}

	return nil
}

// indexArtifact runs the write path shared by all seven kinds: gate on the store, insert
// against the unique index, take a blob reference if the row is linking one. subject names the
// kind in the errors the agent sees.
func (i *Indexer) indexArtifact(ctx context.Context, p *persistence.InsertArtifactParams, subject string, logFields logrus.Fields) error {
	if err := i.gateOnStore(ctx, p); err != nil {
		return err
	}

	outcome, err := i.db.InsertArtifact(ctx, p)
	if err != nil {
		i.log.WithError(err).WithFields(logFields).Error("Failed to index " + subject)

		return status.Error(codes.Internal, "failed to index "+subject)
	}

	switch outcome {
	case persistence.ArtifactInsertDuplicate:
		return status.Error(codes.AlreadyExists, subject+" already indexed")
	case persistence.ArtifactInsertUnlinkable:
		// Distinguishable on purpose: the agent's payload is not in the store any more, so it
		// re-fetches and re-uploads rather than retrying the same write.
		return status.Error(codes.FailedPrecondition, "blob not linkable")
	case persistence.ArtifactInsertLinked, persistence.ArtifactInsertUnlinked:
		return nil
	}

	return nil
}

func (i *Indexer) GetStorageHandshakeToken(ctx context.Context, req *indexer.GetStorageHandshakeTokenRequest) (*indexer.GetStorageHandshakeTokenResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	token, err := i.store.GetStorageHandshakeToken(ctx, req.Node)
	if err != nil {
		i.log.WithError(err).WithField(KeyNode, req.GetNode()).Debug("Failed to get storage handshake")

		return nil, status.Error(codes.Internal, err.Error())
	}

	if token != req.GetToken() {
		i.log.
			WithField("agent", req.GetNode()).
			Warn(`Storage handshake token mismatch.
			It's highly likely that the node is not pointed at the same storage backend as the indexer.
			Check the storage backend configuration for both the indexer and the agent instance.`)

		return nil, status.Error(codes.Unauthenticated, "storage handshake token mismatch")
	}

	return &indexer.GetStorageHandshakeTokenResponse{
		Token: token,
	}, nil
}

func (i *Indexer) CreateBeaconState(ctx context.Context, req *indexer.CreateBeaconStateRequest) (*indexer.CreateBeaconStateResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Create the state
	state := &indexer.BeaconState{
		Id:                   wrapperspb.String(uuid.New().String()),
		Node:                 req.GetNode(),
		Network:              req.GetNetwork(),
		Slot:                 req.GetSlot(),
		Epoch:                req.GetEpoch(),
		StateRoot:            req.GetStateRoot(),
		NodeVersion:          req.GetNodeVersion(),
		ContentEncoding:      req.GetContentEncoding(),
		Location:             req.GetLocation(),
		FetchedAt:            req.GetFetchedAt(),
		BeaconImplementation: req.GetBeaconImplementation(),
		ContentHash:          req.GetContentHash(),
		VerifiedAt:           req.GetVerifiedAt(),
		ContentMatchedAt:     req.GetContentMatchedAt(),
	}

	if err := state.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	logFields := logrus.Fields{
		KeyNode:                 req.GetNode().GetValue(),
		KeyNetwork:              req.GetNetwork().GetValue(),
		KeySlot:                 req.GetSlot().GetValue(),
		KeyEpoch:                req.GetEpoch().GetValue(),
		KeyStateRoot:            req.GetStateRoot().GetValue(),
		KeyContentEncoding:      req.GetContentEncoding().GetValue(),
		KeyNodeVersion:          req.GetNodeVersion().GetValue(),
		KeyLocation:             req.GetLocation().GetValue(),
		KeyFetchedAt:            req.GetFetchedAt().AsTime(),
		KeyBeaconImplementation: req.GetBeaconImplementation().GetValue(),
	}

	params := &persistence.InsertArtifactParams{
		Row:             ProtoBeaconStateToDBBeaconState(state),
		ConflictColumns: []string{KeyNetwork, KeySlot, KeyStateRoot, KeyNode},
		Kind:            persistence.KindBeaconState,
		Network:         req.GetNetwork().GetValue(),
		DedupKey:        req.GetDedupKey().GetValue(),
		ContentHash:     req.GetContentHash().GetValue(),
		Location:        req.GetLocation().GetValue(),
	}

	if err := i.indexArtifact(ctx, params, "beacon state", logFields); err != nil {
		return nil, err
	}

	i.log.WithFields(logFields).WithField("id", state.GetId().GetValue()).Debug("Indexed beacon state")

	return &indexer.CreateBeaconStateResponse{
		Id: state.GetId(),
	}, nil
}

// agreementRef identifies one verified payload: the network it was observed on and the blob
// location its row references. The location, not the content hash, is the key — a linked row
// carries its blob's location, and blobs under different dedupe keys can share bytes (an empty
// trace is identical JSON on every client build) without their counts being one number. Counts
// are resolved per ref rather than per row so one blob query covers a whole page of rows that
// mostly share payloads.
type agreementRef struct {
	network  string
	location string
}

// verifiedRef builds the ref for a row, or a zero ref for one that carries no evidence — an
// unverified row, one with no recorded hash, or one with no location has no agreement to
// count.
func verifiedRef(network, hash, location string, verifiedAt *time.Time) agreementRef {
	if verifiedAt == nil || hash == "" || location == "" {
		return agreementRef{}
	}

	return agreementRef{network: network, location: location}
}

// agreementCounts resolves how many rows share each referenced payload's bytes, from the
// ready blobs' reference counts. A missing key means no ready blob backs that location and
// the count is unknown. Failures degrade to what has been resolved so far: the count is
// decoration on a list response, not data worth failing the request over.
func (i *Indexer) agreementCounts(ctx context.Context, kind string, refs map[agreementRef]struct{}) map[agreementRef]uint32 {
	byNetwork := make(map[string][]string)

	for ref := range refs {
		if ref == (agreementRef{}) {
			continue
		}

		byNetwork[ref.network] = append(byNetwork[ref.network], ref.location)
	}

	out := make(map[agreementRef]uint32, len(refs))

	for network, locations := range byNetwork {
		counts, err := i.db.AgreementCountsByLocation(ctx, kind, network, locations)
		if err != nil {
			i.log.WithError(err).WithField("kind", kind).Warn("Failed to resolve payload agreement counts")

			continue
		}

		for location, count := range counts {
			if count < 0 || count > math.MaxUint32 {
				continue
			}

			out[agreementRef{network: network, location: location}] = uint32(count)
		}
	}

	return out
}

func (i *Indexer) ListBeaconState(ctx context.Context, req *indexer.ListBeaconStateRequest) (*indexer.ListBeaconStateResponse, error) {
	filter := &persistence.BeaconStateFilter{}

	if req.Id != "" {
		filter.AddID(req.Id)
	}

	if req.Node != "" {
		filter.AddNode(req.Node)
	}

	if req.Slot != 0 {
		filter.AddSlot(req.Slot)
	}

	if req.Epoch != 0 {
		filter.AddEpoch(req.Epoch)
	}

	if req.StateRoot != "" {
		filter.AddStateRoot(req.StateRoot)
	}

	if req.NodeVersion != "" {
		filter.AddNodeVersion(req.NodeVersion)
	}

	if req.Location != "" {
		filter.AddLocation(req.Location)
	}

	if req.Network != "" {
		filter.AddNetwork(req.Network)
	}

	if req.Before != nil {
		filter.AddBefore(req.Before.AsTime())
	}

	if req.After != nil {
		filter.AddAfter(req.After.AsTime())
	}

	if req.BeaconImplementation != "" {
		filter.AddBeaconImplementation(req.BeaconImplementation)
	}

	pagination := &persistence.PaginationCursor{
		Limit:   1000,
		Offset:  0,
		OrderBy: OrderFetchedAtDesc,
	}

	if req.Pagination != nil {
		p, err := ProtoPaginationCursorToDBPaginationCursor(req.Pagination)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		pagination = p
	}

	beaconStates, err := i.db.ListBeaconState(ctx, filter, pagination)
	if err != nil {
		return nil, err
	}

	refs := make(map[agreementRef]struct{}, len(beaconStates))
	for _, state := range beaconStates {
		refs[verifiedRef(state.Network, state.ContentHash, state.Location, state.VerifiedAt)] = struct{}{}
	}

	counts := i.agreementCounts(ctx, persistence.KindBeaconState, refs)

	protoBeaconStates := make([]*indexer.BeaconState, len(beaconStates))
	for idx, state := range beaconStates {
		proto := DBBeaconStateToProtoBeaconState(state)
		if count, ok := counts[verifiedRef(state.Network, state.ContentHash, state.Location, state.VerifiedAt)]; ok {
			proto.AgreementCount = wrapperspb.UInt32(count)
		}

		protoBeaconStates[idx] = proto
	}

	return &indexer.ListBeaconStateResponse{
		BeaconStates: protoBeaconStates,
	}, nil
}

func (i *Indexer) CountBeaconState(ctx context.Context, req *indexer.CountBeaconStateRequest) (*indexer.CountBeaconStateResponse, error) {
	filter := &persistence.BeaconStateFilter{}

	if req.Id != "" {
		filter.AddID(req.Id)
	}

	if req.Node != "" {
		filter.AddNode(req.Node)
	}

	if req.Slot != 0 {
		filter.AddSlot(req.Slot)
	}

	if req.Epoch != 0 {
		filter.AddEpoch(req.Epoch)
	}

	if req.StateRoot != "" {
		filter.AddStateRoot(req.StateRoot)
	}

	if req.NodeVersion != "" {
		filter.AddNodeVersion(req.NodeVersion)
	}

	if req.Location != "" {
		filter.AddLocation(req.Location)
	}

	if req.Network != "" {
		filter.AddNetwork(req.Network)
	}

	if req.Before != nil {
		filter.AddBefore(req.Before.AsTime())
	}

	if req.After != nil {
		filter.AddAfter(req.After.AsTime())
	}

	if req.BeaconImplementation != "" {
		filter.AddBeaconImplementation(req.BeaconImplementation)
	}

	beaconStates, err := i.db.CountBeaconState(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &indexer.CountBeaconStateResponse{
		//nolint:gosec // not worried about int64 overflow here
		Count: wrapperspb.UInt64(uint64(beaconStates)),
	}, nil
}

func (i *Indexer) ListUniqueBeaconStateValues(ctx context.Context, req *indexer.ListUniqueBeaconStateValuesRequest) (*indexer.ListUniqueBeaconStateValuesResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	fields := make([]string, len(req.Fields))

	for idx, field := range req.Fields {
		switch field {
		case indexer.ListUniqueBeaconStateValuesRequest_NODE:
			fields[idx] = KeyNode
		case indexer.ListUniqueBeaconStateValuesRequest_SLOT:
			fields[idx] = KeySlot
		case indexer.ListUniqueBeaconStateValuesRequest_EPOCH:
			fields[idx] = KeyEpoch
		case indexer.ListUniqueBeaconStateValuesRequest_STATE_ROOT:
			fields[idx] = KeyStateRoot
		case indexer.ListUniqueBeaconStateValuesRequest_NODE_VERSION:
			fields[idx] = KeyNodeVersion
		case indexer.ListUniqueBeaconStateValuesRequest_LOCATION:
			fields[idx] = KeyLocation
		case indexer.ListUniqueBeaconStateValuesRequest_NETWORK:
			fields[idx] = KeyNetwork
		case indexer.ListUniqueBeaconStateValuesRequest_BEACON_IMPLEMENTATION:
			fields[idx] = KeyBeaconImplementation
		}
	}

	distinctValues, err := i.db.DistinctBeaconStateValues(ctx, fields, req.GetNetwork())
	if err != nil {
		return nil, err
	}

	response := &indexer.ListUniqueBeaconStateValuesResponse{
		Node:                 distinctValues.Node,
		Slot:                 distinctValues.Slot,
		Epoch:                distinctValues.Epoch,
		StateRoot:            distinctValues.StateRoot,
		NodeVersion:          distinctValues.NodeVersion,
		Location:             distinctValues.Location,
		Network:              distinctValues.Network,
		BeaconImplementation: distinctValues.BeaconImplementation,
	}

	return response, nil
}

func (i *Indexer) CreateBeaconBlock(ctx context.Context, req *indexer.CreateBeaconBlockRequest) (*indexer.CreateBeaconBlockResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Create the block
	block := &indexer.BeaconBlock{
		Id:                   wrapperspb.String(uuid.New().String()),
		Node:                 req.GetNode(),
		Network:              req.GetNetwork(),
		Slot:                 req.GetSlot(),
		Epoch:                req.GetEpoch(),
		BlockRoot:            req.GetBlockRoot(),
		NodeVersion:          req.GetNodeVersion(),
		ContentEncoding:      req.GetContentEncoding(),
		Location:             req.GetLocation(),
		FetchedAt:            req.GetFetchedAt(),
		BeaconImplementation: req.GetBeaconImplementation(),
		ContentHash:          req.GetContentHash(),
		VerifiedAt:           req.GetVerifiedAt(),
		ContentMatchedAt:     req.GetContentMatchedAt(),
	}

	if err := block.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	logFields := logrus.Fields{
		KeyNode:                 req.GetNode().GetValue(),
		KeyNetwork:              req.GetNetwork().GetValue(),
		KeySlot:                 req.GetSlot().GetValue(),
		KeyEpoch:                req.GetEpoch().GetValue(),
		KeyBlockRoot:            req.GetBlockRoot().GetValue(),
		KeyNodeVersion:          req.GetNodeVersion().GetValue(),
		KeyContentEncoding:      req.GetContentEncoding().GetValue(),
		KeyLocation:             req.GetLocation().GetValue(),
		KeyFetchedAt:            req.GetFetchedAt().AsTime(),
		KeyBeaconImplementation: req.GetBeaconImplementation().GetValue(),
	}

	params := &persistence.InsertArtifactParams{
		Row:             ProtoBeaconBlockToDBBeaconBlock(block),
		ConflictColumns: []string{KeyNetwork, KeySlot, KeyBlockRoot, KeyNode},
		Kind:            persistence.KindBeaconBlock,
		Network:         req.GetNetwork().GetValue(),
		DedupKey:        req.GetDedupKey().GetValue(),
		ContentHash:     req.GetContentHash().GetValue(),
		Location:        req.GetLocation().GetValue(),
	}

	if err := i.indexArtifact(ctx, params, "beacon block", logFields); err != nil {
		return nil, err
	}

	i.log.WithFields(logFields).WithField("id", block.GetId().GetValue()).Debug("Indexed beacon block")

	// Queue the block for permanent storage
	i.permanentStore.QueueBlock(PermanentStoreBlock{
		Kind:      persistence.KindBeaconBlock,
		Location:  req.GetLocation().GetValue(),
		BlockRoot: req.GetBlockRoot().GetValue(),
		Network:   req.GetNetwork().GetValue(),
		Slot:      phase0.Slot(req.GetSlot().GetValue()),
	})

	return &indexer.CreateBeaconBlockResponse{
		Id: block.GetId(),
	}, nil
}

func (i *Indexer) ListBeaconBlock(ctx context.Context, req *indexer.ListBeaconBlockRequest) (*indexer.ListBeaconBlockResponse, error) {
	filter := &persistence.BeaconBlockFilter{}

	if req.Id != "" {
		filter.AddID(req.Id)
	}

	if req.Node != "" {
		filter.AddNode(req.Node)
	}

	if req.Slot != 0 {
		filter.AddSlot(req.Slot)
	}

	if req.Epoch != 0 {
		filter.AddEpoch(req.Epoch)
	}

	if req.BlockRoot != "" {
		filter.AddBlockRoot(req.BlockRoot)
	}

	if req.NodeVersion != "" {
		filter.AddNodeVersion(req.NodeVersion)
	}

	if req.Location != "" {
		filter.AddLocation(req.Location)
	}

	if req.Network != "" {
		filter.AddNetwork(req.Network)
	}

	if req.Before != nil {
		filter.AddBefore(req.Before.AsTime())
	}

	if req.After != nil {
		filter.AddAfter(req.After.AsTime())
	}

	if req.BeaconImplementation != "" {
		filter.AddBeaconImplementation(req.BeaconImplementation)
	}

	pagination := &persistence.PaginationCursor{
		Limit:   1000,
		Offset:  0,
		OrderBy: OrderFetchedAtDesc,
	}

	if req.Pagination != nil {
		p, err := ProtoPaginationCursorToDBPaginationCursor(req.Pagination)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		pagination = p
	}

	beaconBlocks, err := i.db.ListBeaconBlock(ctx, filter, pagination)
	if err != nil {
		return nil, err
	}

	refs := make(map[agreementRef]struct{}, len(beaconBlocks))
	for _, block := range beaconBlocks {
		refs[verifiedRef(block.Network, block.ContentHash, block.Location, block.VerifiedAt)] = struct{}{}
	}

	counts := i.agreementCounts(ctx, persistence.KindBeaconBlock, refs)

	protoBeaconBlocks := make([]*indexer.BeaconBlock, len(beaconBlocks))
	for idx, block := range beaconBlocks {
		proto := DBBeaconBlockToProtoBeaconBlock(block)
		if count, ok := counts[verifiedRef(block.Network, block.ContentHash, block.Location, block.VerifiedAt)]; ok {
			proto.AgreementCount = wrapperspb.UInt32(count)
		}

		protoBeaconBlocks[idx] = proto
	}

	return &indexer.ListBeaconBlockResponse{
		BeaconBlocks: protoBeaconBlocks,
	}, nil
}

func (i *Indexer) CountBeaconBlock(ctx context.Context, req *indexer.CountBeaconBlockRequest) (*indexer.CountBeaconBlockResponse, error) {
	filter := &persistence.BeaconBlockFilter{}

	if req.Id != "" {
		filter.AddID(req.Id)
	}

	if req.Node != "" {
		filter.AddNode(req.Node)
	}

	if req.Slot != 0 {
		filter.AddSlot(req.Slot)
	}

	if req.Epoch != 0 {
		filter.AddEpoch(req.Epoch)
	}

	if req.BlockRoot != "" {
		filter.AddBlockRoot(req.BlockRoot)
	}

	if req.NodeVersion != "" {
		filter.AddNodeVersion(req.NodeVersion)
	}

	if req.Location != "" {
		filter.AddLocation(req.Location)
	}

	if req.Network != "" {
		filter.AddNetwork(req.Network)
	}

	if req.Before != nil {
		filter.AddBefore(req.Before.AsTime())
	}

	if req.After != nil {
		filter.AddAfter(req.After.AsTime())
	}

	if req.BeaconImplementation != "" {
		filter.AddBeaconImplementation(req.BeaconImplementation)
	}

	beaconBlocks, err := i.db.CountBeaconBlock(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &indexer.CountBeaconBlockResponse{
		//nolint:gosec // not worried about int64 overflow here
		Count: wrapperspb.UInt64(uint64(beaconBlocks)),
	}, nil
}

func (i *Indexer) ListUniqueBeaconBlockValues(ctx context.Context, req *indexer.ListUniqueBeaconBlockValuesRequest) (*indexer.ListUniqueBeaconBlockValuesResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	fields := make([]string, len(req.Fields))

	for idx, field := range req.Fields {
		switch field {
		case indexer.ListUniqueBeaconBlockValuesRequest_NODE:
			fields[idx] = KeyNode
		case indexer.ListUniqueBeaconBlockValuesRequest_SLOT:
			fields[idx] = KeySlot
		case indexer.ListUniqueBeaconBlockValuesRequest_EPOCH:
			fields[idx] = KeyEpoch
		case indexer.ListUniqueBeaconBlockValuesRequest_BLOCK_ROOT:
			fields[idx] = KeyBlockRoot
		case indexer.ListUniqueBeaconBlockValuesRequest_NODE_VERSION:
			fields[idx] = KeyNodeVersion
		case indexer.ListUniqueBeaconBlockValuesRequest_LOCATION:
			fields[idx] = KeyLocation
		case indexer.ListUniqueBeaconBlockValuesRequest_NETWORK:
			fields[idx] = KeyNetwork
		case indexer.ListUniqueBeaconBlockValuesRequest_BEACON_IMPLEMENTATION:
			fields[idx] = KeyBeaconImplementation
		}
	}

	distinctValues, err := i.db.DistinctBeaconBlockValues(ctx, fields, req.GetNetwork())
	if err != nil {
		return nil, err
	}

	response := &indexer.ListUniqueBeaconBlockValuesResponse{
		Node:                 distinctValues.Node,
		Slot:                 distinctValues.Slot,
		Epoch:                distinctValues.Epoch,
		BlockRoot:            distinctValues.BlockRoot,
		NodeVersion:          distinctValues.NodeVersion,
		Location:             distinctValues.Location,
		Network:              distinctValues.Network,
		BeaconImplementation: distinctValues.BeaconImplementation,
	}

	return response, nil
}

func (i *Indexer) CreateExecutionPayloadEnvelope(ctx context.Context, req *indexer.CreateExecutionPayloadEnvelopeRequest) (*indexer.CreateExecutionPayloadEnvelopeResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Create the envelope
	envelope := &indexer.ExecutionPayloadEnvelope{
		Id:                   wrapperspb.String(uuid.New().String()),
		Node:                 req.GetNode(),
		Network:              req.GetNetwork(),
		Slot:                 req.GetSlot(),
		Epoch:                req.GetEpoch(),
		BlockRoot:            req.GetBlockRoot(),
		NodeVersion:          req.GetNodeVersion(),
		ContentEncoding:      req.GetContentEncoding(),
		Location:             req.GetLocation(),
		FetchedAt:            req.GetFetchedAt(),
		BeaconImplementation: req.GetBeaconImplementation(),
		ContentHash:          req.GetContentHash(),
		VerifiedAt:           req.GetVerifiedAt(),
		ContentMatchedAt:     req.GetContentMatchedAt(),
	}

	if err := envelope.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	logFields := logrus.Fields{
		KeyNode:                 req.GetNode().GetValue(),
		KeyNetwork:              req.GetNetwork().GetValue(),
		KeySlot:                 req.GetSlot().GetValue(),
		KeyEpoch:                req.GetEpoch().GetValue(),
		KeyBlockRoot:            req.GetBlockRoot().GetValue(),
		KeyNodeVersion:          req.GetNodeVersion().GetValue(),
		KeyContentEncoding:      req.GetContentEncoding().GetValue(),
		KeyLocation:             req.GetLocation().GetValue(),
		KeyFetchedAt:            req.GetFetchedAt().AsTime(),
		KeyBeaconImplementation: req.GetBeaconImplementation().GetValue(),
	}

	params := &persistence.InsertArtifactParams{
		Row:             ProtoExecutionPayloadEnvelopeToDBExecutionPayloadEnvelope(envelope),
		ConflictColumns: []string{KeyNetwork, KeySlot, KeyBlockRoot, KeyNode},
		Kind:            persistence.KindExecutionPayloadEnvelope,
		Network:         req.GetNetwork().GetValue(),
		DedupKey:        req.GetDedupKey().GetValue(),
		ContentHash:     req.GetContentHash().GetValue(),
		Location:        req.GetLocation().GetValue(),
	}

	if err := i.indexArtifact(ctx, params, "execution payload envelope", logFields); err != nil {
		return nil, err
	}

	i.log.WithFields(logFields).WithField("id", envelope.GetId().GetValue()).Debug("Indexed execution payload envelope")

	// Queue the envelope for permanent storage. From gloas on it carries the execution
	// payload the block no longer does, so archiving one without the other keeps a slot
	// that cannot be re-executed.
	i.permanentStore.QueueBlock(PermanentStoreBlock{
		Kind:      persistence.KindExecutionPayloadEnvelope,
		Location:  req.GetLocation().GetValue(),
		BlockRoot: req.GetBlockRoot().GetValue(),
		Network:   req.GetNetwork().GetValue(),
		Slot:      phase0.Slot(req.GetSlot().GetValue()),
	})

	return &indexer.CreateExecutionPayloadEnvelopeResponse{
		Id: envelope.GetId(),
	}, nil
}

func (i *Indexer) ListExecutionPayloadEnvelope(ctx context.Context, req *indexer.ListExecutionPayloadEnvelopeRequest) (*indexer.ListExecutionPayloadEnvelopeResponse, error) {
	filter := &persistence.ExecutionPayloadEnvelopeFilter{}

	if req.Id != "" {
		filter.AddID(req.Id)
	}

	if req.Node != "" {
		filter.AddNode(req.Node)
	}

	if req.Slot != 0 {
		filter.AddSlot(req.Slot)
	}

	if req.Epoch != 0 {
		filter.AddEpoch(req.Epoch)
	}

	if req.BlockRoot != "" {
		filter.AddBlockRoot(req.BlockRoot)
	}

	if req.NodeVersion != "" {
		filter.AddNodeVersion(req.NodeVersion)
	}

	if req.Location != "" {
		filter.AddLocation(req.Location)
	}

	if req.Network != "" {
		filter.AddNetwork(req.Network)
	}

	if req.Before != nil {
		filter.AddBefore(req.Before.AsTime())
	}

	if req.After != nil {
		filter.AddAfter(req.After.AsTime())
	}

	if req.BeaconImplementation != "" {
		filter.AddBeaconImplementation(req.BeaconImplementation)
	}

	pagination := &persistence.PaginationCursor{
		Limit:   1000,
		Offset:  0,
		OrderBy: OrderFetchedAtDesc,
	}

	if req.Pagination != nil {
		p, err := ProtoPaginationCursorToDBPaginationCursor(req.Pagination)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		pagination = p
	}

	envelopes, err := i.db.ListExecutionPayloadEnvelope(ctx, filter, pagination)
	if err != nil {
		return nil, err
	}

	refs := make(map[agreementRef]struct{}, len(envelopes))
	for _, envelope := range envelopes {
		refs[verifiedRef(envelope.Network, envelope.ContentHash, envelope.Location, envelope.VerifiedAt)] = struct{}{}
	}

	counts := i.agreementCounts(ctx, persistence.KindExecutionPayloadEnvelope, refs)

	protoEnvelopes := make([]*indexer.ExecutionPayloadEnvelope, len(envelopes))
	for idx, envelope := range envelopes {
		proto := DBExecutionPayloadEnvelopeToProtoExecutionPayloadEnvelope(envelope)
		if count, ok := counts[verifiedRef(envelope.Network, envelope.ContentHash, envelope.Location, envelope.VerifiedAt)]; ok {
			proto.AgreementCount = wrapperspb.UInt32(count)
		}

		protoEnvelopes[idx] = proto
	}

	return &indexer.ListExecutionPayloadEnvelopeResponse{
		ExecutionPayloadEnvelopes: protoEnvelopes,
	}, nil
}

func (i *Indexer) CountExecutionPayloadEnvelope(ctx context.Context, req *indexer.CountExecutionPayloadEnvelopeRequest) (*indexer.CountExecutionPayloadEnvelopeResponse, error) {
	filter := &persistence.ExecutionPayloadEnvelopeFilter{}

	if req.Id != "" {
		filter.AddID(req.Id)
	}

	if req.Node != "" {
		filter.AddNode(req.Node)
	}

	if req.Slot != 0 {
		filter.AddSlot(req.Slot)
	}

	if req.Epoch != 0 {
		filter.AddEpoch(req.Epoch)
	}

	if req.BlockRoot != "" {
		filter.AddBlockRoot(req.BlockRoot)
	}

	if req.NodeVersion != "" {
		filter.AddNodeVersion(req.NodeVersion)
	}

	if req.Location != "" {
		filter.AddLocation(req.Location)
	}

	if req.Network != "" {
		filter.AddNetwork(req.Network)
	}

	if req.Before != nil {
		filter.AddBefore(req.Before.AsTime())
	}

	if req.After != nil {
		filter.AddAfter(req.After.AsTime())
	}

	if req.BeaconImplementation != "" {
		filter.AddBeaconImplementation(req.BeaconImplementation)
	}

	envelopes, err := i.db.CountExecutionPayloadEnvelope(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &indexer.CountExecutionPayloadEnvelopeResponse{
		//nolint:gosec // not worried about int64 overflow here
		Count: wrapperspb.UInt64(uint64(envelopes)),
	}, nil
}

func (i *Indexer) ListUniqueExecutionPayloadEnvelopeValues(ctx context.Context, req *indexer.ListUniqueExecutionPayloadEnvelopeValuesRequest) (*indexer.ListUniqueExecutionPayloadEnvelopeValuesResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	fields := make([]string, len(req.Fields))

	for idx, field := range req.Fields {
		switch field {
		case indexer.ListUniqueExecutionPayloadEnvelopeValuesRequest_NODE:
			fields[idx] = KeyNode
		case indexer.ListUniqueExecutionPayloadEnvelopeValuesRequest_SLOT:
			fields[idx] = KeySlot
		case indexer.ListUniqueExecutionPayloadEnvelopeValuesRequest_EPOCH:
			fields[idx] = KeyEpoch
		case indexer.ListUniqueExecutionPayloadEnvelopeValuesRequest_BLOCK_ROOT:
			fields[idx] = KeyBlockRoot
		case indexer.ListUniqueExecutionPayloadEnvelopeValuesRequest_NODE_VERSION:
			fields[idx] = KeyNodeVersion
		case indexer.ListUniqueExecutionPayloadEnvelopeValuesRequest_LOCATION:
			fields[idx] = KeyLocation
		case indexer.ListUniqueExecutionPayloadEnvelopeValuesRequest_NETWORK:
			fields[idx] = KeyNetwork
		case indexer.ListUniqueExecutionPayloadEnvelopeValuesRequest_BEACON_IMPLEMENTATION:
			fields[idx] = KeyBeaconImplementation
		}
	}

	distinctValues, err := i.db.DistinctExecutionPayloadEnvelopeValues(ctx, fields, req.GetNetwork())
	if err != nil {
		return nil, err
	}

	response := &indexer.ListUniqueExecutionPayloadEnvelopeValuesResponse{
		Node:                 distinctValues.Node,
		Slot:                 distinctValues.Slot,
		Epoch:                distinctValues.Epoch,
		BlockRoot:            distinctValues.BlockRoot,
		NodeVersion:          distinctValues.NodeVersion,
		Location:             distinctValues.Location,
		Network:              distinctValues.Network,
		BeaconImplementation: distinctValues.BeaconImplementation,
	}

	return response, nil
}

func (i *Indexer) CreateBeaconBadBlock(ctx context.Context, req *indexer.CreateBeaconBadBlockRequest) (*indexer.CreateBeaconBadBlockResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Create the bad block
	badBlock := &indexer.BeaconBadBlock{
		Id:                   wrapperspb.String(uuid.New().String()),
		Node:                 req.GetNode(),
		Network:              req.GetNetwork(),
		Slot:                 req.GetSlot(),
		Epoch:                req.GetEpoch(),
		BlockRoot:            req.GetBlockRoot(),
		NodeVersion:          req.GetNodeVersion(),
		ContentEncoding:      req.GetContentEncoding(),
		Location:             req.GetLocation(),
		FetchedAt:            req.GetFetchedAt(),
		BeaconImplementation: req.GetBeaconImplementation(),
		ContentHash:          req.GetContentHash(),
		VerifiedAt:           req.GetVerifiedAt(),
	}

	if err := badBlock.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	logFields := logrus.Fields{
		KeyNode:                 req.GetNode().GetValue(),
		KeyNetwork:              req.GetNetwork().GetValue(),
		KeySlot:                 req.GetSlot().GetValue(),
		KeyEpoch:                req.GetEpoch().GetValue(),
		KeyBlockRoot:            req.GetBlockRoot().GetValue(),
		KeyNodeVersion:          req.GetNodeVersion().GetValue(),
		KeyContentEncoding:      req.GetContentEncoding().GetValue(),
		KeyLocation:             req.GetLocation().GetValue(),
		KeyFetchedAt:            req.GetFetchedAt().AsTime(),
		KeyBeaconImplementation: req.GetBeaconImplementation().GetValue(),
	}

	// A bad block never joins the deduplicated set: its bytes are the evidence of a fault and
	// are kept per node, so it carries no dedupe key and owns its object outright.
	params := &persistence.InsertArtifactParams{
		Row:             ProtoBeaconBadBlockToDBBeaconBadBlock(badBlock),
		ConflictColumns: []string{KeyNetwork, KeySlot, KeyBlockRoot, KeyNode},
		Kind:            persistence.KindBeaconBadBlock,
		Network:         req.GetNetwork().GetValue(),
		Location:        req.GetLocation().GetValue(),
	}

	if err := i.indexArtifact(ctx, params, "beacon bad block", logFields); err != nil {
		return nil, err
	}

	i.log.WithFields(logFields).WithField("id", badBlock.GetId().GetValue()).Debug("Indexed beacon block")

	return &indexer.CreateBeaconBadBlockResponse{
		Id: badBlock.GetId(),
	}, nil
}

func (i *Indexer) ListBeaconBadBlock(ctx context.Context, req *indexer.ListBeaconBadBlockRequest) (*indexer.ListBeaconBadBlockResponse, error) {
	filter := &persistence.BeaconBadBlockFilter{}

	if req.Id != "" {
		filter.AddID(req.Id)
	}

	if req.Node != "" {
		filter.AddNode(req.Node)
	}

	if req.Slot != 0 {
		filter.AddSlot(req.Slot)
	}

	if req.Epoch != 0 {
		filter.AddEpoch(req.Epoch)
	}

	if req.BlockRoot != "" {
		filter.AddBlockRoot(req.BlockRoot)
	}

	if req.NodeVersion != "" {
		filter.AddNodeVersion(req.NodeVersion)
	}

	if req.Location != "" {
		filter.AddLocation(req.Location)
	}

	if req.Network != "" {
		filter.AddNetwork(req.Network)
	}

	if req.Before != nil {
		filter.AddBefore(req.Before.AsTime())
	}

	if req.After != nil {
		filter.AddAfter(req.After.AsTime())
	}

	if req.BeaconImplementation != "" {
		filter.AddBeaconImplementation(req.BeaconImplementation)
	}

	pagination := &persistence.PaginationCursor{
		Limit:   1000,
		Offset:  0,
		OrderBy: OrderFetchedAtDesc,
	}

	if req.Pagination != nil {
		p, err := ProtoPaginationCursorToDBPaginationCursor(req.Pagination)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		pagination = p
	}

	beaconBlocks, err := i.db.ListBeaconBadBlock(ctx, filter, pagination)
	if err != nil {
		return nil, err
	}

	protoBeaconBadBlocks := make([]*indexer.BeaconBadBlock, len(beaconBlocks))
	for i, badBlock := range beaconBlocks {
		protoBeaconBadBlocks[i] = DBBeaconBadBlockToProtoBeaconBadBlock(badBlock)
	}

	return &indexer.ListBeaconBadBlockResponse{
		BeaconBadBlocks: protoBeaconBadBlocks,
	}, nil
}

func (i *Indexer) CountBeaconBadBlock(ctx context.Context, req *indexer.CountBeaconBadBlockRequest) (*indexer.CountBeaconBadBlockResponse, error) {
	filter := &persistence.BeaconBadBlockFilter{}

	if req.Id != "" {
		filter.AddID(req.Id)
	}

	if req.Node != "" {
		filter.AddNode(req.Node)
	}

	if req.Slot != 0 {
		filter.AddSlot(req.Slot)
	}

	if req.Epoch != 0 {
		filter.AddEpoch(req.Epoch)
	}

	if req.BlockRoot != "" {
		filter.AddBlockRoot(req.BlockRoot)
	}

	if req.NodeVersion != "" {
		filter.AddNodeVersion(req.NodeVersion)
	}

	if req.Location != "" {
		filter.AddLocation(req.Location)
	}

	if req.Network != "" {
		filter.AddNetwork(req.Network)
	}

	if req.Before != nil {
		filter.AddBefore(req.Before.AsTime())
	}

	if req.After != nil {
		filter.AddAfter(req.After.AsTime())
	}

	if req.BeaconImplementation != "" {
		filter.AddBeaconImplementation(req.BeaconImplementation)
	}

	beaconBlocks, err := i.db.CountBeaconBadBlock(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &indexer.CountBeaconBadBlockResponse{
		//nolint:gosec // not worried about int64 overflow here
		Count: wrapperspb.UInt64(uint64(beaconBlocks)),
	}, nil
}

func (i *Indexer) ListUniqueBeaconBadBlockValues(ctx context.Context, req *indexer.ListUniqueBeaconBadBlockValuesRequest) (*indexer.ListUniqueBeaconBadBlockValuesResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	fields := make([]string, len(req.Fields))

	for idx, field := range req.Fields {
		switch field {
		case indexer.ListUniqueBeaconBadBlockValuesRequest_NODE:
			fields[idx] = KeyNode
		case indexer.ListUniqueBeaconBadBlockValuesRequest_SLOT:
			fields[idx] = KeySlot
		case indexer.ListUniqueBeaconBadBlockValuesRequest_EPOCH:
			fields[idx] = KeyEpoch
		case indexer.ListUniqueBeaconBadBlockValuesRequest_BLOCK_ROOT:
			fields[idx] = KeyBlockRoot
		case indexer.ListUniqueBeaconBadBlockValuesRequest_NODE_VERSION:
			fields[idx] = KeyNodeVersion
		case indexer.ListUniqueBeaconBadBlockValuesRequest_LOCATION:
			fields[idx] = KeyLocation
		case indexer.ListUniqueBeaconBadBlockValuesRequest_NETWORK:
			fields[idx] = KeyNetwork
		case indexer.ListUniqueBeaconBadBlockValuesRequest_BEACON_IMPLEMENTATION:
			fields[idx] = KeyBeaconImplementation
		}
	}

	distinctValues, err := i.db.DistinctBeaconBadBlockValues(ctx, fields, req.GetNetwork())
	if err != nil {
		return nil, err
	}

	response := &indexer.ListUniqueBeaconBadBlockValuesResponse{
		Node:                 distinctValues.Node,
		Slot:                 distinctValues.Slot,
		Epoch:                distinctValues.Epoch,
		BlockRoot:            distinctValues.BlockRoot,
		NodeVersion:          distinctValues.NodeVersion,
		Location:             distinctValues.Location,
		Network:              distinctValues.Network,
		BeaconImplementation: distinctValues.BeaconImplementation,
	}

	return response, nil
}

func (i *Indexer) CreateBeaconBadBlob(ctx context.Context, req *indexer.CreateBeaconBadBlobRequest) (*indexer.CreateBeaconBadBlobResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Create the bad blob
	badBlob := &indexer.BeaconBadBlob{
		Id:                   wrapperspb.String(uuid.New().String()),
		Node:                 req.GetNode(),
		Network:              req.GetNetwork(),
		Slot:                 req.GetSlot(),
		Epoch:                req.GetEpoch(),
		BlockRoot:            req.GetBlockRoot(),
		NodeVersion:          req.GetNodeVersion(),
		ContentEncoding:      req.GetContentEncoding(),
		Location:             req.GetLocation(),
		FetchedAt:            req.GetFetchedAt(),
		BeaconImplementation: req.GetBeaconImplementation(),
		Index:                req.GetIndex(),
		ContentHash:          req.GetContentHash(),
		VerifiedAt:           req.GetVerifiedAt(),
	}

	if err := badBlob.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	logFields := logrus.Fields{
		KeyNode:                 req.GetNode().GetValue(),
		KeyNetwork:              req.GetNetwork().GetValue(),
		KeySlot:                 req.GetSlot().GetValue(),
		KeyEpoch:                req.GetEpoch().GetValue(),
		KeyBlockRoot:            req.GetBlockRoot().GetValue(),
		KeyNodeVersion:          req.GetNodeVersion().GetValue(),
		KeyContentEncoding:      req.GetContentEncoding().GetValue(),
		KeyLocation:             req.GetLocation().GetValue(),
		KeyFetchedAt:            req.GetFetchedAt().AsTime(),
		KeyBeaconImplementation: req.GetBeaconImplementation().GetValue(),
		KeyIndex:                req.GetIndex().GetValue(),
	}

	params := &persistence.InsertArtifactParams{
		Row:             ProtoBeaconBadBlobToDBBeaconBadBlob(badBlob),
		ConflictColumns: []string{KeyNetwork, KeySlot, KeyBlockRoot, KeyIndex, KeyNode},
		Kind:            persistence.KindBeaconBadBlob,
		Network:         req.GetNetwork().GetValue(),
		Location:        req.GetLocation().GetValue(),
	}

	if err := i.indexArtifact(ctx, params, "beacon bad blob", logFields); err != nil {
		return nil, err
	}

	i.log.WithFields(logFields).WithField("id", badBlob.GetId().GetValue()).Debug("Indexed beacon blob")

	return &indexer.CreateBeaconBadBlobResponse{
		Id: badBlob.GetId(),
	}, nil
}

func (i *Indexer) ListBeaconBadBlob(ctx context.Context, req *indexer.ListBeaconBadBlobRequest) (*indexer.ListBeaconBadBlobResponse, error) {
	filter := &persistence.BeaconBadBlobFilter{}

	if req.Id != "" {
		filter.AddID(req.Id)
	}

	if req.Node != "" {
		filter.AddNode(req.Node)
	}

	if req.Slot != 0 {
		filter.AddSlot(req.Slot)
	}

	if req.Epoch != 0 {
		filter.AddEpoch(req.Epoch)
	}

	if req.BlockRoot != "" {
		filter.AddBlockRoot(req.BlockRoot)
	}

	if req.NodeVersion != "" {
		filter.AddNodeVersion(req.NodeVersion)
	}

	if req.Location != "" {
		filter.AddLocation(req.Location)
	}

	if req.Network != "" {
		filter.AddNetwork(req.Network)
	}

	if req.Before != nil {
		filter.AddBefore(req.Before.AsTime())
	}

	if req.After != nil {
		filter.AddAfter(req.After.AsTime())
	}

	if req.BeaconImplementation != "" {
		filter.AddBeaconImplementation(req.BeaconImplementation)
	}

	if req.Index != nil {
		filter.AddIndex(req.Index.GetValue())
	}

	pagination := &persistence.PaginationCursor{
		Limit:   1000,
		Offset:  0,
		OrderBy: OrderFetchedAtDesc,
	}

	if req.Pagination != nil {
		p, err := ProtoPaginationCursorToDBPaginationCursor(req.Pagination)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		pagination = p
	}

	beaconBlobs, err := i.db.ListBeaconBadBlob(ctx, filter, pagination)
	if err != nil {
		return nil, err
	}

	protoBeaconBadBlobs := make([]*indexer.BeaconBadBlob, len(beaconBlobs))
	for i, badBlob := range beaconBlobs {
		protoBeaconBadBlobs[i] = DBBeaconBadBlobToProtoBeaconBadBlob(badBlob)
	}

	return &indexer.ListBeaconBadBlobResponse{
		BeaconBadBlobs: protoBeaconBadBlobs,
	}, nil
}

func (i *Indexer) CountBeaconBadBlob(ctx context.Context, req *indexer.CountBeaconBadBlobRequest) (*indexer.CountBeaconBadBlobResponse, error) {
	filter := &persistence.BeaconBadBlobFilter{}

	if req.Id != "" {
		filter.AddID(req.Id)
	}

	if req.Node != "" {
		filter.AddNode(req.Node)
	}

	if req.Slot != 0 {
		filter.AddSlot(req.Slot)
	}

	if req.Epoch != 0 {
		filter.AddEpoch(req.Epoch)
	}

	if req.BlockRoot != "" {
		filter.AddBlockRoot(req.BlockRoot)
	}

	if req.NodeVersion != "" {
		filter.AddNodeVersion(req.NodeVersion)
	}

	if req.Location != "" {
		filter.AddLocation(req.Location)
	}

	if req.Network != "" {
		filter.AddNetwork(req.Network)
	}

	if req.Before != nil {
		filter.AddBefore(req.Before.AsTime())
	}

	if req.After != nil {
		filter.AddAfter(req.After.AsTime())
	}

	if req.BeaconImplementation != "" {
		filter.AddBeaconImplementation(req.BeaconImplementation)
	}

	if req.Index != nil {
		filter.AddIndex(req.Index.GetValue())
	}

	beaconBlobs, err := i.db.CountBeaconBadBlob(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &indexer.CountBeaconBadBlobResponse{
		//nolint:gosec // not worried about int64 overflow here
		Count: wrapperspb.UInt64(uint64(beaconBlobs)),
	}, nil
}

func (i *Indexer) ListUniqueBeaconBadBlobValues(ctx context.Context, req *indexer.ListUniqueBeaconBadBlobValuesRequest) (*indexer.ListUniqueBeaconBadBlobValuesResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	fields := make([]string, len(req.Fields))

	for idx, field := range req.Fields {
		switch field {
		case indexer.ListUniqueBeaconBadBlobValuesRequest_NODE:
			fields[idx] = KeyNode
		case indexer.ListUniqueBeaconBadBlobValuesRequest_SLOT:
			fields[idx] = KeySlot
		case indexer.ListUniqueBeaconBadBlobValuesRequest_EPOCH:
			fields[idx] = KeyEpoch
		case indexer.ListUniqueBeaconBadBlobValuesRequest_BLOCK_ROOT:
			fields[idx] = KeyBlockRoot
		case indexer.ListUniqueBeaconBadBlobValuesRequest_NODE_VERSION:
			fields[idx] = KeyNodeVersion
		case indexer.ListUniqueBeaconBadBlobValuesRequest_LOCATION:
			fields[idx] = KeyLocation
		case indexer.ListUniqueBeaconBadBlobValuesRequest_NETWORK:
			fields[idx] = KeyNetwork
		case indexer.ListUniqueBeaconBadBlobValuesRequest_BEACON_IMPLEMENTATION:
			fields[idx] = KeyBeaconImplementation
		case indexer.ListUniqueBeaconBadBlobValuesRequest_INDEX:
			fields[idx] = "index"
		}
	}

	distinctValues, err := i.db.DistinctBeaconBadBlobValues(ctx, fields, req.GetNetwork())
	if err != nil {
		return nil, err
	}

	response := &indexer.ListUniqueBeaconBadBlobValuesResponse{
		Node:                 distinctValues.Node,
		Slot:                 distinctValues.Slot,
		Epoch:                distinctValues.Epoch,
		BlockRoot:            distinctValues.BlockRoot,
		NodeVersion:          distinctValues.NodeVersion,
		Location:             distinctValues.Location,
		Network:              distinctValues.Network,
		BeaconImplementation: distinctValues.BeaconImplementation,
		Index:                distinctValues.Index,
	}

	return response, nil
}

func (i *Indexer) CreateExecutionBlockTrace(ctx context.Context, req *indexer.CreateExecutionBlockTraceRequest) (*indexer.CreateExecutionBlockTraceResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Create the execution block trace
	trace := &indexer.ExecutionBlockTrace{
		Id:                      wrapperspb.String(uuid.New().String()),
		Node:                    req.GetNode(),
		FetchedAt:               req.GetFetchedAt(),
		BlockHash:               req.GetBlockHash(),
		BlockNumber:             req.GetBlockNumber(),
		Location:                req.GetLocation(),
		Network:                 req.GetNetwork(),
		ExecutionImplementation: req.GetExecutionImplementation(),
		NodeVersion:             req.GetNodeVersion(),
		ContentHash:             req.GetContentHash(),
		VerifiedAt:              req.GetVerifiedAt(),
		ContentMatchedAt:        req.GetContentMatchedAt(),
	}

	if err := trace.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	logFields := logrus.Fields{
		KeyNode:        req.GetNode().GetValue(),
		KeyNetwork:     req.GetNetwork().GetValue(),
		KeyNodeVersion: req.GetNodeVersion().GetValue(),
		KeyLocation:    req.GetLocation().GetValue(),
		KeyFetchedAt:   req.GetFetchedAt().AsTime(),
	}

	params := &persistence.InsertArtifactParams{
		Row:             ProtoExecutionBlockTraceToDBExecutionBlockTrace(trace),
		ConflictColumns: []string{KeyNetwork, KeyBlockHash, KeyNode},
		Kind:            persistence.KindExecutionBlockTrace,
		Network:         req.GetNetwork().GetValue(),
		DedupKey:        req.GetDedupKey().GetValue(),
		ContentHash:     req.GetContentHash().GetValue(),
		Location:        req.GetLocation().GetValue(),
	}

	if err := i.indexArtifact(ctx, params, "execution block trace", logFields); err != nil {
		return nil, err
	}

	i.log.WithFields(logFields).WithField("id", trace.GetId().GetValue()).Debug("Indexed execution block trace")

	return &indexer.CreateExecutionBlockTraceResponse{
		Id: trace.GetId(),
	}, nil
}

func (i *Indexer) ListExecutionBlockTrace(ctx context.Context, req *indexer.ListExecutionBlockTraceRequest) (*indexer.ListExecutionBlockTraceResponse, error) {
	filter := &persistence.ExecutionBlockTraceFilter{}

	if req.Id != "" {
		filter.AddID(req.Id)
	}

	if req.Node != "" {
		filter.AddNode(req.Node)
	}

	if req.BlockNumber != 0 {
		filter.AddBlockNumber(req.BlockNumber)
	}

	if req.BlockHash != "" {
		filter.AddBlockHash(req.BlockHash)
	}

	if req.Location != "" {
		filter.AddLocation(req.Location)
	}

	if req.Network != "" {
		filter.AddNetwork(req.Network)
	}

	if req.Before != nil {
		filter.AddBefore(req.Before.AsTime())
	}

	if req.After != nil {
		filter.AddAfter(req.After.AsTime())
	}

	if req.ExecutionImplementation != "" {
		filter.AddExecutionImplementation(req.ExecutionImplementation)
	}

	if req.NodeVersion != "" {
		filter.AddNodeVersion(req.NodeVersion)
	}

	pagination := &persistence.PaginationCursor{
		Limit:   1000,
		Offset:  0,
		OrderBy: OrderFetchedAtDesc,
	}

	if req.Pagination != nil {
		p, err := ProtoPaginationCursorToDBPaginationCursor(req.Pagination)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		pagination = p
	}

	executionBlockTraces, err := i.db.ListExecutionBlockTrace(ctx, filter, pagination)
	if err != nil {
		return nil, err
	}

	refs := make(map[agreementRef]struct{}, len(executionBlockTraces))
	for _, trace := range executionBlockTraces {
		refs[verifiedRef(trace.Network, trace.ContentHash, trace.Location, trace.VerifiedAt)] = struct{}{}
	}

	counts := i.agreementCounts(ctx, persistence.KindExecutionBlockTrace, refs)

	protoExecutionBlockTraces := make([]*indexer.ExecutionBlockTrace, len(executionBlockTraces))
	for idx, trace := range executionBlockTraces {
		proto := DBExecutionBlockTraceToProtoExecutionBlockTrace(trace)
		if count, ok := counts[verifiedRef(trace.Network, trace.ContentHash, trace.Location, trace.VerifiedAt)]; ok {
			proto.AgreementCount = wrapperspb.UInt32(count)
		}

		protoExecutionBlockTraces[idx] = proto
	}

	return &indexer.ListExecutionBlockTraceResponse{
		ExecutionBlockTraces: protoExecutionBlockTraces,
	}, nil
}

func (i *Indexer) CountExecutionBlockTrace(ctx context.Context, req *indexer.CountExecutionBlockTraceRequest) (*indexer.CountExecutionBlockTraceResponse, error) {
	filter := &persistence.ExecutionBlockTraceFilter{}

	if req.Id != "" {
		filter.AddID(req.Id)
	}

	if req.Node != "" {
		filter.AddNode(req.Node)
	}

	if req.BlockNumber != 0 {
		filter.AddBlockNumber(req.BlockNumber)
	}

	if req.BlockHash != "" {
		filter.AddBlockHash(req.BlockHash)
	}

	if req.Location != "" {
		filter.AddLocation(req.Location)
	}

	if req.Network != "" {
		filter.AddNetwork(req.Network)
	}

	if req.Before != nil {
		filter.AddBefore(req.Before.AsTime())
	}

	if req.After != nil {
		filter.AddAfter(req.After.AsTime())
	}

	if req.ExecutionImplementation != "" {
		filter.AddExecutionImplementation(req.ExecutionImplementation)
	}

	if req.NodeVersion != "" {
		filter.AddNodeVersion(req.NodeVersion)
	}

	executionBlockTraces, err := i.db.CountExecutionBlockTrace(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &indexer.CountExecutionBlockTraceResponse{
		//nolint:gosec // not worried about int64 overflow here
		Count: wrapperspb.UInt64(uint64(executionBlockTraces)),
	}, nil
}

func (i *Indexer) ListUniqueExecutionBlockTraceValues(ctx context.Context, req *indexer.ListUniqueExecutionBlockTraceValuesRequest) (*indexer.ListUniqueExecutionBlockTraceValuesResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	fields := make([]string, len(req.Fields))

	for idx, field := range req.Fields {
		switch field {
		case indexer.ListUniqueExecutionBlockTraceValuesRequest_NODE:
			fields[idx] = KeyNode
		case indexer.ListUniqueExecutionBlockTraceValuesRequest_BLOCK_HASH:
			fields[idx] = KeyBlockHash
		case indexer.ListUniqueExecutionBlockTraceValuesRequest_BLOCK_NUMBER:
			fields[idx] = KeyBlockNumber
		case indexer.ListUniqueExecutionBlockTraceValuesRequest_LOCATION:
			fields[idx] = KeyLocation
		case indexer.ListUniqueExecutionBlockTraceValuesRequest_NETWORK:
			fields[idx] = KeyNetwork
		case indexer.ListUniqueExecutionBlockTraceValuesRequest_EXECUTION_IMPLEMENTATION:
			fields[idx] = KeyExecutionImplementation
		case indexer.ListUniqueExecutionBlockTraceValuesRequest_NODE_VERSION:
			fields[idx] = KeyNodeVersion
		}
	}

	distinctValues, err := i.db.DistinctExecutionBlockTraceValues(ctx, fields, req.GetNetwork())
	if err != nil {
		return nil, err
	}

	response := &indexer.ListUniqueExecutionBlockTraceValuesResponse{
		Node:                    distinctValues.Node,
		BlockHash:               distinctValues.BlockHash,
		BlockNumber:             distinctValues.BlockNumber,
		Location:                distinctValues.Location,
		Network:                 distinctValues.Network,
		ExecutionImplementation: distinctValues.ExecutionImplementation,
		NodeVersion:             distinctValues.NodeVersion,
	}

	return response, nil
}

func (i *Indexer) CreateExecutionBadBlock(ctx context.Context, req *indexer.CreateExecutionBadBlockRequest) (*indexer.CreateExecutionBadBlockResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Create the execution bad block
	block := &indexer.ExecutionBadBlock{
		Id:                      wrapperspb.String(uuid.New().String()),
		Node:                    req.GetNode(),
		FetchedAt:               req.GetFetchedAt(),
		BlockHash:               req.GetBlockHash(),
		BlockNumber:             req.GetBlockNumber(),
		BlockExtraData:          req.GetBlockExtraData(),
		Location:                req.GetLocation(),
		ContentEncoding:         req.GetContentEncoding(),
		Network:                 req.GetNetwork(),
		ExecutionImplementation: req.GetExecutionImplementation(),
		NodeVersion:             req.GetNodeVersion(),
		ContentHash:             req.GetContentHash(),
		VerifiedAt:              req.GetVerifiedAt(),
	}

	if err := block.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	logFields := logrus.Fields{
		KeyNode:            req.GetNode().GetValue(),
		KeyNetwork:         req.GetNetwork().GetValue(),
		KeyNodeVersion:     req.GetNodeVersion().GetValue(),
		KeyContentEncoding: req.GetContentEncoding().GetValue(),
		KeyLocation:        req.GetLocation().GetValue(),
		KeyFetchedAt:       req.GetFetchedAt().AsTime(),
	}

	params := &persistence.InsertArtifactParams{
		Row:             ProtoExecutionBadBlockToDBExecutionBadBlock(block),
		ConflictColumns: []string{KeyNetwork, KeyBlockHash, KeyNode},
		Kind:            persistence.KindExecutionBadBlock,
		Network:         req.GetNetwork().GetValue(),
		Location:        req.GetLocation().GetValue(),
	}

	if err := i.indexArtifact(ctx, params, "execution bad block", logFields); err != nil {
		return nil, err
	}

	i.log.WithFields(logFields).WithField("id", block.GetId().GetValue()).Debug("Indexed execution bad block")

	return &indexer.CreateExecutionBadBlockResponse{
		Id: block.GetId(),
	}, nil
}

func (i *Indexer) ListExecutionBadBlock(ctx context.Context, req *indexer.ListExecutionBadBlockRequest) (*indexer.ListExecutionBadBlockResponse, error) {
	filter := &persistence.ExecutionBadBlockFilter{}

	if req.Id != "" {
		filter.AddID(req.Id)
	}

	if req.Node != "" {
		filter.AddNode(req.Node)
	}

	if req.BlockNumber != 0 {
		filter.AddBlockNumber(req.BlockNumber)
	}

	if req.BlockHash != "" {
		filter.AddBlockHash(req.BlockHash)
	}

	if req.Location != "" {
		filter.AddLocation(req.Location)
	}

	if req.Network != "" {
		filter.AddNetwork(req.Network)
	}

	if req.Before != nil {
		filter.AddBefore(req.Before.AsTime())
	}

	if req.After != nil {
		filter.AddAfter(req.After.AsTime())
	}

	if req.ExecutionImplementation != "" {
		filter.AddExecutionImplementation(req.ExecutionImplementation)
	}

	if req.NodeVersion != "" {
		filter.AddNodeVersion(req.NodeVersion)
	}

	if req.BlockExtraData != "" {
		filter.AddBlockExtraData(req.BlockExtraData)
	}

	pagination := &persistence.PaginationCursor{
		Limit:   1000,
		Offset:  0,
		OrderBy: OrderFetchedAtDesc,
	}

	if req.Pagination != nil {
		p, err := ProtoPaginationCursorToDBPaginationCursor(req.Pagination)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		pagination = p
	}

	executionBadBlocks, err := i.db.ListExecutionBadBlock(ctx, filter, pagination)
	if err != nil {
		return nil, err
	}

	protoExecutionBadBlocks := make([]*indexer.ExecutionBadBlock, len(executionBadBlocks))
	for i, block := range executionBadBlocks {
		protoExecutionBadBlocks[i] = DBExecutionBadBlockToProtoExecutionBadBlock(block)
	}

	return &indexer.ListExecutionBadBlockResponse{
		ExecutionBadBlocks: protoExecutionBadBlocks,
	}, nil
}

func (i *Indexer) CountExecutionBadBlock(ctx context.Context, req *indexer.CountExecutionBadBlockRequest) (*indexer.CountExecutionBadBlockResponse, error) {
	filter := &persistence.ExecutionBadBlockFilter{}

	if req.Id != "" {
		filter.AddID(req.Id)
	}

	if req.Node != "" {
		filter.AddNode(req.Node)
	}

	if req.BlockNumber != 0 {
		filter.AddBlockNumber(req.BlockNumber)
	}

	if req.BlockHash != "" {
		filter.AddBlockHash(req.BlockHash)
	}

	if req.Location != "" {
		filter.AddLocation(req.Location)
	}

	if req.Network != "" {
		filter.AddNetwork(req.Network)
	}

	if req.Before != nil {
		filter.AddBefore(req.Before.AsTime())
	}

	if req.After != nil {
		filter.AddAfter(req.After.AsTime())
	}

	if req.ExecutionImplementation != "" {
		filter.AddExecutionImplementation(req.ExecutionImplementation)
	}

	if req.NodeVersion != "" {
		filter.AddNodeVersion(req.NodeVersion)
	}

	if req.BlockExtraData != "" {
		filter.AddBlockExtraData(req.BlockExtraData)
	}

	executionBadBlocks, err := i.db.CountExecutionBadBlock(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &indexer.CountExecutionBadBlockResponse{
		//nolint:gosec // not worried about int64 overflow here
		Count: wrapperspb.UInt64(uint64(executionBadBlocks)),
	}, nil
}

func (i *Indexer) ListUniqueExecutionBadBlockValues(ctx context.Context, req *indexer.ListUniqueExecutionBadBlockValuesRequest) (*indexer.ListUniqueExecutionBadBlockValuesResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	fields := make([]string, len(req.Fields))

	for idx, field := range req.Fields {
		switch field {
		case indexer.ListUniqueExecutionBadBlockValuesRequest_NODE:
			fields[idx] = KeyNode
		case indexer.ListUniqueExecutionBadBlockValuesRequest_BLOCK_HASH:
			fields[idx] = KeyBlockHash
		case indexer.ListUniqueExecutionBadBlockValuesRequest_BLOCK_NUMBER:
			fields[idx] = KeyBlockNumber
		case indexer.ListUniqueExecutionBadBlockValuesRequest_LOCATION:
			fields[idx] = KeyLocation
		case indexer.ListUniqueExecutionBadBlockValuesRequest_NETWORK:
			fields[idx] = KeyNetwork
		case indexer.ListUniqueExecutionBadBlockValuesRequest_EXECUTION_IMPLEMENTATION:
			fields[idx] = "execution_implementation"
		case indexer.ListUniqueExecutionBadBlockValuesRequest_NODE_VERSION:
			fields[idx] = KeyNodeVersion
		case indexer.ListUniqueExecutionBadBlockValuesRequest_BLOCK_EXTRA_DATA:
			fields[idx] = "block_extra_data"
		}
	}

	distinctValues, err := i.db.DistinctExecutionBadBlockValues(ctx, fields, req.GetNetwork())
	if err != nil {
		return nil, err
	}

	response := &indexer.ListUniqueExecutionBadBlockValuesResponse{
		Node:                    distinctValues.Node,
		BlockHash:               distinctValues.BlockHash,
		BlockNumber:             distinctValues.BlockNumber,
		BlockExtraData:          distinctValues.BlockExtraData,
		Location:                distinctValues.Location,
		Network:                 distinctValues.Network,
		ExecutionImplementation: distinctValues.ExecutionImplementation,
		NodeVersion:             distinctValues.NodeVersion,
	}

	return response, nil
}

func (i *Indexer) GetBlob(ctx context.Context, req *indexer.GetBlobRequest) (*indexer.GetBlobResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	blob, err := i.db.GetBlob(ctx, req.GetKind().GetValue(), req.GetNetwork().GetValue(), req.GetDedupKey().GetValue())
	if err != nil {
		if errors.Is(err, persistence.ErrBlobNotFound) {
			return nil, status.Error(codes.NotFound, "blob not found")
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	return &indexer.GetBlobResponse{
		Blob: DBBlobToProtoBlob(blob),
	}, nil
}

func (i *Indexer) CreateBlob(ctx context.Context, req *indexer.CreateBlobRequest) (*indexer.CreateBlobResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	blob := &persistence.Blob{
		Kind:            req.GetKind().GetValue(),
		Network:         req.GetNetwork().GetValue(),
		DedupKey:        req.GetDedupKey().GetValue(),
		ContentHash:     req.GetContentHash().GetValue(),
		Location:        req.GetLocation().GetValue(),
		ContentEncoding: req.GetContentEncoding().GetValue(),
		RawSize:         req.GetRawSize().GetValue(),
		CompressedSize:  req.GetCompressedSize().GetValue(),
		State:           persistence.BlobStateReady,
		CreatedAt:       time.Now().UTC(),
	}

	// The winner is whichever row is in the table afterwards: a concurrent creator with
	// different bytes must be visible to the loser so it can compare hashes.
	winner, err := i.db.InsertBlob(ctx, blob)
	if err != nil {
		i.log.WithError(err).WithFields(logrus.Fields{
			KeyKind:     blob.Kind,
			KeyNetwork:  blob.Network,
			KeyDedupKey: blob.DedupKey,
		}).Error("Failed to create blob")

		return nil, status.Error(codes.Internal, "failed to create blob")
	}

	return &indexer.CreateBlobResponse{
		Blob: DBBlobToProtoBlob(winner),
	}, nil
}

func (i *Indexer) CreatePayloadDivergence(ctx context.Context, req *indexer.CreatePayloadDivergenceRequest) (*indexer.CreatePayloadDivergenceResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	divergence := &indexer.PayloadDivergence{
		Id:           wrapperspb.String(uuid.New().String()),
		ObservedAt:   req.GetObservedAt(),
		Network:      req.GetNetwork(),
		Kind:         req.GetKind(),
		Node:         req.GetNode(),
		DedupKey:     req.GetDedupKey(),
		ExpectedHash: req.GetExpectedHash(),
		ActualHash:   req.GetActualHash(),
		Slot:         req.GetSlot(),
		Identifier:   req.GetIdentifier(),
		Attempt:      req.GetAttempt(),
		Severity:     req.GetSeverity(),
		Location:     req.GetLocation(),
	}

	row := ProtoPayloadDivergenceToDBPayloadDivergence(divergence)
	if req.GetObservedAt() == nil {
		row.ObservedAt = time.Now().UTC()
	}

	// Attempt 1 is the first observation; an unset attempt means exactly that.
	if req.GetAttempt() == nil {
		row.Attempt = 1
	}

	if err := i.db.InsertPayloadDivergence(ctx, row); err != nil {
		i.log.WithError(err).WithFields(logrus.Fields{
			KeyKind:     row.Kind,
			KeyNetwork:  row.Network,
			KeyDedupKey: row.DedupKey,
			KeyNode:     row.Node,
		}).Error("Failed to record payload divergence")

		return nil, status.Error(codes.Internal, "failed to record payload divergence")
	}

	i.log.WithFields(logrus.Fields{
		KeyKind:     row.Kind,
		KeyNetwork:  row.Network,
		KeyDedupKey: row.DedupKey,
		KeyNode:     row.Node,
		"expected":  row.ExpectedHash,
		"actual":    row.ActualHash,
	}).Warn("Payload divergence recorded")

	return &indexer.CreatePayloadDivergenceResponse{
		Id: divergence.GetId(),
	}, nil
}

func (i *Indexer) ListPayloadDivergence(ctx context.Context, req *indexer.ListPayloadDivergenceRequest) (*indexer.ListPayloadDivergenceResponse, error) {
	filter := &persistence.PayloadDivergenceFilter{}

	if req.GetNetwork() != "" {
		filter.AddNetwork(req.GetNetwork())
	}

	if req.GetKind() != "" {
		filter.AddKind(req.GetKind())
	}

	if req.GetDedupKey() != "" {
		filter.AddDedupKey(req.GetDedupKey())
	}

	if req.GetBefore() != nil {
		filter.AddBefore(req.GetBefore().AsTime())
	}

	if req.GetAfter() != nil {
		filter.AddAfter(req.GetAfter().AsTime())
	}

	pagination := &persistence.PaginationCursor{
		Limit:  persistence.DefaultPageLimit,
		Offset: 0,
	}

	if req.GetPagination() != nil {
		p, err := ProtoPaginationCursorToDBPaginationCursor(req.GetPagination())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		pagination = p
	}

	divergences, err := i.db.ListPayloadDivergence(ctx, filter, pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	protoDivergences := make([]*indexer.PayloadDivergence, len(divergences))
	for idx, divergence := range divergences {
		protoDivergences[idx] = DBPayloadDivergenceToProtoPayloadDivergence(divergence)
	}

	return &indexer.ListPayloadDivergenceResponse{
		PayloadDivergences: protoDivergences,
	}, nil
}

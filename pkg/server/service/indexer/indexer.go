package indexer

import (
	"context"

	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
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
	ServiceType = "tracoor.indexer"
)

type Indexer struct {
	indexer.IndexerServer

	log logrus.FieldLogger

	store store.Store

	db *persistence.Indexer

	config *Config
}

func NewIndexer(ctx context.Context, log logrus.FieldLogger, conf *Config, db *persistence.Indexer, st store.Store) (*Indexer, error) {
	i := &Indexer{
		log:    log.WithField("server/module", ServiceType),
		db:     db,
		store:  st,
		config: conf,
	}

	return i, nil
}

func (i *Indexer) Start(ctx context.Context, grpcServer *grpc.Server) error {
	i.log.Info("Starting module")

	if err := i.store.Healthy(ctx); err != nil {
		return errors.Wrap(err, "failed to connect to store")
	}

	indexer.RegisterIndexerServer(grpcServer, i)

	go i.startRetentionWatchers(ctx)

	return nil
}

func (i *Indexer) Stop(ctx context.Context) error {
	i.log.Info("Stopping module")

	// Wait for all requests to finish?

	return nil
}

func (i *Indexer) GetStorageHandshakeToken(ctx context.Context, req *indexer.GetStorageHandshakeTokenRequest) (*indexer.GetStorageHandshakeTokenResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	token, err := i.store.GetStorageHandshakeToken(ctx, req.Node)
	if err != nil {
		i.log.WithError(err).WithField("node", req.GetNode()).Debug("Failed to get storage handshake")

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

	// Check the store for the state
	exists, err := i.store.Exists(ctx, req.GetLocation().GetValue())
	if err != nil {
		i.log.
			WithError(err).
			WithField("location", req.GetLocation().GetValue()).
			WithField("node", req.GetNode().GetValue()).
			Error("Failed to index a beacon state because the state could not be found in the store. Check that the agent and server are pointed at the same storage backend.")

		return nil, status.Error(codes.Internal, err.Error())
	}

	if exists {
		// Check if the state is already indexed
		filter := &persistence.BeaconStateFilter{}

		filter.AddNetwork(req.GetNetwork().GetValue())
		filter.AddSlot(req.GetSlot().GetValue())
		filter.AddStateRoot(req.GetStateRoot().GetValue())
		filter.AddNode(req.GetNode().GetValue())

		states, err := i.db.ListBeaconState(ctx, filter, &persistence.PaginationCursor{Limit: 1, Offset: 0})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		if len(states) > 0 {
			return nil, status.Error(codes.AlreadyExists, "beacon state already indexed")
		}
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
		Location:             req.GetLocation(),
		FetchedAt:            req.GetFetchedAt(),
		BeaconImplementation: req.GetBeaconImplementation(),
	}

	if err := state.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	logFields := logrus.Fields{
		"node":                  req.GetNode().GetValue(),
		"network":               req.GetNetwork().GetValue(),
		"slot":                  req.GetSlot().GetValue(),
		"epoch":                 req.GetEpoch().GetValue(),
		"state_root":            req.GetStateRoot().GetValue(),
		"node_version":          req.GetNodeVersion().GetValue(),
		"location":              req.GetLocation().GetValue(),
		"fetched_at":            req.GetFetchedAt().AsTime(),
		"beacon_implementation": req.GetBeaconImplementation().GetValue(),
	}

	if err := i.db.InsertBeaconState(ctx, ProtoBeaconStateToDBBeaconState(state)); err != nil {
		i.log.WithError(err).WithFields(logFields).Error("Failed to index state")

		return nil, status.Error(codes.Internal, "failed to index state")
	}

	i.log.WithFields(logFields).WithField("id", state.GetId().GetValue()).Debug("Indexed beacon state")

	return &indexer.CreateBeaconStateResponse{
		Id: state.GetId(),
	}, nil
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
		OrderBy: "fetched_at DESC",
	}

	if req.Pagination != nil {
		pagination = ProtoPaginationCursorToDBPaginationCursor(req.Pagination)
	}

	beaconStates, err := i.db.ListBeaconState(ctx, filter, pagination)
	if err != nil {
		return nil, err
	}

	protoBeaconStates := make([]*indexer.BeaconState, len(beaconStates))
	for i, state := range beaconStates {
		protoBeaconStates[i] = DBBeaconStateToProtoBeaconState(state)
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
			fields[idx] = "node"
		case indexer.ListUniqueBeaconStateValuesRequest_SLOT:
			fields[idx] = "slot"
		case indexer.ListUniqueBeaconStateValuesRequest_EPOCH:
			fields[idx] = "epoch"
		case indexer.ListUniqueBeaconStateValuesRequest_STATE_ROOT:
			fields[idx] = "state_root"
		case indexer.ListUniqueBeaconStateValuesRequest_NODE_VERSION:
			fields[idx] = "node_version"
		case indexer.ListUniqueBeaconStateValuesRequest_LOCATION:
			fields[idx] = "location"
		case indexer.ListUniqueBeaconStateValuesRequest_NETWORK:
			fields[idx] = "network"
		case indexer.ListUniqueBeaconStateValuesRequest_BEACON_IMPLEMENTATION:
			fields[idx] = "beacon_implementation"
		}
	}

	distinctValues, err := i.db.DistinctBeaconStateValues(ctx, fields)
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
	}

	if err := trace.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := i.db.InsertExecutionBlockTrace(ctx, ProtoExecutionBlockTraceToDBExecutionBlockTrace(trace)); err != nil {
		return nil, status.Error(codes.Internal, "failed to insert execution block trace")
	}

	logFields := logrus.Fields{
		"node":         req.GetNode().GetValue(),
		"network":      req.GetNetwork().GetValue(),
		"node_version": req.GetNodeVersion().GetValue(),
		"location":     req.GetLocation().GetValue(),
		"fetched_at":   req.GetFetchedAt().AsTime(),
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
		OrderBy: "fetched_at DESC",
	}

	if req.Pagination != nil {
		pagination = ProtoPaginationCursorToDBPaginationCursor(req.Pagination)
	}

	executionBlockTraces, err := i.db.ListExecutionBlockTrace(ctx, filter, pagination)
	if err != nil {
		return nil, err
	}

	protoExecutionBlockTraces := make([]*indexer.ExecutionBlockTrace, len(executionBlockTraces))
	for i, trace := range executionBlockTraces {
		protoExecutionBlockTraces[i] = DBExecutionBlockTraceToProtoExecutionBlockTrace(trace)
	}

	return &indexer.ListExecutionBlockTraceResponse{
		ExecutionBlockTraces: protoExecutionBlockTraces,
	}, nil
}

func (i *Indexer) CountExecutionBlockTrace(ctx context.Context, req *indexer.CountExecutionBlockTraceRequest) (*indexer.CountExecutionBlockTraceResponse, error) {
	filter := &persistence.ExecutionBlockTraceFilter{}

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
			fields[idx] = "node"
		case indexer.ListUniqueExecutionBlockTraceValuesRequest_BLOCK_HASH:
			fields[idx] = "block_hash"
		case indexer.ListUniqueExecutionBlockTraceValuesRequest_BLOCK_NUMBER:
			fields[idx] = "block_number"
		case indexer.ListUniqueExecutionBlockTraceValuesRequest_LOCATION:
			fields[idx] = "location"
		case indexer.ListUniqueExecutionBlockTraceValuesRequest_NETWORK:
			fields[idx] = "network"
		case indexer.ListUniqueExecutionBlockTraceValuesRequest_EXECUTION_IMPLEMENTATION:
			fields[idx] = "execution_implementation"
		case indexer.ListUniqueExecutionBlockTraceValuesRequest_NODE_VERSION:
			fields[idx] = "node_version"
		}
	}

	distinctValues, err := i.db.DistinctExecutionBlockTraceValues(ctx, fields)
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
		Network:                 req.GetNetwork(),
		ExecutionImplementation: req.GetExecutionImplementation(),
		NodeVersion:             req.GetNodeVersion(),
	}

	if err := block.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := i.db.InsertExecutionBadBlock(ctx, ProtoExecutionBadBlockToDBExecutionBadBlock(block)); err != nil {
		return nil, status.Error(codes.Internal, "failed to insert execution bad block")
	}

	logFields := logrus.Fields{
		"node":         req.GetNode().GetValue(),
		"network":      req.GetNetwork().GetValue(),
		"node_version": req.GetNodeVersion().GetValue(),
		"location":     req.GetLocation().GetValue(),
		"fetched_at":   req.GetFetchedAt().AsTime(),
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
		OrderBy: "fetched_at DESC",
	}

	if req.Pagination != nil {
		pagination = ProtoPaginationCursorToDBPaginationCursor(req.Pagination)
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
			fields[idx] = "node"
		case indexer.ListUniqueExecutionBadBlockValuesRequest_BLOCK_HASH:
			fields[idx] = "block_hash"
		case indexer.ListUniqueExecutionBadBlockValuesRequest_BLOCK_NUMBER:
			fields[idx] = "block_number"
		case indexer.ListUniqueExecutionBadBlockValuesRequest_LOCATION:
			fields[idx] = "location"
		case indexer.ListUniqueExecutionBadBlockValuesRequest_NETWORK:
			fields[idx] = "network"
		case indexer.ListUniqueExecutionBadBlockValuesRequest_EXECUTION_IMPLEMENTATION:
			fields[idx] = "execution_implementation"
		case indexer.ListUniqueExecutionBadBlockValuesRequest_NODE_VERSION:
			fields[idx] = "node_version"
		case indexer.ListUniqueExecutionBadBlockValuesRequest_BLOCK_EXTRA_DATA:
			fields[idx] = "block_extra_data"
		}
	}

	distinctValues, err := i.db.DistinctExecutionBadBlockValues(ctx, fields)
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

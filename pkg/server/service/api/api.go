package api

import (
	"context"
	"fmt"

	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/api"
	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	ServiceType = "tracoor.api"

	OrderFetchedAtDesc = "fetched_at DESC"
)

type API struct {
	api.APIServer

	log logrus.FieldLogger

	store store.Store

	indexer indexer.IndexerClient

	grpcConn string
	grpcOpts []grpc.DialOption
}

func NewAPI(ctx context.Context, log logrus.FieldLogger, conf *Config, st store.Store, grpcConn string, grpcOpts []grpc.DialOption) (*API, error) {
	e := &API{
		log:      log.WithField("server/module", ServiceType),
		grpcConn: grpcConn,
		grpcOpts: grpcOpts,
		store:    st,
	}

	return e, nil
}

func (i *API) Start(ctx context.Context, grpcServer *grpc.Server) error {
	i.log.Info("Starting module")

	// Connect to the indexer
	conn, err := grpc.NewClient(i.grpcConn, i.grpcOpts...)
	if err != nil {
		return fmt.Errorf("fail to dial: %v", err)
	}

	i.indexer = indexer.NewIndexerClient(conn)

	api.RegisterAPIServer(grpcServer, i)

	return nil
}

func (i *API) Stop(ctx context.Context) error {
	i.log.Info("Stopping module")

	// Wait for all requests to finish?

	return nil
}

func (i *API) GetConfig(ctx context.Context, req *api.GetConfigRequest) (*api.GetConfigResponse, error) {
	conf, err := i.indexer.GetConfig(ctx, &indexer.GetConfigRequest{})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("failed to get config: %w", err).Error())
	}

	rsp := &api.GetConfigResponse{
		Config: &api.Config{
			Ethereum: &api.EthereumConfig{
				Config: &api.EthereumNetworkConfig{
					Repository: conf.GetConfig().GetEthereum().GetConfig().GetRepository().GetValue(),
					Branch:     conf.GetConfig().GetEthereum().GetConfig().GetBranch().GetValue(),
					Path:       conf.GetConfig().GetEthereum().GetConfig().GetPath().GetValue(),
				},
				Tools: &api.ToolsConfig{
					Ncli: &api.GitRepositoryConfig{
						Repository: conf.GetConfig().GetEthereum().GetTools().GetNcli().GetRepository().GetValue(),
						Branch:     conf.GetConfig().GetEthereum().GetTools().GetNcli().GetBranch().GetValue(),
					},
					Lcli: &api.GitRepositoryConfig{
						Repository: conf.GetConfig().GetEthereum().GetTools().GetLcli().GetRepository().GetValue(),
						Branch:     conf.GetConfig().GetEthereum().GetTools().GetLcli().GetBranch().GetValue(),
					},
					Zcli: &api.ZcliConfig{
						Fork: conf.GetConfig().GetEthereum().GetTools().GetZcli().GetFork().GetValue(),
					},
				},
			},
		},
	}

	return rsp, nil
}

func (i *API) ListBeaconState(ctx context.Context, req *api.ListBeaconStateRequest) (*api.ListBeaconStateResponse, error) {
	pagination, err := paginationFromRequest(req.Pagination)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Errorf("invalid pagination: %w", err).Error())
	}

	rq := &indexer.ListBeaconStateRequest{
		Id:                   req.Id,
		Node:                 req.Node,
		Slot:                 req.Slot,
		Epoch:                req.Epoch,
		StateRoot:            req.StateRoot,
		NodeVersion:          req.NodeVersion,
		Network:              req.Network,
		Before:               req.Before,
		After:                req.After,
		BeaconImplementation: req.BeaconImplementation,

		Pagination: pagination,
	}

	resp, err := i.indexer.ListBeaconState(ctx, rq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("failed to list beacon states: %w", err).Error())
	}

	protoBeaconStates := make([]*api.BeaconState, len(resp.BeaconStates))
	for i, state := range resp.BeaconStates {
		protoBeaconStates[i] = &api.BeaconState{
			Id:                   state.Id,
			Node:                 state.Node,
			Slot:                 state.Slot,
			Epoch:                state.Epoch,
			StateRoot:            state.StateRoot,
			NodeVersion:          state.NodeVersion,
			Network:              state.Network,
			FetchedAt:            state.FetchedAt,
			BeaconImplementation: state.BeaconImplementation,
			ContentHash:          state.ContentHash,
			VerifiedAt:           state.VerifiedAt,
			ContentMatchedAt:     state.ContentMatchedAt,
		}
	}

	return &api.ListBeaconStateResponse{
		BeaconStates: protoBeaconStates,
	}, nil
}

func (i *API) CountBeaconState(ctx context.Context, req *api.CountBeaconStateRequest) (*api.CountBeaconStateResponse, error) {
	rq := &indexer.CountBeaconStateRequest{
		Node:                 req.Node,
		Slot:                 req.Slot,
		Epoch:                req.Epoch,
		StateRoot:            req.StateRoot,
		NodeVersion:          req.NodeVersion,
		Network:              req.Network,
		BeaconImplementation: req.BeaconImplementation,
		Before:               req.Before,
		After:                req.After,
	}

	resp, err := i.indexer.CountBeaconState(ctx, rq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("failed to count beacon states: %w", err).Error())
	}

	return &api.CountBeaconStateResponse{
		Count: wrapperspb.UInt64(resp.GetCount().GetValue()),
	}, nil
}

func (i *API) ListUniqueBeaconStateValues(ctx context.Context, req *api.ListUniqueBeaconStateValuesRequest) (*api.ListUniqueBeaconStateValuesResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Errorf("invalid request: %w", err).Error())
	}

	// Create our "indexer" equivalent structs
	rq := indexer.ListUniqueBeaconStateValuesRequest{
		Fields:  []indexer.ListUniqueBeaconStateValuesRequest_Field{},
		Network: req.GetNetworkFilter(),
	}

	for _, field := range req.Fields {
		var f indexer.ListUniqueBeaconStateValuesRequest_Field

		switch field {
		case api.ListUniqueBeaconStateValuesRequest_node:
			f = indexer.ListUniqueBeaconStateValuesRequest_NODE
		case api.ListUniqueBeaconStateValuesRequest_node_version:
			f = indexer.ListUniqueBeaconStateValuesRequest_NODE_VERSION
		case api.ListUniqueBeaconStateValuesRequest_network:
			f = indexer.ListUniqueBeaconStateValuesRequest_NETWORK
		case api.ListUniqueBeaconStateValuesRequest_slot:
			f = indexer.ListUniqueBeaconStateValuesRequest_SLOT
		case api.ListUniqueBeaconStateValuesRequest_epoch:
			f = indexer.ListUniqueBeaconStateValuesRequest_EPOCH
		case api.ListUniqueBeaconStateValuesRequest_state_root:
			f = indexer.ListUniqueBeaconStateValuesRequest_STATE_ROOT
		case api.ListUniqueBeaconStateValuesRequest_beacon_implementation:
			f = indexer.ListUniqueBeaconStateValuesRequest_BEACON_IMPLEMENTATION
		default:
			return nil, status.Error(codes.InvalidArgument, fmt.Errorf("invalid field: %s", field.String()).Error())
		}

		rq.Fields = append(rq.Fields, f)
	}

	// Call the indexer
	resp, err := i.indexer.ListUniqueBeaconStateValues(ctx, &rq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("failed to list unique beacon state values: %w", err).Error())
	}

	// Convert the response
	response := &api.ListUniqueBeaconStateValuesResponse{
		Node:                 resp.Node,
		Slot:                 resp.Slot,
		Epoch:                resp.Epoch,
		StateRoot:            resp.StateRoot,
		NodeVersion:          resp.NodeVersion,
		Network:              resp.Network,
		BeaconImplementation: resp.BeaconImplementation,
	}

	return response, nil
}

func (i *API) ListBeaconBlock(ctx context.Context, req *api.ListBeaconBlockRequest) (*api.ListBeaconBlockResponse, error) {
	pagination, err := paginationFromRequest(req.Pagination)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Errorf("invalid pagination: %w", err).Error())
	}

	rq := &indexer.ListBeaconBlockRequest{
		Id:                   req.Id,
		Node:                 req.Node,
		Slot:                 req.Slot,
		Epoch:                req.Epoch,
		BlockRoot:            req.BlockRoot,
		NodeVersion:          req.NodeVersion,
		Network:              req.Network,
		Before:               req.Before,
		After:                req.After,
		BeaconImplementation: req.BeaconImplementation,

		Pagination: pagination,
	}

	resp, err := i.indexer.ListBeaconBlock(ctx, rq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("failed to list beacon blocks: %w", err).Error())
	}

	protoBeaconBlocks := make([]*api.BeaconBlock, len(resp.BeaconBlocks))
	for i, block := range resp.BeaconBlocks {
		protoBeaconBlocks[i] = &api.BeaconBlock{
			Id:                   block.Id,
			Node:                 block.Node,
			Slot:                 block.Slot,
			Epoch:                block.Epoch,
			BlockRoot:            block.BlockRoot,
			NodeVersion:          block.NodeVersion,
			Network:              block.Network,
			FetchedAt:            block.FetchedAt,
			BeaconImplementation: block.BeaconImplementation,
			ContentHash:          block.ContentHash,
			VerifiedAt:           block.VerifiedAt,
			ContentMatchedAt:     block.ContentMatchedAt,
		}
	}

	return &api.ListBeaconBlockResponse{
		BeaconBlocks: protoBeaconBlocks,
	}, nil
}

func (i *API) CountBeaconBlock(ctx context.Context, req *api.CountBeaconBlockRequest) (*api.CountBeaconBlockResponse, error) {
	rq := &indexer.CountBeaconBlockRequest{
		Node:                 req.Node,
		Slot:                 req.Slot,
		Epoch:                req.Epoch,
		BlockRoot:            req.BlockRoot,
		NodeVersion:          req.NodeVersion,
		Network:              req.Network,
		BeaconImplementation: req.BeaconImplementation,
		Before:               req.Before,
		After:                req.After,
	}

	resp, err := i.indexer.CountBeaconBlock(ctx, rq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("failed to count beacon blocks: %w", err).Error())
	}

	return &api.CountBeaconBlockResponse{
		Count: wrapperspb.UInt64(resp.GetCount().GetValue()),
	}, nil
}

func (i *API) ListUniqueBeaconBlockValues(ctx context.Context, req *api.ListUniqueBeaconBlockValuesRequest) (*api.ListUniqueBeaconBlockValuesResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Errorf("invalid request: %w", err).Error())
	}

	// Create our "indexer" equivalent structs
	rq := indexer.ListUniqueBeaconBlockValuesRequest{
		Fields:  []indexer.ListUniqueBeaconBlockValuesRequest_Field{},
		Network: req.GetNetworkFilter(),
	}

	for _, field := range req.Fields {
		var f indexer.ListUniqueBeaconBlockValuesRequest_Field

		switch field {
		case api.ListUniqueBeaconBlockValuesRequest_node:
			f = indexer.ListUniqueBeaconBlockValuesRequest_NODE
		case api.ListUniqueBeaconBlockValuesRequest_node_version:
			f = indexer.ListUniqueBeaconBlockValuesRequest_NODE_VERSION
		case api.ListUniqueBeaconBlockValuesRequest_network:
			f = indexer.ListUniqueBeaconBlockValuesRequest_NETWORK
		case api.ListUniqueBeaconBlockValuesRequest_slot:
			f = indexer.ListUniqueBeaconBlockValuesRequest_SLOT
		case api.ListUniqueBeaconBlockValuesRequest_epoch:
			f = indexer.ListUniqueBeaconBlockValuesRequest_EPOCH
		case api.ListUniqueBeaconBlockValuesRequest_block_root:
			f = indexer.ListUniqueBeaconBlockValuesRequest_BLOCK_ROOT
		case api.ListUniqueBeaconBlockValuesRequest_beacon_implementation:
			f = indexer.ListUniqueBeaconBlockValuesRequest_BEACON_IMPLEMENTATION
		default:
			return nil, status.Error(codes.InvalidArgument, fmt.Errorf("invalid field: %s", field.String()).Error())
		}

		rq.Fields = append(rq.Fields, f)
	}

	// Call the indexer
	resp, err := i.indexer.ListUniqueBeaconBlockValues(ctx, &rq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("failed to list unique beacon block values: %w", err).Error())
	}

	// Convert the response
	response := &api.ListUniqueBeaconBlockValuesResponse{
		Node:                 resp.Node,
		Slot:                 resp.Slot,
		Epoch:                resp.Epoch,
		BlockRoot:            resp.BlockRoot,
		NodeVersion:          resp.NodeVersion,
		Network:              resp.Network,
		BeaconImplementation: resp.BeaconImplementation,
	}

	return response, nil
}

func (i *API) ListExecutionPayloadEnvelope(ctx context.Context, req *api.ListExecutionPayloadEnvelopeRequest) (*api.ListExecutionPayloadEnvelopeResponse, error) {
	pagination, err := paginationFromRequest(req.Pagination)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Errorf("invalid pagination: %w", err).Error())
	}

	rq := &indexer.ListExecutionPayloadEnvelopeRequest{
		Id:                   req.Id,
		Node:                 req.Node,
		Slot:                 req.Slot,
		Epoch:                req.Epoch,
		BlockRoot:            req.BlockRoot,
		NodeVersion:          req.NodeVersion,
		Network:              req.Network,
		Before:               req.Before,
		After:                req.After,
		BeaconImplementation: req.BeaconImplementation,

		Pagination: pagination,
	}

	resp, err := i.indexer.ListExecutionPayloadEnvelope(ctx, rq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("failed to list execution payload envelopes: %w", err).Error())
	}

	protoEnvelopes := make([]*api.ExecutionPayloadEnvelope, len(resp.ExecutionPayloadEnvelopes))
	for i, envelope := range resp.ExecutionPayloadEnvelopes {
		protoEnvelopes[i] = &api.ExecutionPayloadEnvelope{
			Id:                   envelope.Id,
			Node:                 envelope.Node,
			Slot:                 envelope.Slot,
			Epoch:                envelope.Epoch,
			BlockRoot:            envelope.BlockRoot,
			NodeVersion:          envelope.NodeVersion,
			Network:              envelope.Network,
			FetchedAt:            envelope.FetchedAt,
			BeaconImplementation: envelope.BeaconImplementation,
			ContentHash:          envelope.ContentHash,
			VerifiedAt:           envelope.VerifiedAt,
			ContentMatchedAt:     envelope.ContentMatchedAt,
		}
	}

	return &api.ListExecutionPayloadEnvelopeResponse{
		ExecutionPayloadEnvelopes: protoEnvelopes,
	}, nil
}

func (i *API) CountExecutionPayloadEnvelope(ctx context.Context, req *api.CountExecutionPayloadEnvelopeRequest) (*api.CountExecutionPayloadEnvelopeResponse, error) {
	rq := &indexer.CountExecutionPayloadEnvelopeRequest{
		Node:                 req.Node,
		Slot:                 req.Slot,
		Epoch:                req.Epoch,
		BlockRoot:            req.BlockRoot,
		NodeVersion:          req.NodeVersion,
		Network:              req.Network,
		BeaconImplementation: req.BeaconImplementation,
		Before:               req.Before,
		After:                req.After,
	}

	resp, err := i.indexer.CountExecutionPayloadEnvelope(ctx, rq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("failed to count execution payload envelopes: %w", err).Error())
	}

	return &api.CountExecutionPayloadEnvelopeResponse{
		Count: wrapperspb.UInt64(resp.GetCount().GetValue()),
	}, nil
}

func (i *API) ListUniqueExecutionPayloadEnvelopeValues(ctx context.Context, req *api.ListUniqueExecutionPayloadEnvelopeValuesRequest) (*api.ListUniqueExecutionPayloadEnvelopeValuesResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Errorf("invalid request: %w", err).Error())
	}

	// Create our "indexer" equivalent structs
	rq := indexer.ListUniqueExecutionPayloadEnvelopeValuesRequest{
		Fields:  []indexer.ListUniqueExecutionPayloadEnvelopeValuesRequest_Field{},
		Network: req.GetNetworkFilter(),
	}

	for _, field := range req.Fields {
		var f indexer.ListUniqueExecutionPayloadEnvelopeValuesRequest_Field

		switch field {
		case api.ListUniqueExecutionPayloadEnvelopeValuesRequest_node:
			f = indexer.ListUniqueExecutionPayloadEnvelopeValuesRequest_NODE
		case api.ListUniqueExecutionPayloadEnvelopeValuesRequest_node_version:
			f = indexer.ListUniqueExecutionPayloadEnvelopeValuesRequest_NODE_VERSION
		case api.ListUniqueExecutionPayloadEnvelopeValuesRequest_network:
			f = indexer.ListUniqueExecutionPayloadEnvelopeValuesRequest_NETWORK
		case api.ListUniqueExecutionPayloadEnvelopeValuesRequest_slot:
			f = indexer.ListUniqueExecutionPayloadEnvelopeValuesRequest_SLOT
		case api.ListUniqueExecutionPayloadEnvelopeValuesRequest_epoch:
			f = indexer.ListUniqueExecutionPayloadEnvelopeValuesRequest_EPOCH
		case api.ListUniqueExecutionPayloadEnvelopeValuesRequest_block_root:
			f = indexer.ListUniqueExecutionPayloadEnvelopeValuesRequest_BLOCK_ROOT
		case api.ListUniqueExecutionPayloadEnvelopeValuesRequest_beacon_implementation:
			f = indexer.ListUniqueExecutionPayloadEnvelopeValuesRequest_BEACON_IMPLEMENTATION
		default:
			return nil, status.Error(codes.InvalidArgument, fmt.Errorf("invalid field: %s", field.String()).Error())
		}

		rq.Fields = append(rq.Fields, f)
	}

	// Call the indexer
	resp, err := i.indexer.ListUniqueExecutionPayloadEnvelopeValues(ctx, &rq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("failed to list unique execution payload envelope values: %w", err).Error())
	}

	// Convert the response
	response := &api.ListUniqueExecutionPayloadEnvelopeValuesResponse{
		Node:                 resp.Node,
		Slot:                 resp.Slot,
		Epoch:                resp.Epoch,
		BlockRoot:            resp.BlockRoot,
		NodeVersion:          resp.NodeVersion,
		Network:              resp.Network,
		BeaconImplementation: resp.BeaconImplementation,
	}

	return response, nil
}

func (i *API) ListBeaconBadBlock(ctx context.Context, req *api.ListBeaconBadBlockRequest) (*api.ListBeaconBadBlockResponse, error) {
	pagination, err := paginationFromRequest(req.Pagination)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Errorf("invalid pagination: %w", err).Error())
	}

	rq := &indexer.ListBeaconBadBlockRequest{
		Id:                   req.Id,
		Node:                 req.Node,
		Slot:                 req.Slot,
		Epoch:                req.Epoch,
		BlockRoot:            req.BlockRoot,
		NodeVersion:          req.NodeVersion,
		Network:              req.Network,
		Before:               req.Before,
		After:                req.After,
		BeaconImplementation: req.BeaconImplementation,

		Pagination: pagination,
	}

	resp, err := i.indexer.ListBeaconBadBlock(ctx, rq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("failed to list beacon bad blocks: %w", err).Error())
	}

	protoBeaconBadBlocks := make([]*api.BeaconBadBlock, len(resp.BeaconBadBlocks))
	for i, block := range resp.BeaconBadBlocks {
		protoBeaconBadBlocks[i] = &api.BeaconBadBlock{
			Id:                   block.Id,
			Node:                 block.Node,
			Slot:                 block.Slot,
			Epoch:                block.Epoch,
			BlockRoot:            block.BlockRoot,
			NodeVersion:          block.NodeVersion,
			Network:              block.Network,
			FetchedAt:            block.FetchedAt,
			BeaconImplementation: block.BeaconImplementation,
			ContentHash:          block.ContentHash,
			VerifiedAt:           block.VerifiedAt,
			ContentMatchedAt:     block.ContentMatchedAt,
		}
	}

	return &api.ListBeaconBadBlockResponse{
		BeaconBadBlocks: protoBeaconBadBlocks,
	}, nil
}

func (i *API) CountBeaconBadBlock(ctx context.Context, req *api.CountBeaconBadBlockRequest) (*api.CountBeaconBadBlockResponse, error) {
	rq := &indexer.CountBeaconBadBlockRequest{
		Node:                 req.Node,
		Slot:                 req.Slot,
		Epoch:                req.Epoch,
		BlockRoot:            req.BlockRoot,
		NodeVersion:          req.NodeVersion,
		Network:              req.Network,
		BeaconImplementation: req.BeaconImplementation,
		Before:               req.Before,
		After:                req.After,
	}

	resp, err := i.indexer.CountBeaconBadBlock(ctx, rq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("failed to count beacon bad blocks: %w", err).Error())
	}

	return &api.CountBeaconBadBlockResponse{
		Count: wrapperspb.UInt64(resp.GetCount().GetValue()),
	}, nil
}

func (i *API) ListUniqueBeaconBadBlockValues(ctx context.Context, req *api.ListUniqueBeaconBadBlockValuesRequest) (*api.ListUniqueBeaconBadBlockValuesResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Errorf("invalid request: %w", err).Error())
	}

	// Create our "indexer" equivalent structs
	rq := indexer.ListUniqueBeaconBadBlockValuesRequest{
		Fields:  []indexer.ListUniqueBeaconBadBlockValuesRequest_Field{},
		Network: req.GetNetworkFilter(),
	}

	for _, field := range req.Fields {
		var f indexer.ListUniqueBeaconBadBlockValuesRequest_Field

		switch field {
		case api.ListUniqueBeaconBadBlockValuesRequest_node:
			f = indexer.ListUniqueBeaconBadBlockValuesRequest_NODE
		case api.ListUniqueBeaconBadBlockValuesRequest_node_version:
			f = indexer.ListUniqueBeaconBadBlockValuesRequest_NODE_VERSION
		case api.ListUniqueBeaconBadBlockValuesRequest_network:
			f = indexer.ListUniqueBeaconBadBlockValuesRequest_NETWORK
		case api.ListUniqueBeaconBadBlockValuesRequest_slot:
			f = indexer.ListUniqueBeaconBadBlockValuesRequest_SLOT
		case api.ListUniqueBeaconBadBlockValuesRequest_epoch:
			f = indexer.ListUniqueBeaconBadBlockValuesRequest_EPOCH
		case api.ListUniqueBeaconBadBlockValuesRequest_block_root:
			f = indexer.ListUniqueBeaconBadBlockValuesRequest_BLOCK_ROOT
		case api.ListUniqueBeaconBadBlockValuesRequest_beacon_implementation:
			f = indexer.ListUniqueBeaconBadBlockValuesRequest_BEACON_IMPLEMENTATION
		default:
			return nil, status.Error(codes.InvalidArgument, fmt.Errorf("invalid field: %s", field.String()).Error())
		}

		rq.Fields = append(rq.Fields, f)
	}

	// Call the indexer
	resp, err := i.indexer.ListUniqueBeaconBadBlockValues(ctx, &rq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("failed to list unique beacon bad block values: %w", err).Error())
	}

	// Convert the response
	response := &api.ListUniqueBeaconBadBlockValuesResponse{
		Node:                 resp.Node,
		Slot:                 resp.Slot,
		Epoch:                resp.Epoch,
		BlockRoot:            resp.BlockRoot,
		NodeVersion:          resp.NodeVersion,
		Network:              resp.Network,
		BeaconImplementation: resp.BeaconImplementation,
	}

	return response, nil
}

func (i *API) ListBeaconBadBlob(ctx context.Context, req *api.ListBeaconBadBlobRequest) (*api.ListBeaconBadBlobResponse, error) {
	pagination, err := paginationFromRequest(req.Pagination)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Errorf("invalid pagination: %w", err).Error())
	}

	rq := &indexer.ListBeaconBadBlobRequest{
		Id:                   req.Id,
		Node:                 req.Node,
		Slot:                 req.Slot,
		Epoch:                req.Epoch,
		BlockRoot:            req.BlockRoot,
		NodeVersion:          req.NodeVersion,
		Network:              req.Network,
		Before:               req.Before,
		After:                req.After,
		BeaconImplementation: req.BeaconImplementation,
		Index:                req.Index,

		Pagination: pagination,
	}

	resp, err := i.indexer.ListBeaconBadBlob(ctx, rq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("failed to list beacon bad blobs: %w", err).Error())
	}

	protoBeaconBadBlobs := make([]*api.BeaconBadBlob, len(resp.BeaconBadBlobs))
	for i, blob := range resp.BeaconBadBlobs {
		protoBeaconBadBlobs[i] = &api.BeaconBadBlob{
			Id:                   blob.Id,
			Node:                 blob.Node,
			Slot:                 blob.Slot,
			Epoch:                blob.Epoch,
			BlockRoot:            blob.BlockRoot,
			NodeVersion:          blob.NodeVersion,
			Network:              blob.Network,
			FetchedAt:            blob.FetchedAt,
			BeaconImplementation: blob.BeaconImplementation,
			Index:                blob.Index,
			ContentHash:          blob.ContentHash,
			VerifiedAt:           blob.VerifiedAt,
			ContentMatchedAt:     blob.ContentMatchedAt,
		}
	}

	return &api.ListBeaconBadBlobResponse{
		BeaconBadBlobs: protoBeaconBadBlobs,
	}, nil
}

func (i *API) CountBeaconBadBlob(ctx context.Context, req *api.CountBeaconBadBlobRequest) (*api.CountBeaconBadBlobResponse, error) {
	rq := &indexer.CountBeaconBadBlobRequest{
		Node:                 req.Node,
		Slot:                 req.Slot,
		Epoch:                req.Epoch,
		BlockRoot:            req.BlockRoot,
		NodeVersion:          req.NodeVersion,
		Network:              req.Network,
		BeaconImplementation: req.BeaconImplementation,
		Index:                req.Index,
		Before:               req.Before,
		After:                req.After,
	}

	resp, err := i.indexer.CountBeaconBadBlob(ctx, rq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("failed to count beacon bad blobs: %w", err).Error())
	}

	return &api.CountBeaconBadBlobResponse{
		Count: wrapperspb.UInt64(resp.GetCount().GetValue()),
	}, nil
}

func (i *API) ListUniqueBeaconBadBlobValues(ctx context.Context, req *api.ListUniqueBeaconBadBlobValuesRequest) (*api.ListUniqueBeaconBadBlobValuesResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Errorf("invalid request: %w", err).Error())
	}

	// Create our "indexer" equivalent structs
	rq := indexer.ListUniqueBeaconBadBlobValuesRequest{
		Fields:  []indexer.ListUniqueBeaconBadBlobValuesRequest_Field{},
		Network: req.GetNetworkFilter(),
	}

	for _, field := range req.Fields {
		var f indexer.ListUniqueBeaconBadBlobValuesRequest_Field

		switch field {
		case api.ListUniqueBeaconBadBlobValuesRequest_node:
			f = indexer.ListUniqueBeaconBadBlobValuesRequest_NODE
		case api.ListUniqueBeaconBadBlobValuesRequest_node_version:
			f = indexer.ListUniqueBeaconBadBlobValuesRequest_NODE_VERSION
		case api.ListUniqueBeaconBadBlobValuesRequest_network:
			f = indexer.ListUniqueBeaconBadBlobValuesRequest_NETWORK
		case api.ListUniqueBeaconBadBlobValuesRequest_slot:
			f = indexer.ListUniqueBeaconBadBlobValuesRequest_SLOT
		case api.ListUniqueBeaconBadBlobValuesRequest_epoch:
			f = indexer.ListUniqueBeaconBadBlobValuesRequest_EPOCH
		case api.ListUniqueBeaconBadBlobValuesRequest_block_root:
			f = indexer.ListUniqueBeaconBadBlobValuesRequest_BLOCK_ROOT
		case api.ListUniqueBeaconBadBlobValuesRequest_beacon_implementation:
			f = indexer.ListUniqueBeaconBadBlobValuesRequest_BEACON_IMPLEMENTATION
		case api.ListUniqueBeaconBadBlobValuesRequest_index:
			f = indexer.ListUniqueBeaconBadBlobValuesRequest_INDEX
		default:
			return nil, status.Error(codes.InvalidArgument, fmt.Errorf("invalid field: %s", field.String()).Error())
		}

		rq.Fields = append(rq.Fields, f)
	}

	// Call the indexer
	resp, err := i.indexer.ListUniqueBeaconBadBlobValues(ctx, &rq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("failed to list unique beacon bad blob values: %w", err).Error())
	}

	// Convert the response
	response := &api.ListUniqueBeaconBadBlobValuesResponse{
		Node:                 resp.Node,
		Slot:                 resp.Slot,
		Epoch:                resp.Epoch,
		BlockRoot:            resp.BlockRoot,
		NodeVersion:          resp.NodeVersion,
		Network:              resp.Network,
		BeaconImplementation: resp.BeaconImplementation,
		Index:                resp.Index,
	}

	return response, nil
}

func (i *API) ListExecutionBlockTrace(ctx context.Context, req *api.ListExecutionBlockTraceRequest) (*api.ListExecutionBlockTraceResponse, error) {
	pagination, err := paginationFromRequest(req.Pagination)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Errorf("invalid pagination: %w", err).Error())
	}

	rq := &indexer.ListExecutionBlockTraceRequest{
		Node:        req.Node,
		BlockNumber: req.BlockNumber,
		BlockHash:   req.BlockHash,
		Network:     req.Network,
		Before:      req.Before,
		After:       req.After,
		Id:          req.Id,

		Pagination: pagination,
	}

	resp, err := i.indexer.ListExecutionBlockTrace(ctx, rq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("failed to list execution block traces: %w", err).Error())
	}

	protoExecutionBlockTraces := make([]*api.ExecutionBlockTrace, len(resp.ExecutionBlockTraces))
	for i, trace := range resp.ExecutionBlockTraces {
		protoExecutionBlockTraces[i] = &api.ExecutionBlockTrace{
			Id:                      trace.Id,
			Node:                    trace.Node,
			FetchedAt:               trace.FetchedAt,
			BlockHash:               trace.BlockHash,
			BlockNumber:             trace.BlockNumber,
			Network:                 trace.Network,
			ExecutionImplementation: trace.ExecutionImplementation,
			NodeVersion:             trace.NodeVersion,
			ContentHash:             trace.ContentHash,
			VerifiedAt:              trace.VerifiedAt,
			ContentMatchedAt:        trace.ContentMatchedAt,
		}
	}

	return &api.ListExecutionBlockTraceResponse{
		ExecutionBlockTraces: protoExecutionBlockTraces,
	}, nil
}

func (i *API) CountExecutionBlockTrace(ctx context.Context, req *api.CountExecutionBlockTraceRequest) (*api.CountExecutionBlockTraceResponse, error) {
	rq := &indexer.CountExecutionBlockTraceRequest{
		Node:                    req.Node,
		BlockNumber:             req.BlockNumber,
		BlockHash:               req.BlockHash,
		Network:                 req.Network,
		Before:                  req.Before,
		After:                   req.After,
		ExecutionImplementation: req.ExecutionImplementation,
		NodeVersion:             req.NodeVersion,
	}

	resp, err := i.indexer.CountExecutionBlockTrace(ctx, rq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("failed to count execution block traces: %w", err).Error())
	}

	return &api.CountExecutionBlockTraceResponse{
		Count: wrapperspb.UInt64(resp.GetCount().GetValue()),
	}, nil
}

func (i *API) ListUniqueExecutionBlockTraceValues(ctx context.Context, req *api.ListUniqueExecutionBlockTraceValuesRequest) (*api.ListUniqueExecutionBlockTraceValuesResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Errorf("invalid request: %w", err).Error())
	}

	// Create our "indexer" equivalent structs
	rq := indexer.ListUniqueExecutionBlockTraceValuesRequest{
		Fields:  []indexer.ListUniqueExecutionBlockTraceValuesRequest_Field{},
		Network: req.GetNetworkFilter(),
	}

	for _, field := range req.Fields {
		var f indexer.ListUniqueExecutionBlockTraceValuesRequest_Field

		switch field {
		case api.ListUniqueExecutionBlockTraceValuesRequest_node:
			f = indexer.ListUniqueExecutionBlockTraceValuesRequest_NODE
		case api.ListUniqueExecutionBlockTraceValuesRequest_block_hash:
			f = indexer.ListUniqueExecutionBlockTraceValuesRequest_BLOCK_HASH
		case api.ListUniqueExecutionBlockTraceValuesRequest_block_number:
			f = indexer.ListUniqueExecutionBlockTraceValuesRequest_BLOCK_NUMBER
		case api.ListUniqueExecutionBlockTraceValuesRequest_network:
			f = indexer.ListUniqueExecutionBlockTraceValuesRequest_NETWORK
		case api.ListUniqueExecutionBlockTraceValuesRequest_execution_implementation:
			f = indexer.ListUniqueExecutionBlockTraceValuesRequest_EXECUTION_IMPLEMENTATION
		case api.ListUniqueExecutionBlockTraceValuesRequest_node_version:
			f = indexer.ListUniqueExecutionBlockTraceValuesRequest_NODE_VERSION
		default:
			return nil, status.Error(codes.InvalidArgument, fmt.Errorf("invalid field: %s", field.String()).Error())
		}

		rq.Fields = append(rq.Fields, f)
	}

	// Call the indexer
	resp, err := i.indexer.ListUniqueExecutionBlockTraceValues(ctx, &rq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("failed to list unique execution block trace values: %w", err).Error())
	}

	// Convert the response
	response := &api.ListUniqueExecutionBlockTraceValuesResponse{
		Node:                    resp.Node,
		BlockHash:               resp.BlockHash,
		BlockNumber:             resp.BlockNumber,
		Network:                 resp.Network,
		ExecutionImplementation: resp.ExecutionImplementation,
		NodeVersion:             resp.NodeVersion,
	}

	return response, nil
}

func (i *API) ListExecutionBadBlock(ctx context.Context, req *api.ListExecutionBadBlockRequest) (*api.ListExecutionBadBlockResponse, error) {
	pagination, err := paginationFromRequest(req.Pagination)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Errorf("invalid pagination: %w", err).Error())
	}

	rq := &indexer.ListExecutionBadBlockRequest{
		Node:           req.Node,
		BlockNumber:    req.BlockNumber,
		BlockHash:      req.BlockHash,
		Network:        req.Network,
		Before:         req.Before,
		After:          req.After,
		Id:             req.Id,
		BlockExtraData: req.BlockExtraData,

		Pagination: pagination,
	}

	resp, err := i.indexer.ListExecutionBadBlock(ctx, rq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("failed to list execution block traces: %w", err).Error())
	}

	protoExecutionBadBlocks := make([]*api.ExecutionBadBlock, len(resp.ExecutionBadBlocks))
	for i, trace := range resp.ExecutionBadBlocks {
		protoExecutionBadBlocks[i] = &api.ExecutionBadBlock{
			Id:                      trace.Id,
			Node:                    trace.Node,
			FetchedAt:               trace.FetchedAt,
			BlockHash:               trace.BlockHash,
			BlockNumber:             trace.BlockNumber,
			Network:                 trace.Network,
			ExecutionImplementation: trace.ExecutionImplementation,
			NodeVersion:             trace.NodeVersion,
			BlockExtraData:          trace.BlockExtraData,
			ContentHash:             trace.ContentHash,
			VerifiedAt:              trace.VerifiedAt,
			ContentMatchedAt:        trace.ContentMatchedAt,
		}
	}

	return &api.ListExecutionBadBlockResponse{
		ExecutionBadBlocks: protoExecutionBadBlocks,
	}, nil
}

func (i *API) CountExecutionBadBlock(ctx context.Context, req *api.CountExecutionBadBlockRequest) (*api.CountExecutionBadBlockResponse, error) {
	rq := &indexer.CountExecutionBadBlockRequest{
		Node:                    req.Node,
		BlockNumber:             req.BlockNumber,
		BlockHash:               req.BlockHash,
		Network:                 req.Network,
		Before:                  req.Before,
		After:                   req.After,
		ExecutionImplementation: req.ExecutionImplementation,
		NodeVersion:             req.NodeVersion,
		BlockExtraData:          req.BlockExtraData,
	}

	resp, err := i.indexer.CountExecutionBadBlock(ctx, rq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("failed to count execution block traces: %w", err).Error())
	}

	return &api.CountExecutionBadBlockResponse{
		Count: wrapperspb.UInt64(resp.GetCount().GetValue()),
	}, nil
}

func (i *API) ListUniqueExecutionBadBlockValues(ctx context.Context, req *api.ListUniqueExecutionBadBlockValuesRequest) (*api.ListUniqueExecutionBadBlockValuesResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Errorf("invalid request: %w", err).Error())
	}

	// Create our "indexer" equivalent structs
	rq := indexer.ListUniqueExecutionBadBlockValuesRequest{
		Fields:  []indexer.ListUniqueExecutionBadBlockValuesRequest_Field{},
		Network: req.GetNetworkFilter(),
	}

	for _, field := range req.Fields {
		var f indexer.ListUniqueExecutionBadBlockValuesRequest_Field

		switch field {
		case api.ListUniqueExecutionBadBlockValuesRequest_node:
			f = indexer.ListUniqueExecutionBadBlockValuesRequest_NODE
		case api.ListUniqueExecutionBadBlockValuesRequest_block_hash:
			f = indexer.ListUniqueExecutionBadBlockValuesRequest_BLOCK_HASH
		case api.ListUniqueExecutionBadBlockValuesRequest_block_number:
			f = indexer.ListUniqueExecutionBadBlockValuesRequest_BLOCK_NUMBER
		case api.ListUniqueExecutionBadBlockValuesRequest_network:
			f = indexer.ListUniqueExecutionBadBlockValuesRequest_NETWORK
		case api.ListUniqueExecutionBadBlockValuesRequest_execution_implementation:
			f = indexer.ListUniqueExecutionBadBlockValuesRequest_EXECUTION_IMPLEMENTATION
		case api.ListUniqueExecutionBadBlockValuesRequest_node_version:
			f = indexer.ListUniqueExecutionBadBlockValuesRequest_NODE_VERSION
		case api.ListUniqueExecutionBadBlockValuesRequest_block_extra_data:
			f = indexer.ListUniqueExecutionBadBlockValuesRequest_BLOCK_EXTRA_DATA
		default:
			return nil, status.Error(codes.InvalidArgument, fmt.Errorf("invalid field: %s", field.String()).Error())
		}

		rq.Fields = append(rq.Fields, f)
	}

	// Call the indexer
	resp, err := i.indexer.ListUniqueExecutionBadBlockValues(ctx, &rq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("failed to list unique execution block trace values: %w", err).Error())
	}

	// Convert the response
	response := &api.ListUniqueExecutionBadBlockValuesResponse{
		Node:                    resp.Node,
		BlockHash:               resp.BlockHash,
		BlockNumber:             resp.BlockNumber,
		Network:                 resp.Network,
		ExecutionImplementation: resp.ExecutionImplementation,
		NodeVersion:             resp.NodeVersion,
		BlockExtraData:          resp.BlockExtraData,
	}

	return response, nil
}

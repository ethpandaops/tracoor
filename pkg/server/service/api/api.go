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

func (e *API) Start(ctx context.Context, grpcServer *grpc.Server) error {
	e.log.Info("Starting module")

	// Connect to the indexer
	conn, err := grpc.Dial(e.grpcConn, e.grpcOpts...)
	if err != nil {
		return fmt.Errorf("fail to dial: %v", err)
	}

	e.indexer = indexer.NewIndexerClient(conn)

	api.RegisterAPIServer(grpcServer, e)

	return nil
}

func (e *API) Stop(ctx context.Context) error {
	e.log.Info("Stopping module")

	// Wait for all requests to finish?

	return nil
}

func (i *API) ListBeaconState(ctx context.Context, req *api.ListBeaconStateRequest) (*api.ListBeaconStateResponse, error) {
	pagination := &indexer.PaginationCursor{
		Limit:   100,
		Offset:  0,
		OrderBy: "fetched_at DESC",
	}

	if req.Pagination != nil {
		pagination = &indexer.PaginationCursor{
			Limit:   req.Pagination.Limit,
			Offset:  req.Pagination.Offset,
			OrderBy: req.Pagination.OrderBy,
		}
	}

	rq := &indexer.ListBeaconStateRequest{
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
		Fields: []indexer.ListUniqueBeaconStateValuesRequest_Field{},
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

func (i *API) ListExecutionBlockTrace(ctx context.Context, req *api.ListExecutionBlockTraceRequest) (*api.ListExecutionBlockTraceResponse, error) {
	pagination := &indexer.PaginationCursor{
		Limit:   100,
		Offset:  0,
		OrderBy: "fetched_at DESC",
	}

	if req.Pagination != nil {
		pagination = &indexer.PaginationCursor{
			Limit:   req.Pagination.Limit,
			Offset:  req.Pagination.Offset,
			OrderBy: req.Pagination.OrderBy,
		}
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
		Fields: []indexer.ListUniqueExecutionBlockTraceValuesRequest_Field{},
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
	pagination := &indexer.PaginationCursor{
		Limit:   100,
		Offset:  0,
		OrderBy: "fetched_at DESC",
	}

	if req.Pagination != nil {
		pagination = &indexer.PaginationCursor{
			Limit:   req.Pagination.Limit,
			Offset:  req.Pagination.Offset,
			OrderBy: req.Pagination.OrderBy,
		}
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
		Fields: []indexer.ListUniqueExecutionBadBlockValuesRequest_Field{},
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

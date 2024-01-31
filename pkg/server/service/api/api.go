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
		Node:        req.Node,
		Slot:        req.Slot,
		Epoch:       req.Epoch,
		StateRoot:   req.StateRoot,
		NodeVersion: req.NodeVersion,
		Network:     req.Network,
		Before:      req.Before,
		After:       req.After,

		Pagination: pagination,
	}

	resp, err := i.indexer.ListBeaconState(ctx, rq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Errorf("failed to list beacon states: %w", err).Error())
	}

	protoBeaconStates := make([]*api.BeaconState, len(resp.BeaconStates))
	for i, state := range resp.BeaconStates {
		protoBeaconStates[i] = &api.BeaconState{
			Id:          state.Id,
			Node:        state.Node,
			Slot:        state.Slot,
			Epoch:       state.Epoch,
			StateRoot:   state.StateRoot,
			NodeVersion: state.NodeVersion,
			Network:     state.Network,
			FetchedAt:   state.FetchedAt,
		}
	}

	return &api.ListBeaconStateResponse{
		BeaconStates: protoBeaconStates,
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
		Node:        resp.Node,
		Slot:        resp.Slot,
		Epoch:       resp.Epoch,
		StateRoot:   resp.StateRoot,
		NodeVersion: resp.NodeVersion,
		Network:     resp.Network,
	}

	return response, nil

}

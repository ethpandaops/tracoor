package indexer

import (
	"context"

	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	ServiceType = "indexer"
)

type Indexer struct {
	indexer.IndexerServer

	log logrus.FieldLogger

	store store.Store

	db *persistence.Indexer
}

func NewIndexer(ctx context.Context, log logrus.FieldLogger, conf *Config, db *persistence.Indexer, st store.Store) (*Indexer, error) {
	e := &Indexer{
		log:   log.WithField("server/module", ServiceType),
		db:    db,
		store: st,
	}

	return e, nil
}

func (e *Indexer) Start(ctx context.Context, grpcServer *grpc.Server) error {
	e.log.Info("Starting module")

	indexer.RegisterIndexerServer(grpcServer, e)

	return nil
}

func (e *Indexer) Stop(ctx context.Context) error {
	e.log.Info("Stopping module")

	// Wait for all requests to finish?

	return nil
}

func (e *Indexer) CreateBeaconState(ctx context.Context, req *indexer.CreateBeaconStateRequest) (*indexer.CreateBeaconStateResponse, error) {
	e.log.Debug("Received CreateBeaconState request")

	// Ensure request is valid
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Check the store for the state
	exists, err := e.store.Exists(ctx, req.GetLocation().GetValue())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if exists {
		// Check if the state is already indexed
		filter := &persistence.BeaconStateFilter{}

		filter.AddNetwork(req.GetNetwork().GetValue())
		filter.AddSlot(req.GetSlot().GetValue())
		filter.AddStateRoot(req.GetStateRoot().GetValue())
		filter.AddNode(req.GetNode().GetValue())

		states, err := e.db.ListBeaconState(ctx, filter, &persistence.PaginationCursor{Limit: 1, Offset: 0})
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		if len(states) > 0 {
			return nil, status.Error(codes.AlreadyExists, "beacon state already indexed")
		}
	}

	// Create the state
	state := &indexer.BeaconState{
		Id:          wrapperspb.String(uuid.New().String()),
		Node:        req.GetNode(),
		Network:     req.GetNetwork(),
		Slot:        req.GetSlot(),
		Epoch:       req.GetEpoch(),
		StateRoot:   req.GetStateRoot(),
		NodeVersion: req.GetNodeVersion(),
		Location:    req.GetLocation(),
		FetchedAt:   req.GetFetchedAt(),
	}

	logFields := logrus.Fields{
		"node":         req.GetNode().GetValue(),
		"network":      req.GetNetwork().GetValue(),
		"slot":         req.GetSlot().GetValue(),
		"epoch":        req.GetEpoch().GetValue(),
		"state_root":   req.GetStateRoot().GetValue(),
		"node_version": req.GetNodeVersion().GetValue(),
		"location":     req.GetLocation().GetValue(),
		"fetched_at":   req.GetFetchedAt().AsTime(),
	}

	if err := e.db.InsertBeaconState(ctx, ProtoBeaconStateToDBBeaconState(state)); err != nil {
		e.log.WithError(err).WithFields(logFields).Error("Failed to index state")

		return nil, status.Error(codes.Internal, "failed to index state")
	}

	e.log.WithFields(logFields).WithField("id", state.GetId().GetValue()).Debug("Indexed beacon state")

	return &indexer.CreateBeaconStateResponse{
		Id: state.GetId(),
	}, nil
}

func (i *Indexer) ListBeaconState(ctx context.Context, req *indexer.ListBeaconStateRequest) (*indexer.ListBeaconStateResponse, error) {
	filter := &persistence.BeaconStateFilter{}

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

	pagination := &persistence.PaginationCursor{
		Limit:   100,
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

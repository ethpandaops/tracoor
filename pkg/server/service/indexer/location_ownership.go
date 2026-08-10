package indexer

import (
	"context"

	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// These checks stop a create request from claiming a location that is
// already associated with a different record. A location is expected to be
// unique to the record that first uploaded a blob there; if two records with
// different identities are allowed to share one location, deleting either
// record (for example, once it ages out of retention) deletes the shared
// blob out from under the other one, even though that other record is still
// active and was never meant to be touched.

func (i *Indexer) checkBeaconStateLocationOwnership(ctx context.Context, req *indexer.CreateBeaconStateRequest) error {
	filter := &persistence.BeaconStateFilter{}
	filter.AddLocation(req.GetLocation().GetValue())

	existing, err := i.db.ListBeaconState(ctx, filter, &persistence.PaginationCursor{Limit: 1, Offset: 0})
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	if len(existing) == 0 {
		return nil
	}

	record := existing[0]

	//nolint:gosec // slot is well within int64 range
	sameRecord := record.Node == req.GetNode().GetValue() &&
		record.Network == req.GetNetwork().GetValue() &&
		record.Slot == int64(req.GetSlot().GetValue()) &&
		record.StateRoot == req.GetStateRoot().GetValue()

	if !sameRecord {
		return status.Error(codes.AlreadyExists, "location is already associated with a different beacon state")
	}

	return nil
}

func (i *Indexer) checkBeaconBlockLocationOwnership(ctx context.Context, req *indexer.CreateBeaconBlockRequest) error {
	filter := &persistence.BeaconBlockFilter{}
	filter.AddLocation(req.GetLocation().GetValue())

	existing, err := i.db.ListBeaconBlock(ctx, filter, &persistence.PaginationCursor{Limit: 1, Offset: 0})
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	if len(existing) == 0 {
		return nil
	}

	record := existing[0]

	//nolint:gosec // slot is well within int64 range
	sameRecord := record.Node == req.GetNode().GetValue() &&
		record.Network == req.GetNetwork().GetValue() &&
		record.Slot == int64(req.GetSlot().GetValue()) &&
		record.BlockRoot == req.GetBlockRoot().GetValue()

	if !sameRecord {
		return status.Error(codes.AlreadyExists, "location is already associated with a different beacon block")
	}

	return nil
}

func (i *Indexer) checkBeaconBadBlockLocationOwnership(ctx context.Context, req *indexer.CreateBeaconBadBlockRequest) error {
	filter := &persistence.BeaconBadBlockFilter{}
	filter.AddLocation(req.GetLocation().GetValue())

	existing, err := i.db.ListBeaconBadBlock(ctx, filter, &persistence.PaginationCursor{Limit: 1, Offset: 0})
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	if len(existing) == 0 {
		return nil
	}

	record := existing[0]

	//nolint:gosec // slot is well within int64 range
	sameRecord := record.Node == req.GetNode().GetValue() &&
		record.Network == req.GetNetwork().GetValue() &&
		record.Slot == int64(req.GetSlot().GetValue()) &&
		record.BlockRoot == req.GetBlockRoot().GetValue()

	if !sameRecord {
		return status.Error(codes.AlreadyExists, "location is already associated with a different beacon bad block")
	}

	return nil
}

func (i *Indexer) checkBeaconBadBlobLocationOwnership(ctx context.Context, req *indexer.CreateBeaconBadBlobRequest) error {
	filter := &persistence.BeaconBadBlobFilter{}
	filter.AddLocation(req.GetLocation().GetValue())

	existing, err := i.db.ListBeaconBadBlob(ctx, filter, &persistence.PaginationCursor{Limit: 1, Offset: 0})
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	if len(existing) == 0 {
		return nil
	}

	record := existing[0]

	//nolint:gosec // slot and index are well within int64 range
	sameRecord := record.Node == req.GetNode().GetValue() &&
		record.Network == req.GetNetwork().GetValue() &&
		record.Slot == int64(req.GetSlot().GetValue()) &&
		record.BlockRoot == req.GetBlockRoot().GetValue() &&
		record.Index == int64(req.GetIndex().GetValue())

	if !sameRecord {
		return status.Error(codes.AlreadyExists, "location is already associated with a different beacon bad blob")
	}

	return nil
}

func (i *Indexer) checkExecutionBlockTraceLocationOwnership(ctx context.Context, req *indexer.CreateExecutionBlockTraceRequest) error {
	filter := &persistence.ExecutionBlockTraceFilter{}
	filter.AddLocation(req.GetLocation().GetValue())

	existing, err := i.db.ListExecutionBlockTrace(ctx, filter, &persistence.PaginationCursor{Limit: 1, Offset: 0})
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	if len(existing) == 0 {
		return nil
	}

	record := existing[0]

	sameRecord := record.Node == req.GetNode().GetValue() &&
		record.Network == req.GetNetwork().GetValue() &&
		record.BlockHash == req.GetBlockHash().GetValue()

	if !sameRecord {
		return status.Error(codes.AlreadyExists, "location is already associated with a different execution block trace")
	}

	return nil
}

func (i *Indexer) checkExecutionBadBlockLocationOwnership(ctx context.Context, req *indexer.CreateExecutionBadBlockRequest) error {
	filter := &persistence.ExecutionBadBlockFilter{}
	filter.AddLocation(req.GetLocation().GetValue())

	existing, err := i.db.ListExecutionBadBlock(ctx, filter, &persistence.PaginationCursor{Limit: 1, Offset: 0})
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	if len(existing) == 0 {
		return nil
	}

	record := existing[0]

	sameRecord := record.Node == req.GetNode().GetValue() &&
		record.Network == req.GetNetwork().GetValue() &&
		record.BlockHash == req.GetBlockHash().GetValue()

	if !sameRecord {
		return status.Error(codes.AlreadyExists, "location is already associated with a different execution bad block")
	}

	return nil
}

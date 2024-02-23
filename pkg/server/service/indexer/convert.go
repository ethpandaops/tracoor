package indexer

import (
	"database/sql"

	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func ProtoBeaconStateToDBBeaconState(bs *indexer.BeaconState) *persistence.BeaconState {
	return &persistence.BeaconState{
		ID:                   bs.GetId().GetValue(),
		Node:                 bs.GetNode().GetValue(),
		Slot:                 int64(bs.GetSlot().GetValue()),
		Epoch:                int64(bs.GetEpoch().GetValue()),
		StateRoot:            bs.GetStateRoot().GetValue(),
		FetchedAt:            bs.GetFetchedAt().AsTime(),
		NodeVersion:          bs.GetNodeVersion().GetValue(),
		Location:             bs.GetLocation().GetValue(),
		Network:              bs.GetNetwork().GetValue(),
		BeaconImplementation: bs.GetBeaconImplementation().GetValue(),
	}
}

func DBBeaconStateToProtoBeaconState(bs *persistence.BeaconState) *indexer.BeaconState {
	return &indexer.BeaconState{
		Id:                   &wrapperspb.StringValue{Value: bs.ID},
		Node:                 &wrapperspb.StringValue{Value: bs.Node},
		Slot:                 &wrapperspb.UInt64Value{Value: uint64(bs.Slot)},
		Epoch:                &wrapperspb.UInt64Value{Value: uint64(bs.Epoch)},
		StateRoot:            &wrapperspb.StringValue{Value: bs.StateRoot},
		FetchedAt:            timestamppb.New(bs.FetchedAt),
		NodeVersion:          &wrapperspb.StringValue{Value: bs.NodeVersion},
		Location:             &wrapperspb.StringValue{Value: bs.Location},
		Network:              &wrapperspb.StringValue{Value: bs.Network},
		BeaconImplementation: &wrapperspb.StringValue{Value: bs.BeaconImplementation},
	}
}

func ProtoExecutionBlockTraceToDBExecutionBlockTrace(eb *indexer.ExecutionBlockTrace) *persistence.ExecutionBlockTrace {
	return &persistence.ExecutionBlockTrace{
		BlockHash:               eb.GetBlockHash().GetValue(),
		BlockNumber:             eb.GetBlockNumber().GetValue(),
		Node:                    eb.GetNode().GetValue(),
		NodeVersion:             eb.GetNodeVersion().GetValue(),
		FetchedAt:               eb.GetFetchedAt().AsTime(),
		Location:                eb.GetLocation().GetValue(),
		Network:                 eb.GetNetwork().GetValue(),
		ExecutionImplementation: eb.GetExecutionImplementation().GetValue(),
		ID:                      eb.GetId().GetValue(),
	}
}

func DBExecutionBlockTraceToProtoExecutionBlockTrace(eb *persistence.ExecutionBlockTrace) *indexer.ExecutionBlockTrace {
	return &indexer.ExecutionBlockTrace{
		BlockHash:               &wrapperspb.StringValue{Value: eb.BlockHash},
		BlockNumber:             &wrapperspb.Int64Value{Value: eb.BlockNumber},
		Node:                    &wrapperspb.StringValue{Value: eb.Node},
		NodeVersion:             &wrapperspb.StringValue{Value: eb.NodeVersion},
		FetchedAt:               timestamppb.New(eb.FetchedAt),
		Location:                &wrapperspb.StringValue{Value: eb.Location},
		Network:                 &wrapperspb.StringValue{Value: eb.Network},
		ExecutionImplementation: &wrapperspb.StringValue{Value: eb.ExecutionImplementation},
		Id:                      &wrapperspb.StringValue{Value: eb.ID},
	}
}

func ProtoExecutionBadBlockToDBExecutionBadBlock(eb *indexer.ExecutionBadBlock) *persistence.ExecutionBadBlock {
	return &persistence.ExecutionBadBlock{
		BlockHash:               eb.GetBlockHash().GetValue(),
		BlockNumber:             sql.NullInt64{Int64: eb.GetBlockNumber().GetValue(), Valid: true},
		Node:                    eb.GetNode().GetValue(),
		NodeVersion:             eb.GetNodeVersion().GetValue(),
		FetchedAt:               eb.GetFetchedAt().AsTime(),
		Location:                eb.GetLocation().GetValue(),
		Network:                 eb.GetNetwork().GetValue(),
		ExecutionImplementation: eb.GetExecutionImplementation().GetValue(),
		ID:                      eb.GetId().GetValue(),
		BlockExtraData:          sql.NullString{String: eb.GetBlockExtraData().GetValue(), Valid: true},
	}
}

func DBExecutionBadBlockToProtoExecutionBadBlock(eb *persistence.ExecutionBadBlock) *indexer.ExecutionBadBlock {
	return &indexer.ExecutionBadBlock{
		BlockHash:               &wrapperspb.StringValue{Value: eb.BlockHash},
		BlockNumber:             &wrapperspb.Int64Value{Value: eb.BlockNumber.Int64},
		Node:                    &wrapperspb.StringValue{Value: eb.Node},
		NodeVersion:             &wrapperspb.StringValue{Value: eb.NodeVersion},
		FetchedAt:               timestamppb.New(eb.FetchedAt),
		Location:                &wrapperspb.StringValue{Value: eb.Location},
		Network:                 &wrapperspb.StringValue{Value: eb.Network},
		ExecutionImplementation: &wrapperspb.StringValue{Value: eb.ExecutionImplementation},
		Id:                      &wrapperspb.StringValue{Value: eb.ID},
		BlockExtraData:          &wrapperspb.StringValue{Value: eb.BlockExtraData.String},
	}
}

func ProtoPaginationCursorToDBPaginationCursor(pc *indexer.PaginationCursor) *persistence.PaginationCursor {
	return &persistence.PaginationCursor{
		Offset:  int(pc.GetOffset()),
		Limit:   int(pc.GetLimit()),
		OrderBy: pc.GetOrderBy(),
	}
}

func DBPaginationCursorToProtoPaginationCursor(pc *persistence.PaginationCursor) *indexer.PaginationCursor {
	return &indexer.PaginationCursor{
		Offset:  int32(pc.Offset),
		Limit:   int32(pc.Limit),
		OrderBy: pc.OrderBy,
	}
}

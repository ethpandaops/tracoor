package api

import (
	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/api"
	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func ProtoBeaconStateToDBBeaconState(bs *api.BeaconState) *persistence.BeaconState {
	return &persistence.BeaconState{
		ID:          bs.GetId().GetValue(),
		Node:        bs.GetNode().GetValue(),
		Slot:        int64(bs.GetSlot().GetValue()),
		Epoch:       int64(bs.GetEpoch().GetValue()),
		StateRoot:   bs.GetStateRoot().GetValue(),
		FetchedAt:   bs.GetFetchedAt().AsTime(),
		NodeVersion: bs.GetNodeVersion().GetValue(),
		Network:     bs.GetNetwork().GetValue(),
	}
}

func DBBeaconStateToProtoBeaconState(bs *persistence.BeaconState) *api.BeaconState {
	return &api.BeaconState{
		Id:          &wrapperspb.StringValue{Value: bs.ID},
		Node:        &wrapperspb.StringValue{Value: bs.Node},
		Slot:        &wrapperspb.UInt64Value{Value: uint64(bs.Slot)},
		Epoch:       &wrapperspb.UInt64Value{Value: uint64(bs.Epoch)},
		StateRoot:   &wrapperspb.StringValue{Value: bs.StateRoot},
		FetchedAt:   timestamppb.New(bs.FetchedAt),
		NodeVersion: &wrapperspb.StringValue{Value: bs.NodeVersion},
		Network:     &wrapperspb.StringValue{Value: bs.Network},
	}
}

func ProtoPaginationCursorToDBPaginationCursor(pc *api.PaginationCursor) *persistence.PaginationCursor {
	return &persistence.PaginationCursor{
		Offset:  int(pc.GetOffset()),
		Limit:   int(pc.GetLimit()),
		OrderBy: pc.GetOrderBy(),
	}
}

func DBPaginationCursorToProtoPaginationCursor(pc *persistence.PaginationCursor) *api.PaginationCursor {
	return &api.PaginationCursor{
		Offset:  int32(pc.Offset),
		Limit:   int32(pc.Limit),
		OrderBy: pc.OrderBy,
	}
}

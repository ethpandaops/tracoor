package indexer

import (
	"database/sql"
	"time"

	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// protoTimestampToTime keeps "never happened" distinct from the zero time.
func protoTimestampToTime(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}

	t := ts.AsTime()

	return &t
}

// timeToProtoTimestamp keeps "never happened" distinct from the zero time.
func timeToProtoTimestamp(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}

	return timestamppb.New(*t)
}

func ProtoBeaconStateToDBBeaconState(bs *indexer.BeaconState) *persistence.BeaconState {
	return &persistence.BeaconState{
		ID:   bs.GetId().GetValue(),
		Node: bs.GetNode().GetValue(),
		//nolint:gosec // not worried about int64 overflow here
		Slot: int64(bs.GetSlot().GetValue()),
		//nolint:gosec // not worried about int64 overflow here
		Epoch:                int64(bs.GetEpoch().GetValue()),
		StateRoot:            bs.GetStateRoot().GetValue(),
		FetchedAt:            bs.GetFetchedAt().AsTime(),
		NodeVersion:          bs.GetNodeVersion().GetValue(),
		Location:             bs.GetLocation().GetValue(),
		ContentEncoding:      bs.GetContentEncoding().GetValue(),
		Network:              bs.GetNetwork().GetValue(),
		BeaconImplementation: bs.GetBeaconImplementation().GetValue(),
		ContentHash:          bs.GetContentHash().GetValue(),
		VerifiedAt:           protoTimestampToTime(bs.GetVerifiedAt()),
		ContentMatchedAt:     protoTimestampToTime(bs.GetContentMatchedAt()),
	}
}

func DBBeaconStateToProtoBeaconState(bs *persistence.BeaconState) *indexer.BeaconState {
	return &indexer.BeaconState{
		Id:   &wrapperspb.StringValue{Value: bs.ID},
		Node: &wrapperspb.StringValue{Value: bs.Node},
		//nolint:gosec // not worried about int64 overflow here
		Slot: &wrapperspb.UInt64Value{Value: uint64(bs.Slot)},
		//nolint:gosec // not worried about int64 overflow here
		Epoch:                &wrapperspb.UInt64Value{Value: uint64(bs.Epoch)},
		StateRoot:            &wrapperspb.StringValue{Value: bs.StateRoot},
		FetchedAt:            timestamppb.New(bs.FetchedAt),
		NodeVersion:          &wrapperspb.StringValue{Value: bs.NodeVersion},
		Location:             &wrapperspb.StringValue{Value: bs.Location},
		ContentEncoding:      &wrapperspb.StringValue{Value: bs.ContentEncoding},
		Network:              &wrapperspb.StringValue{Value: bs.Network},
		BeaconImplementation: &wrapperspb.StringValue{Value: bs.BeaconImplementation},
		ContentHash:          &wrapperspb.StringValue{Value: bs.ContentHash},
		VerifiedAt:           timeToProtoTimestamp(bs.VerifiedAt),
		ContentMatchedAt:     timeToProtoTimestamp(bs.ContentMatchedAt),
	}
}

func ProtoBeaconBlockToDBBeaconBlock(bs *indexer.BeaconBlock) *persistence.BeaconBlock {
	return &persistence.BeaconBlock{
		ID:   bs.GetId().GetValue(),
		Node: bs.GetNode().GetValue(),
		//nolint:gosec // not worried about int64 overflow here
		Slot: int64(bs.GetSlot().GetValue()),
		//nolint:gosec // not worried about int64 overflow here
		Epoch:                int64(bs.GetEpoch().GetValue()),
		BlockRoot:            bs.GetBlockRoot().GetValue(),
		FetchedAt:            bs.GetFetchedAt().AsTime(),
		NodeVersion:          bs.GetNodeVersion().GetValue(),
		Location:             bs.GetLocation().GetValue(),
		ContentEncoding:      bs.GetContentEncoding().GetValue(),
		Network:              bs.GetNetwork().GetValue(),
		BeaconImplementation: bs.GetBeaconImplementation().GetValue(),
		ContentHash:          bs.GetContentHash().GetValue(),
		VerifiedAt:           protoTimestampToTime(bs.GetVerifiedAt()),
		ContentMatchedAt:     protoTimestampToTime(bs.GetContentMatchedAt()),
	}
}

func DBBeaconBlockToProtoBeaconBlock(bs *persistence.BeaconBlock) *indexer.BeaconBlock {
	return &indexer.BeaconBlock{
		Id:   &wrapperspb.StringValue{Value: bs.ID},
		Node: &wrapperspb.StringValue{Value: bs.Node},
		//nolint:gosec // not worried about int64 overflow here
		Slot: &wrapperspb.UInt64Value{Value: uint64(bs.Slot)},
		//nolint:gosec // not worried about int64 overflow here
		Epoch:                &wrapperspb.UInt64Value{Value: uint64(bs.Epoch)},
		BlockRoot:            &wrapperspb.StringValue{Value: bs.BlockRoot},
		FetchedAt:            timestamppb.New(bs.FetchedAt),
		NodeVersion:          &wrapperspb.StringValue{Value: bs.NodeVersion},
		Location:             &wrapperspb.StringValue{Value: bs.Location},
		Network:              &wrapperspb.StringValue{Value: bs.Network},
		ContentEncoding:      &wrapperspb.StringValue{Value: bs.ContentEncoding},
		BeaconImplementation: &wrapperspb.StringValue{Value: bs.BeaconImplementation},
		ContentHash:          &wrapperspb.StringValue{Value: bs.ContentHash},
		VerifiedAt:           timeToProtoTimestamp(bs.VerifiedAt),
		ContentMatchedAt:     timeToProtoTimestamp(bs.ContentMatchedAt),
	}
}

func ProtoExecutionPayloadEnvelopeToDBExecutionPayloadEnvelope(bs *indexer.ExecutionPayloadEnvelope) *persistence.ExecutionPayloadEnvelope {
	return &persistence.ExecutionPayloadEnvelope{
		ID:   bs.GetId().GetValue(),
		Node: bs.GetNode().GetValue(),
		//nolint:gosec // not worried about int64 overflow here
		Slot: int64(bs.GetSlot().GetValue()),
		//nolint:gosec // not worried about int64 overflow here
		Epoch:                int64(bs.GetEpoch().GetValue()),
		BlockRoot:            bs.GetBlockRoot().GetValue(),
		FetchedAt:            bs.GetFetchedAt().AsTime(),
		NodeVersion:          bs.GetNodeVersion().GetValue(),
		Location:             bs.GetLocation().GetValue(),
		ContentEncoding:      bs.GetContentEncoding().GetValue(),
		Network:              bs.GetNetwork().GetValue(),
		BeaconImplementation: bs.GetBeaconImplementation().GetValue(),
		ContentHash:          bs.GetContentHash().GetValue(),
		VerifiedAt:           protoTimestampToTime(bs.GetVerifiedAt()),
		ContentMatchedAt:     protoTimestampToTime(bs.GetContentMatchedAt()),
	}
}

func DBExecutionPayloadEnvelopeToProtoExecutionPayloadEnvelope(bs *persistence.ExecutionPayloadEnvelope) *indexer.ExecutionPayloadEnvelope {
	return &indexer.ExecutionPayloadEnvelope{
		Id:   &wrapperspb.StringValue{Value: bs.ID},
		Node: &wrapperspb.StringValue{Value: bs.Node},
		//nolint:gosec // not worried about int64 overflow here
		Slot: &wrapperspb.UInt64Value{Value: uint64(bs.Slot)},
		//nolint:gosec // not worried about int64 overflow here
		Epoch:                &wrapperspb.UInt64Value{Value: uint64(bs.Epoch)},
		BlockRoot:            &wrapperspb.StringValue{Value: bs.BlockRoot},
		FetchedAt:            timestamppb.New(bs.FetchedAt),
		NodeVersion:          &wrapperspb.StringValue{Value: bs.NodeVersion},
		Location:             &wrapperspb.StringValue{Value: bs.Location},
		Network:              &wrapperspb.StringValue{Value: bs.Network},
		ContentEncoding:      &wrapperspb.StringValue{Value: bs.ContentEncoding},
		BeaconImplementation: &wrapperspb.StringValue{Value: bs.BeaconImplementation},
		ContentHash:          &wrapperspb.StringValue{Value: bs.ContentHash},
		VerifiedAt:           timeToProtoTimestamp(bs.VerifiedAt),
		ContentMatchedAt:     timeToProtoTimestamp(bs.ContentMatchedAt),
	}
}

func ProtoBeaconBadBlockToDBBeaconBadBlock(bs *indexer.BeaconBadBlock) *persistence.BeaconBadBlock {
	return &persistence.BeaconBadBlock{
		ID:   bs.GetId().GetValue(),
		Node: bs.GetNode().GetValue(),
		//nolint:gosec // not worried about int64 overflow here
		Slot: int64(bs.GetSlot().GetValue()),
		//nolint:gosec // not worried about int64 overflow here
		Epoch:                int64(bs.GetEpoch().GetValue()),
		BlockRoot:            bs.GetBlockRoot().GetValue(),
		FetchedAt:            bs.GetFetchedAt().AsTime(),
		NodeVersion:          bs.GetNodeVersion().GetValue(),
		Location:             bs.GetLocation().GetValue(),
		ContentEncoding:      bs.GetContentEncoding().GetValue(),
		Network:              bs.GetNetwork().GetValue(),
		BeaconImplementation: bs.GetBeaconImplementation().GetValue(),
		ContentHash:          bs.GetContentHash().GetValue(),
		VerifiedAt:           protoTimestampToTime(bs.GetVerifiedAt()),
		ContentMatchedAt:     protoTimestampToTime(bs.GetContentMatchedAt()),
	}
}

func DBBeaconBadBlockToProtoBeaconBadBlock(bs *persistence.BeaconBadBlock) *indexer.BeaconBadBlock {
	return &indexer.BeaconBadBlock{
		Id:   &wrapperspb.StringValue{Value: bs.ID},
		Node: &wrapperspb.StringValue{Value: bs.Node},
		//nolint:gosec // not worried about int64 overflow here
		Slot: &wrapperspb.UInt64Value{Value: uint64(bs.Slot)},
		//nolint:gosec // not worried about int64 overflow here
		Epoch:                &wrapperspb.UInt64Value{Value: uint64(bs.Epoch)},
		BlockRoot:            &wrapperspb.StringValue{Value: bs.BlockRoot},
		FetchedAt:            timestamppb.New(bs.FetchedAt),
		NodeVersion:          &wrapperspb.StringValue{Value: bs.NodeVersion},
		Location:             &wrapperspb.StringValue{Value: bs.Location},
		Network:              &wrapperspb.StringValue{Value: bs.Network},
		ContentEncoding:      &wrapperspb.StringValue{Value: bs.ContentEncoding},
		BeaconImplementation: &wrapperspb.StringValue{Value: bs.BeaconImplementation},
		ContentHash:          &wrapperspb.StringValue{Value: bs.ContentHash},
		VerifiedAt:           timeToProtoTimestamp(bs.VerifiedAt),
		ContentMatchedAt:     timeToProtoTimestamp(bs.ContentMatchedAt),
	}
}

func ProtoBeaconBadBlobToDBBeaconBadBlob(bs *indexer.BeaconBadBlob) *persistence.BeaconBadBlob {
	return &persistence.BeaconBadBlob{
		ID:   bs.GetId().GetValue(),
		Node: bs.GetNode().GetValue(),
		//nolint:gosec // not worried about int64 overflow here
		Slot: int64(bs.GetSlot().GetValue()),
		//nolint:gosec // not worried about int64 overflow here
		Epoch:                int64(bs.GetEpoch().GetValue()),
		BlockRoot:            bs.GetBlockRoot().GetValue(),
		FetchedAt:            bs.GetFetchedAt().AsTime(),
		NodeVersion:          bs.GetNodeVersion().GetValue(),
		Location:             bs.GetLocation().GetValue(),
		Network:              bs.GetNetwork().GetValue(),
		ContentEncoding:      bs.GetContentEncoding().GetValue(),
		BeaconImplementation: bs.GetBeaconImplementation().GetValue(),
		//nolint:gosec // not worried about int64 overflow here
		Index:            int64(bs.GetIndex().GetValue()),
		ContentHash:      bs.GetContentHash().GetValue(),
		VerifiedAt:       protoTimestampToTime(bs.GetVerifiedAt()),
		ContentMatchedAt: protoTimestampToTime(bs.GetContentMatchedAt()),
	}
}

func DBBeaconBadBlobToProtoBeaconBadBlob(bs *persistence.BeaconBadBlob) *indexer.BeaconBadBlob {
	return &indexer.BeaconBadBlob{
		Id:   &wrapperspb.StringValue{Value: bs.ID},
		Node: &wrapperspb.StringValue{Value: bs.Node},
		//nolint:gosec // not worried about int64 overflow here
		Slot: &wrapperspb.UInt64Value{Value: uint64(bs.Slot)},
		//nolint:gosec // not worried about int64 overflow here
		Epoch:                &wrapperspb.UInt64Value{Value: uint64(bs.Epoch)},
		BlockRoot:            &wrapperspb.StringValue{Value: bs.BlockRoot},
		FetchedAt:            timestamppb.New(bs.FetchedAt),
		NodeVersion:          &wrapperspb.StringValue{Value: bs.NodeVersion},
		Location:             &wrapperspb.StringValue{Value: bs.Location},
		Network:              &wrapperspb.StringValue{Value: bs.Network},
		ContentEncoding:      &wrapperspb.StringValue{Value: bs.ContentEncoding},
		BeaconImplementation: &wrapperspb.StringValue{Value: bs.BeaconImplementation},
		//nolint:gosec // not worried about int64 overflow here
		Index:            &wrapperspb.UInt64Value{Value: uint64(bs.Index)},
		ContentHash:      &wrapperspb.StringValue{Value: bs.ContentHash},
		VerifiedAt:       timeToProtoTimestamp(bs.VerifiedAt),
		ContentMatchedAt: timeToProtoTimestamp(bs.ContentMatchedAt),
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
		ContentEncoding:         eb.GetContentEncoding().GetValue(),
		ExecutionImplementation: eb.GetExecutionImplementation().GetValue(),
		ID:                      eb.GetId().GetValue(),
		ContentHash:             eb.GetContentHash().GetValue(),
		VerifiedAt:              protoTimestampToTime(eb.GetVerifiedAt()),
		ContentMatchedAt:        protoTimestampToTime(eb.GetContentMatchedAt()),
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
		ContentEncoding:         &wrapperspb.StringValue{Value: eb.ContentEncoding},
		Id:                      &wrapperspb.StringValue{Value: eb.ID},
		ContentHash:             &wrapperspb.StringValue{Value: eb.ContentHash},
		VerifiedAt:              timeToProtoTimestamp(eb.VerifiedAt),
		ContentMatchedAt:        timeToProtoTimestamp(eb.ContentMatchedAt),
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
		ContentEncoding:         eb.GetContentEncoding().GetValue(),
		ExecutionImplementation: eb.GetExecutionImplementation().GetValue(),
		ID:                      eb.GetId().GetValue(),
		BlockExtraData:          sql.NullString{String: eb.GetBlockExtraData().GetValue(), Valid: true},
		ContentHash:             eb.GetContentHash().GetValue(),
		VerifiedAt:              protoTimestampToTime(eb.GetVerifiedAt()),
		ContentMatchedAt:        protoTimestampToTime(eb.GetContentMatchedAt()),
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
		ContentEncoding:         &wrapperspb.StringValue{Value: eb.ContentEncoding},
		Id:                      &wrapperspb.StringValue{Value: eb.ID},
		BlockExtraData:          &wrapperspb.StringValue{Value: eb.BlockExtraData.String},
		ContentHash:             &wrapperspb.StringValue{Value: eb.ContentHash},
		VerifiedAt:              timeToProtoTimestamp(eb.VerifiedAt),
		ContentMatchedAt:        timeToProtoTimestamp(eb.ContentMatchedAt),
	}
}

func DBBlobToProtoBlob(b *persistence.Blob) *indexer.Blob {
	return &indexer.Blob{
		Kind:            &wrapperspb.StringValue{Value: b.Kind},
		Network:         &wrapperspb.StringValue{Value: b.Network},
		DedupKey:        &wrapperspb.StringValue{Value: b.DedupKey},
		ContentHash:     &wrapperspb.StringValue{Value: b.ContentHash},
		Location:        &wrapperspb.StringValue{Value: b.Location},
		ContentEncoding: &wrapperspb.StringValue{Value: b.ContentEncoding},
		RawSize:         &wrapperspb.Int64Value{Value: b.RawSize},
		CompressedSize:  &wrapperspb.Int64Value{Value: b.CompressedSize},
		RefCount:        &wrapperspb.Int64Value{Value: b.RefCount},
		State:           &wrapperspb.StringValue{Value: b.State},
		Generation:      &wrapperspb.Int64Value{Value: b.Generation},
		CreatedAt:       timestamppb.New(b.CreatedAt),
	}
}

func ProtoPayloadDivergenceToDBPayloadDivergence(d *indexer.PayloadDivergence) *persistence.PayloadDivergence {
	return &persistence.PayloadDivergence{
		ID:           d.GetId().GetValue(),
		ObservedAt:   d.GetObservedAt().AsTime(),
		Network:      d.GetNetwork().GetValue(),
		Kind:         d.GetKind().GetValue(),
		Node:         d.GetNode().GetValue(),
		DedupKey:     d.GetDedupKey().GetValue(),
		ExpectedHash: d.GetExpectedHash().GetValue(),
		ActualHash:   d.GetActualHash().GetValue(),
		//nolint:gosec // not worried about int64 overflow here
		Slot:       int64(d.GetSlot().GetValue()),
		Identifier: d.GetIdentifier().GetValue(),
		Attempt:    d.GetAttempt().GetValue(),
		Severity:   d.GetSeverity().GetValue(),
		Location:   d.GetLocation().GetValue(),
	}
}

func DBPayloadDivergenceToProtoPayloadDivergence(d *persistence.PayloadDivergence) *indexer.PayloadDivergence {
	return &indexer.PayloadDivergence{
		Id:           &wrapperspb.StringValue{Value: d.ID},
		ObservedAt:   timestamppb.New(d.ObservedAt),
		Network:      &wrapperspb.StringValue{Value: d.Network},
		Kind:         &wrapperspb.StringValue{Value: d.Kind},
		Node:         &wrapperspb.StringValue{Value: d.Node},
		DedupKey:     &wrapperspb.StringValue{Value: d.DedupKey},
		ExpectedHash: &wrapperspb.StringValue{Value: d.ExpectedHash},
		ActualHash:   &wrapperspb.StringValue{Value: d.ActualHash},
		//nolint:gosec // not worried about int64 overflow here
		Slot:       &wrapperspb.UInt64Value{Value: uint64(d.Slot)},
		Identifier: &wrapperspb.StringValue{Value: d.Identifier},
		Attempt:    &wrapperspb.Int32Value{Value: d.Attempt},
		Severity:   &wrapperspb.StringValue{Value: d.Severity},
		Location:   &wrapperspb.StringValue{Value: d.Location},
	}
}

func ProtoPaginationCursorToDBPaginationCursor(pc *indexer.PaginationCursor) (*persistence.PaginationCursor, error) {
	cursor := &persistence.PaginationCursor{
		Offset:  int(pc.GetOffset()),
		Limit:   int(pc.GetLimit()),
		OrderBy: pc.GetOrderBy(),
	}

	if err := cursor.Validate(); err != nil {
		return nil, err
	}

	return cursor, nil
}

func DBPaginationCursorToProtoPaginationCursor(pc *persistence.PaginationCursor) *indexer.PaginationCursor {
	return &indexer.PaginationCursor{
		//nolint:gosec // not worried about int32 overflow here
		Offset: int32(pc.Offset),
		//nolint:gosec // not worried about int32 overflow here
		Limit:   int32(pc.Limit),
		OrderBy: pc.OrderBy,
	}
}

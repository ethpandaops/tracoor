package agent

import (
	"fmt"
	"path"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
)

// A deduplicated artifact is addressed by what determines its bytes rather than
// by which node served them, so its directory carries no node segment. The file
// name inside it is suffixed with the content hash, which is what keeps two
// nodes that disagree under one key from ever writing the same object.

func BeaconStateDirectory(
	network string,
	slot phase0.Slot,
) string {
	return path.Join(
		"beacon_states",
		network,
		"slots",
		fmt.Sprintf("%d", slot),
	)
}

func BeaconBlockDirectory(
	network string,
	slot phase0.Slot,
) string {
	return path.Join(
		"beacon_blocks",
		network,
		"slots",
		fmt.Sprintf("%d", slot),
	)
}

func ExecutionPayloadEnvelopeDirectory(
	network string,
	slot phase0.Slot,
) string {
	return path.Join(
		"execution_payload_envelopes",
		network,
		"slots",
		fmt.Sprintf("%d", slot),
	)
}

func ExecutionBlockTraceDirectory(
	network string,
	blockNumber uint64,
) string {
	return path.Join(
		"execution_block_traces",
		network,
		"blocks",
		fmt.Sprintf("%d", blockNumber),
	)
}

func CreateBeaconBadBlockFileName(
	node string,
	network string,
	slot phase0.Slot,
	blockRoot string,
) string {
	return path.Join(
		"beacon_bad_blocks",
		network,
		"slots",
		fmt.Sprintf("%d", slot),
		node,
		blockRoot,
	)
}

func CreateBeaconBadBlobFileName(
	node string,
	network string,
	slot phase0.Slot,
	blockRoot string,
	index uint64,
) string {
	return path.Join(
		"beacon_bad_blobs",
		network,
		"slots",
		fmt.Sprintf("%d", slot),
		node,
		blockRoot,
		fmt.Sprintf("%d", index),
	)
}

func CreateExecutionBadBlockFileName(
	node string,
	network string,
	blockHash string,
) string {
	return path.Join(
		"execution_bad_blocks",
		network,
		node,
		blockHash,
	)
}

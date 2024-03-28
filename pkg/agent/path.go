package agent

import (
	"fmt"
	"path"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

func CreateBeaconStateFileName(
	node string,
	network string,
	slot phase0.Slot,
	stateRoot string,
) string {
	return path.Join(
		"beacon_states",
		network,
		"slots",
		fmt.Sprintf("%d", slot),
		node,
		stateRoot,
	)
}

func CreateBeaconBlockFileName(
	node string,
	network string,
	slot phase0.Slot,
	blockRoot string,
) string {
	return path.Join(
		"beacon_blocks",
		network,
		"slots",
		fmt.Sprintf("%d", slot),
		node,
		blockRoot,
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

func CreateExecutionBlockTraceFileName(
	node string,
	network string,
	blockNumber uint64,
	blockHash string,
) string {
	return path.Join(
		"execution_block_traces",
		network,
		"blocks",
		fmt.Sprintf("%d", blockNumber),
		node,
		blockHash,
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

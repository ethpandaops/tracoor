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

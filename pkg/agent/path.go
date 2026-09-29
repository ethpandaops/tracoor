package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
)

// traceVariantLength is how much of the dedup key's digest a trace file name
// carries. It separates the handful of client and parameter combinations that
// can exist for one block, not a global namespace.
const traceVariantLength = 16

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

// ExecutionBlockTraceIdentity names a trace inside its directory. A trace is
// only reproducible from the client that produced it and the parameters it was
// asked for, all of which its dedup key carries and its block hash does not.
// Both belong in the location: two dedup keys that happen to produce identical
// bytes must still land on two objects, because collecting either one would
// otherwise empty the other's rows.
func ExecutionBlockTraceIdentity(blockHash, dedupKey string) string {
	digest := sha256.Sum256([]byte(dedupKey))

	return fmt.Sprintf("%s-%s", blockHash, hex.EncodeToString(digest[:])[:traceVariantLength])
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

package promotion

import (
	"fmt"
	"strings"
)

// Corpus layout (schema v1). Self-describing; consumable with nothing but an
// S3 client:
//
//	v1/states/<sha256-of-raw-ssz>.ssz                        uncompressed, content-addressed, deduplicated
//	v1/captures/<network>/<fork>/<capture_id>/input.ssz      the SignedBeaconBlock, raw
//	v1/captures/<network>/<fork>/<capture_id>/manifest.json
const (
	corpusStatePrefix   = "v1/states"
	corpusCapturePrefix = "v1/captures"

	captureBlockName    = "input.ssz"
	captureManifestName = "manifest.json"

	captureRootChars = 12
	networkGVRChars  = 8
)

func statePath(sha string) string {
	return fmt.Sprintf("%s/%s.ssz", corpusStatePrefix, sha)
}

func capturePath(network, fork, captureID, name string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", corpusCapturePrefix, network, fork, captureID, name)
}

// captureID is <slot zero-padded to 9>-<block_root hex[0:12]>: lexical order
// equals slot order, and the root suffix lets both branches of a reorg
// coexist.
func captureID(slot uint64, blockRoot string) string {
	root := strings.TrimPrefix(blockRoot, "0x")
	if len(root) > captureRootChars {
		root = root[:captureRootChars]
	}

	return fmt.Sprintf("%09d-%s", slot, root)
}

// networkID qualifies the agent-reported network name with the first 4 bytes
// of the genesis validators root.
func networkID(configName, gvrHex string) string {
	gvr := strings.TrimPrefix(gvrHex, "0x")
	if len(gvr) > networkGVRChars {
		gvr = gvr[:networkGVRChars]
	}

	return fmt.Sprintf("%s-%s", configName, gvr)
}

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
//	v1/captures/<network>/<fork>/<capture_id>/envelope.ssz   the SignedExecutionPayloadEnvelope, raw (gloas+, when revealed)
//	v1/captures/<network>/<fork>/<capture_id>/manifest.json
const (
	corpusStatePrefix   = "v1/states"
	corpusCapturePrefix = "v1/captures"

	captureBlockName    = "input.ssz"
	captureEnvelopeName = "envelope.ssz"
	captureManifestName = "manifest.json"

	captureRootChars = 12
	networkGVRChars  = 8

	// unknownComponent stands in for a path component we cannot name
	// honestly: an unsanitisable network name, or a fork no supported
	// version decodes.
	unknownComponent = "unknown"
)

// safePathComponent keeps a single path segment a single path segment. The
// network name reaches us from an agent, and the filesystem store resolves
// "../" the way filesystems do; the corpus layout depends on these being
// opaque names, not paths. Everything outside the layout's own alphabet
// becomes an underscore.
func safePathComponent(s string) string {
	if s == "" {
		return unknownComponent
	}

	out := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '-', r == '_', r == '.':
			return r
		default:
			return '_'
		}
	}, s)

	// A component of dots only is "here" or "up one" to a filesystem.
	if strings.Trim(out, ".") == "" {
		return unknownComponent
	}

	return out
}

func statePath(sha string) string {
	return fmt.Sprintf("%s/%s.ssz", corpusStatePrefix, safePathComponent(sha))
}

func capturePath(network, fork, captureID, name string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s",
		corpusCapturePrefix,
		safePathComponent(network),
		safePathComponent(fork),
		safePathComponent(captureID),
		name,
	)
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
// of the genesis validators root. The name is sanitised here rather than only
// at the path layer so that the manifest's network.name and the key it lives
// under can never disagree.
func networkID(configName, gvrHex string) string {
	gvr := strings.TrimPrefix(gvrHex, "0x")
	if len(gvr) > networkGVRChars {
		gvr = gvr[:networkGVRChars]
	}

	return fmt.Sprintf("%s-%s", safePathComponent(configName), gvr)
}

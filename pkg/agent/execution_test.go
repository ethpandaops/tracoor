package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecutionBlockTraceLocationsSeparateDistinctDedupKeys(t *testing.T) {
	const (
		blockHash = "0xdeadbeef"
		hash      = "cafebabecafebabecafebabecafebabecafebabecafebabecafebabecafebabe"
	)

	// Two clients can return byte-identical traces for one block. Their rows
	// are separate, so their objects have to be too: collecting one must never
	// empty the other.
	locationFor := func(implementation, version string, disableMemory, disableStack, disableStorage bool) string {
		key := executionBlockTraceDedupKey(42, blockHash, implementation, version, disableMemory, disableStack, disableStorage)

		target := &dedupTarget{
			directory: ExecutionBlockTraceDirectory("testnet", 42),
			identity:  ExecutionBlockTraceIdentity(blockHash, key),
			extension: ".json",
		}

		return target.finalLocation(hash)
	}

	base := locationFor("geth", "geth/v1.14.0", false, false, false)

	tests := []struct {
		name     string
		location string
	}{
		{
			name:     "a different client version",
			location: locationFor("geth", "geth/v1.14.1", false, false, false),
		},
		{
			name:     "a different implementation",
			location: locationFor("reth", "geth/v1.14.0", false, false, false),
		},
		{
			name:     "a different memory flag",
			location: locationFor("geth", "geth/v1.14.0", true, false, false),
		},
		{
			name:     "a different stack flag",
			location: locationFor("geth", "geth/v1.14.0", false, true, false),
		},
		{
			name:     "a different storage flag",
			location: locationFor("geth", "geth/v1.14.0", false, false, true),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.NotEqual(t, base, test.location, "two dedup keys must never share one object")
		})
	}

	require.Equal(t, base, locationFor("geth", "geth/v1.14.0", false, false, false), "the same key always resolves to the same object")
	require.True(t, strings.HasSuffix(base, "-"+hash[:contentHashSuffixLength]+".json"), "the content hash still separates disagreeing payloads")
	require.Contains(t, base, blockHash, "the block a trace belongs to stays readable in its path")
}

func TestConsensusLocationsAreUnaffectedByTheTraceIdentity(t *testing.T) {
	hash := strings.Repeat("b", 64)

	for _, target := range []*dedupTarget{
		{directory: BeaconStateDirectory("testnet", 12), identity: "0xstateroot", extension: ".ssz"},
		{directory: BeaconBlockDirectory("testnet", 12), identity: "0xblockroot", extension: ".ssz"},
		{directory: ExecutionPayloadEnvelopeDirectory("testnet", 12), identity: "0xblockroot", extension: ".ssz"},
	} {
		require.Equal(t,
			target.directory+"/"+target.identity+"-"+hash[:contentHashSuffixLength]+target.extension,
			target.finalLocation(hash),
			"an artifact whose dedup key is fully described by its identity keeps its location",
		)
	}
}

func TestEmptyJSONResultIsNotATrace(t *testing.T) {
	tests := []struct {
		name  string
		data  string
		empty bool
	}{
		{name: "an explicit null", data: "null", empty: true},
		{name: "a null with whitespace around it", data: " \n null\t", empty: true},
		{name: "an absent result", data: "", empty: true},
		{name: "whitespace only", data: "   ", empty: true},
		{name: "an empty object is still an answer", data: "{}", empty: false},
		{name: "an empty array is still an answer", data: "[]", empty: false},
		{name: "a real trace", data: `[{"txHash":"0x1"}]`, empty: false},
		{name: "a string that merely starts with null", data: `"nullish"`, empty: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.empty, isEmptyJSONResult([]byte(test.data)))
		})
	}
}

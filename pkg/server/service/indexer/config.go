package indexer

import "github.com/ethpandaops/beacon/pkg/human"

type RetentionConfig struct {
	BeaconStates              human.Duration `yaml:"beaconStates" default:"30m"`
	BeaconBlocks              human.Duration `yaml:"beaconBlocks" default:"30m"`
	ExecutionPayloadEnvelopes human.Duration `yaml:"executionPayloadEnvelopes" default:"30m"`
	BeaconBadBlocks           human.Duration `yaml:"beaconBadBlocks" default:"312480m"` // 6 months
	BeaconBadBlobs            human.Duration `yaml:"beaconBadBlobs" default:"312480m"`  // 6 months
	ExecutionBlockTraces      human.Duration `yaml:"executionBlockTraces" default:"30m"`
	ExecutionBadBlocks        human.Duration `yaml:"executionBadBlocks" default:"312480m"` // 6 months
	// PayloadDivergences bounds the divergence log. It is deliberately long: the record of a
	// node serving the wrong bytes is the finding, and it outlives the payload it describes.
	PayloadDivergences human.Duration `yaml:"payloadDivergences" default:"312480m"` // 6 months
}

type Config struct {
	Retention      RetentionConfig      `yaml:"retention"`
	PermanentStore PermanentStoreConfig `yaml:"permanentStore"`

	// BlobGCGracePeriod is the minimum age of a zero-reference payload before it may be
	// collected. It must exceed the longest plausible fetch-plus-upload plus the agents'
	// payload-lookup cache TTL, so an agent mid-hash cannot link to a payload being deleted.
	BlobGCGracePeriod human.Duration `yaml:"blobGcGracePeriod" default:"10m"`

	ReadCache ReadCacheConfig `yaml:"readCache"`
}

// ReadCacheConfig bounds the cache in front of the filter-value and count reads.
type ReadCacheConfig struct {
	// Disabled sends every read straight to the database. It is a flag rather than a zero
	// TTL because the server applies defaults again after loading the config, which turns an
	// explicit 0 back into the default.
	Disabled bool `yaml:"disabled"`
	// TTL stays under the UI's 6s count staleTime, so a cached total is never older than one
	// the UI would have kept on its own.
	TTL        human.Duration `yaml:"ttl" default:"5s"`
	MaxEntries int            `yaml:"maxEntries" default:"4096"`
}

func (c *Config) Validate() error {
	return nil
}

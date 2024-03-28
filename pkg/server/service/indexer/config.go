package indexer

import "github.com/ethpandaops/beacon/pkg/human"

type RetentionConfig struct {
	BeaconStates         human.Duration `yaml:"beaconStates" default:"30m"`
	BeaconBlocks         human.Duration `yaml:"beaconBlocks" default:"30m"`
	BeaconBadBlocks      human.Duration `yaml:"beaconBadBlocks" default:"312480m"` // 6 months
	BeaconBadBlobs       human.Duration `yaml:"beaconBadBlobs" default:"312480m"`  // 6 months
	ExecutionBlockTraces human.Duration `yaml:"executionBlockTraces" default:"30m"`
	ExecutionBadBlocks   human.Duration `yaml:"executionBadBlocks" default:"312480m"` // 6 months
}

type Config struct {
	Retention RetentionConfig `yaml:"retention"`
}

func (c *Config) Validate() error {
	return nil
}

package indexer

import "github.com/ethpandaops/beacon/pkg/human"

type RetentionConfig struct {
	BeaconStates         human.Duration `yaml:"beaconStates" default:"30m"`
	ExecutionBlockTraces human.Duration `yaml:"executionBlockTraces" default:"30m"`
}

type Config struct {
	Retention RetentionConfig `yaml:"retention"`
}

func (c *Config) Validate() error {
	return nil
}

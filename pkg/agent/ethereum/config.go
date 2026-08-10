package ethereum

import (
	"fmt"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/tracoor/pkg/agent/ethereum/beacon"
	"github.com/ethpandaops/tracoor/pkg/agent/ethereum/execution"
)

type Config struct {
	// Execution configuration
	Execution *execution.Config `yaml:"execution"`
	// Beacon configuration
	Beacon *beacon.Config `yaml:"beacon"`
	// OverrideNetworkName is the name of the network to use for the agent.
	// If not set, the network name will be retrieved from the beacon node.
	OverrideNetworkName string `yaml:"overrideNetworkName"  default:""`
	// Features configuration
	Features Features `yaml:"features"`
	// SyncToleranceSlots is the age of the block in slots that is tolerated before it is not fetched.
	// This is to prevent fetching blocks that are too far behind the head.
	SyncToleranceSlots phase0.Slot `yaml:"syncToleranceSlots" default:"64"`

	// BeaconStateAgeThresholdEpochs is the number of epochs that a beacon state is allowed to be old.
	// This should be used to prevent fetching states that are too far behind the head,
	// which can cause the agent to get stuck as old states might not be available in
	// the beacon node cache.
	BeaconStateAgeThresholdEpochs uint64 `yaml:"beaconStateAgeThresholdEpochs" default:"1"`

	// ExecutionBlockTraceAgeThresholdBlocks is how far behind the execution head
	// a block may be before its trace is no longer worth requesting. Execution
	// clients retain only enough state to re-execute a bounded number of recent
	// blocks, so a trace beyond that window is a guaranteed failure. The default
	// follows geth's `reexec` default of 128 blocks.
	ExecutionBlockTraceAgeThresholdBlocks uint64 `yaml:"executionBlockTraceAgeThresholdBlocks" default:"128"`

	// MaxConcurrentFetches bounds how many fetches of any kind - states, blocks,
	// envelopes, block traces, bad blocks and bad blobs - may be in flight at once
	// across every agent in the process. Each holds its whole response in memory,
	// so this caps peak usage independently of how many agents are configured.
	// A single shared budget is deliberate: per-path budgets bound each path but
	// not the total. Applied process-wide by whichever agent starts first.
	MaxConcurrentFetches int `yaml:"maxConcurrentFetches" default:"10"`
}

const defaultExecutionBlockTraceAgeThresholdBlocks = 128

func (c *Config) GetExecutionBlockTraceAgeThresholdBlocks() uint64 {
	if c.ExecutionBlockTraceAgeThresholdBlocks == 0 {
		return defaultExecutionBlockTraceAgeThresholdBlocks
	}

	return c.ExecutionBlockTraceAgeThresholdBlocks
}

func (c *Config) GetMaxConcurrentFetches() int {
	if c.MaxConcurrentFetches <= 0 {
		return 0
	}

	return c.MaxConcurrentFetches
}

func (c *Config) Validate() error {
	if err := c.Execution.Validate(); err != nil {
		return fmt.Errorf("invalid execution configuration: %w", err)
	}

	if err := c.Beacon.Validate(); err != nil {
		return fmt.Errorf("invalid beacon configuration: %w", err)
	}

	if err := c.Features.Validate(); err != nil {
		return fmt.Errorf("invalid features configuration: %w", err)
	}

	return nil
}

// Features contains feature flags for the agent.
type Features struct {
	FetchBeaconState              *bool `yaml:"fetchBeaconState" default:"true"`
	FetchBeaconBlock              *bool `yaml:"fetchBeaconBlock" default:"true"`
	FetchExecutionPayloadEnvelope *bool `yaml:"fetchExecutionPayloadEnvelope" default:"true"`
	FetchBeaconBadBlock           *bool `yaml:"fetchBeaconBadBlock" default:"true"`
	FetchBeaconBadBlob            *bool `yaml:"fetchBeaconBadBlob" default:"true"`
	FetchExecutionBlockTrace      *bool `yaml:"fetchExecutionBlockTrace" default:"true"`
	FetchExecutionBadBlock        *bool `yaml:"fetchExecutionBadBlock" default:"true"`
}

func (f Features) Validate() error {
	return nil
}

func (f Features) GetFetchBeaconState() bool {
	if f.FetchBeaconState == nil {
		return true // default value
	}

	return *f.FetchBeaconState
}

func (f Features) GetFetchBeaconBlock() bool {
	if f.FetchBeaconBlock == nil {
		return true // default value
	}

	return *f.FetchBeaconBlock
}

func (f Features) GetFetchExecutionPayloadEnvelope() bool {
	if f.FetchExecutionPayloadEnvelope == nil {
		return true // default value
	}

	return *f.FetchExecutionPayloadEnvelope
}

func (f Features) GetFetchBeaconBadBlock() bool {
	if f.FetchBeaconBadBlock == nil {
		return true // default value
	}

	return *f.FetchBeaconBadBlock
}

func (f Features) GetFetchBeaconBadBlob() bool {
	if f.FetchBeaconBadBlob == nil {
		return true // default value
	}

	return *f.FetchBeaconBadBlob
}

func (f Features) GetFetchExecutionBlockTrace() bool {
	if f.FetchExecutionBlockTrace == nil {
		return true // default value
	}

	return *f.FetchExecutionBlockTrace
}

func (f Features) GetFetchExecutionBadBlock() bool {
	if f.FetchExecutionBadBlock == nil {
		return true // default value
	}

	return *f.FetchExecutionBadBlock
}

// EnabledFlags returns a list of enabled feature flags.
func (f Features) EnabledFlags() []string {
	var enabled []string
	if f.GetFetchBeaconState() {
		enabled = append(enabled, "FetchBeaconState")
	}

	if f.GetFetchBeaconBlock() {
		enabled = append(enabled, "FetchBeaconBlock")
	}

	if f.GetFetchExecutionPayloadEnvelope() {
		enabled = append(enabled, "FetchExecutionPayloadEnvelope")
	}

	if f.GetFetchBeaconBadBlock() {
		enabled = append(enabled, "FetchBeaconBadBlock")
	}

	if f.GetFetchBeaconBadBlob() {
		enabled = append(enabled, "FetchBeaconBadBlob")
	}

	if f.GetFetchExecutionBlockTrace() {
		enabled = append(enabled, "FetchExecutionBlockTrace")
	}

	if f.GetFetchExecutionBadBlock() {
		enabled = append(enabled, "FetchExecutionBadBlock")
	}

	return enabled
}

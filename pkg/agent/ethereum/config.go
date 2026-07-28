package ethereum

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/phase0"
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

	// MaxConcurrentBeaconStateFetches bounds how many beacon states may be fetched
	// at once across every agent in the process. Each fetch holds the raw state and
	// its compressed copy in memory, so this caps peak memory independently of how
	// many agents are configured. Applied process-wide by whichever agent starts
	// first, so it is only meaningful when set consistently across agents.
	MaxConcurrentBeaconStateFetches int `yaml:"maxConcurrentBeaconStateFetches" default:"10"`

	// MaxConcurrentExecutionBadBlockFetches bounds how many bad block fetches may
	// be in flight at once across every agent in the process. The response carries
	// every bad block the node still holds and is decoded into memory for the whole
	// indexing pass. Applied process-wide, as above.
	MaxConcurrentExecutionBadBlockFetches int `yaml:"maxConcurrentExecutionBadBlockFetches" default:"4"`
}

// GetMaxConcurrentBeaconStateFetches returns the configured beacon state fetch
// concurrency, or 0 to use the agent default.
func (c *Config) GetMaxConcurrentBeaconStateFetches() int {
	if c.MaxConcurrentBeaconStateFetches <= 0 {
		return 0
	}

	return c.MaxConcurrentBeaconStateFetches
}

// GetMaxConcurrentExecutionBadBlockFetches returns the configured bad block fetch
// concurrency, or 0 to use the agent default.
func (c *Config) GetMaxConcurrentExecutionBadBlockFetches() int {
	if c.MaxConcurrentExecutionBadBlockFetches <= 0 {
		return 0
	}

	return c.MaxConcurrentExecutionBadBlockFetches
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
	FetchBeaconState         *bool `yaml:"fetchBeaconState" default:"true"`
	FetchBeaconBlock         *bool `yaml:"fetchBeaconBlock" default:"true"`
	FetchBeaconBadBlock      *bool `yaml:"fetchBeaconBadBlock" default:"true"`
	FetchBeaconBadBlob       *bool `yaml:"fetchBeaconBadBlob" default:"true"`
	FetchExecutionBlockTrace *bool `yaml:"fetchExecutionBlockTrace" default:"true"`
	FetchExecutionBadBlock   *bool `yaml:"fetchExecutionBadBlock" default:"true"`
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

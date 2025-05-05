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

	// FetchOldBeaconStates is a flag to fetch beacon states that are more then
	// a certain number of epochs old.
	// This is to prevent fetching states that are too far behind the head.
	FetchOldBeaconStates *FetchOldBeaconStates `yaml:"fetchOldBeaconStates" default:"{\"enabled\": false}"`
}

type FetchOldBeaconStates struct {
	Enabled *bool  `yaml:"enabled" default:"false"`
	Epochs  uint64 `yaml:"epochs" default:"1"`
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

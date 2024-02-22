package ethereum

import (
	"fmt"

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
}

func (c *Config) Validate() error {
	if err := c.Execution.Validate(); err != nil {
		return fmt.Errorf("invalid execution configuration: %w", err)
	}

	if err := c.Beacon.Validate(); err != nil {
		return fmt.Errorf("invalid beacon configuration: %w", err)
	}

	return nil
}

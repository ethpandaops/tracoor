package single

import (
	"fmt"

	"github.com/ethpandaops/tracoor/pkg/agent"
	"github.com/ethpandaops/tracoor/pkg/agent/indexer"
	"github.com/ethpandaops/tracoor/pkg/server"
	"github.com/ethpandaops/tracoor/pkg/store"
)

type SharedConfig struct {
	// LoggingLevel is the logging level to use.
	LoggingLevel string `yaml:"logging" default:"info"`
	// Indexer is the indexer to use.
	Indexer *indexer.Config `yaml:"indexer"`
	// Store is the store to use.
	Store *store.Config `yaml:"store"`
	// MetricsAddr is the address to serve metrics on.
	MetricsAddr string `yaml:"metricsAddr" default:"localhost:8080"`
}

type Config struct {
	Shared *SharedConfig   `yaml:"shared"`
	Server *server.Config  `yaml:"server"`
	Agents []*agent.Config `yaml:"agents"`
}

func (c *Config) Validate() error {
	if c.Server == nil {
		return fmt.Errorf("server configuration is required")
	}

	if c.Shared == nil || c.Shared.Indexer == nil {
		return fmt.Errorf("indexer configuration is required")
	}

	if len(c.Agents) == 0 {
		return fmt.Errorf("at least one agent configuration is required. If you just want to run the server, use the `server` subcommand instead")
	}

	return nil
}

// ApplyShared sets the shared config in to the server and all agents
func (c *Config) ApplyShared() error {
	c.Server.Store = *c.Shared.Store
	c.Server.LoggingLevel = c.Shared.LoggingLevel
	c.Server.MetricsAddr = c.Shared.MetricsAddr

	for _, agent := range c.Agents {
		agent.Indexer = c.Shared.Indexer
		agent.Store = c.Shared.Store
		agent.LoggingLevel = c.Shared.LoggingLevel
		agent.MetricsAddr = c.Shared.MetricsAddr
	}

	return nil
}

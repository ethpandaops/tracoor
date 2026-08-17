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
	// ShutdownTimeoutSeconds bounds how long each agent waits for its in-flight
	// fetches and uploads once a shutdown starts. One value covers every agent.
	ShutdownTimeoutSeconds int `yaml:"shutdownTimeoutSeconds" default:"10"`
	// FetchTimeouts bounds one fetch attempt per artifact kind. One set of
	// values covers every agent; an agent may still set its own.
	FetchTimeouts agent.FetchTimeouts `yaml:"fetchTimeouts"`
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

	// Agent names key metrics registration, storage handshakes and index rows,
	// so a duplicate cannot be tolerated.
	names := make(map[string]struct{}, len(c.Agents))

	for _, a := range c.Agents {
		if a.Name == "" {
			return fmt.Errorf("every agent requires a name")
		}

		if _, ok := names[a.Name]; ok {
			return fmt.Errorf("duplicate agent name: %s", a.Name)
		}

		names[a.Name] = struct{}{}
	}

	return nil
}

// ApplyShared sets the shared config in to the server and all agents.
func (c *Config) ApplyShared() error {
	c.Server.Store = *c.Shared.Store
	c.Server.LoggingLevel = c.Shared.LoggingLevel
	c.Server.MetricsAddr = c.Shared.MetricsAddr

	for _, agent := range c.Agents {
		agent.Indexer = c.Shared.Indexer
		agent.Store = c.Shared.Store
		agent.LoggingLevel = c.Shared.LoggingLevel
		agent.MetricsAddr = c.Shared.MetricsAddr

		// Unlike the fields above, an agent may want its own grace period — a
		// node behind slow storage drains at a different rate to the rest — so
		// the shared value only fills in the agents that did not set one.
		if agent.ShutdownTimeoutSeconds == 0 {
			agent.ShutdownTimeoutSeconds = c.Shared.ShutdownTimeoutSeconds
		}

		// Per-field, so an agent overriding one artifact's deadline still
		// inherits the shared values for the rest.
		if agent.FetchTimeouts.BeaconStateSeconds == 0 {
			agent.FetchTimeouts.BeaconStateSeconds = c.Shared.FetchTimeouts.BeaconStateSeconds
		}

		if agent.FetchTimeouts.BeaconBlockSeconds == 0 {
			agent.FetchTimeouts.BeaconBlockSeconds = c.Shared.FetchTimeouts.BeaconBlockSeconds
		}

		if agent.FetchTimeouts.ExecutionPayloadEnvelopeSeconds == 0 {
			agent.FetchTimeouts.ExecutionPayloadEnvelopeSeconds = c.Shared.FetchTimeouts.ExecutionPayloadEnvelopeSeconds
		}

		if agent.FetchTimeouts.ExecutionBlockTraceSeconds == 0 {
			agent.FetchTimeouts.ExecutionBlockTraceSeconds = c.Shared.FetchTimeouts.ExecutionBlockTraceSeconds
		}

		if agent.FetchTimeouts.ExecutionBadBlockSeconds == 0 {
			agent.FetchTimeouts.ExecutionBadBlockSeconds = c.Shared.FetchTimeouts.ExecutionBadBlockSeconds
		}
	}

	return nil
}

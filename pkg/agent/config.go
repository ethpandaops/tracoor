package agent

import (
	"errors"
	"time"

	"github.com/ethpandaops/tracoor/pkg/agent/ethereum"
	"github.com/ethpandaops/tracoor/pkg/agent/indexer"
	"github.com/ethpandaops/tracoor/pkg/store"
)

// defaultShutdownTimeout is used when no grace period was configured. It also
// covers `single` mode, where the per-agent configs are unmarshalled after the
// defaults have been applied and so never pick up the struct tag.
const defaultShutdownTimeout = 10 * time.Second

type Config struct {
	LoggingLevel string  `yaml:"logging" default:"info"`
	MetricsAddr  string  `yaml:"metricsAddr" default:":9090"`
	PProfAddr    *string `yaml:"pprofAddr"`

	// ShutdownTimeoutSeconds bounds how long we wait for in-flight fetches and
	// uploads to finish after a shutdown signal. Anything still running when it
	// expires is abandoned; the store-then-index write order makes that safe.
	// Keep it strictly below the orchestrator's termination grace period, or the
	// process is killed mid-drain and the wait achieves nothing.
	ShutdownTimeoutSeconds int `yaml:"shutdownTimeoutSeconds" default:"10"`

	// The name of the agent
	Name string `yaml:"name"`

	// Ethereum configuration
	Ethereum ethereum.Config `yaml:"ethereum"`

	// Indexer configuration
	Indexer *indexer.Config `yaml:"indexer"`

	// Store configuration
	Store *store.Config `yaml:"store"`
}

// ShutdownTimeout is how long the agent waits for its queue workers after its
// context is cancelled, before abandoning whatever is still running.
func (c *Config) ShutdownTimeout() time.Duration {
	if c.ShutdownTimeoutSeconds <= 0 {
		return defaultShutdownTimeout
	}

	return time.Duration(c.ShutdownTimeoutSeconds) * time.Second
}

func (c *Config) Validate() error {
	if c.Name == "" {
		return errors.New("name is required")
	}

	if err := c.Ethereum.Validate(); err != nil {
		return err
	}

	return nil
}

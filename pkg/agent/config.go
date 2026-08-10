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

// FetchTimeouts bounds a single fetch attempt per artifact kind. The defaults
// suit an agent deployed near its node, where even a state transfer is
// sub-second. A WAN link needs minutes for states: a 73 MB devnet state at
// 2 MB/s is already past the default before it finishes. Keep values under the
// artifact's availability window (~6 minutes for states on the tightest
// clients) or the fetch outlives the data it is fetching.
type FetchTimeouts struct {
	BeaconStateSeconds              int `yaml:"beaconStateSeconds" default:"30"`
	BeaconBlockSeconds              int `yaml:"beaconBlockSeconds" default:"30"`
	ExecutionPayloadEnvelopeSeconds int `yaml:"executionPayloadEnvelopeSeconds" default:"60"`
	ExecutionBlockTraceSeconds      int `yaml:"executionBlockTraceSeconds" default:"60"`
	ExecutionBadBlockSeconds        int `yaml:"executionBadBlockSeconds" default:"60"`
}

// fetchTimeout falls back when a value was never configured — `single` mode
// unmarshals per-agent configs after the defaults processor has run, so the
// struct tags alone do not guarantee a value.
func fetchTimeout(seconds int, fallback time.Duration) time.Duration {
	if seconds <= 0 {
		return fallback
	}

	return time.Duration(seconds) * time.Second
}

func (f FetchTimeouts) BeaconState() time.Duration {
	return fetchTimeout(f.BeaconStateSeconds, 30*time.Second)
}

func (f FetchTimeouts) BeaconBlock() time.Duration {
	return fetchTimeout(f.BeaconBlockSeconds, 30*time.Second)
}

func (f FetchTimeouts) ExecutionPayloadEnvelope() time.Duration {
	return fetchTimeout(f.ExecutionPayloadEnvelopeSeconds, 60*time.Second)
}

func (f FetchTimeouts) ExecutionBlockTrace() time.Duration {
	return fetchTimeout(f.ExecutionBlockTraceSeconds, 60*time.Second)
}

func (f FetchTimeouts) ExecutionBadBlock() time.Duration {
	return fetchTimeout(f.ExecutionBadBlockSeconds, 60*time.Second)
}

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

	// FetchTimeouts bounds one fetch attempt per artifact kind. States are the
	// largest artifact by far, so their value decides whether a distant agent
	// can capture them at all.
	FetchTimeouts FetchTimeouts `yaml:"fetchTimeouts"`

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

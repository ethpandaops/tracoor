package promotion

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethpandaops/beacon/pkg/human"
	"github.com/ethpandaops/tracoor/pkg/store"
)

// Config configures the promotion service, which copies "interesting"
// (pre-state, block) pairs from the short-retention buffer into a separate,
// append-only corpus store.
type Config struct {
	// Enabled turns the service on. Opt-in.
	Enabled bool `yaml:"enabled" default:"false"`

	// LagEpochs is how many epochs behind the freshest indexed epoch the
	// service processes. Hindsight is load-bearing: it is what makes both
	// branches of a reorg available in the buffer.
	LagEpochs uint64 `yaml:"lagEpochs" default:"2"`

	// CheckInterval is how often the service looks for newly lagged epochs.
	CheckInterval human.Duration `yaml:"checkInterval" default:"30s"`

	// SlotsPerEpoch and SecondsPerSlot describe the target network(s). They
	// feed the retention safety check and the manifests; epoch grouping
	// itself uses the agent-computed epoch column of the index.
	SlotsPerEpoch  uint64         `yaml:"slotsPerEpoch" default:"32"`
	SecondsPerSlot human.Duration `yaml:"secondsPerSlot" default:"12s"`

	// RateCapPerHour caps common-tier promotions (gap, low_participation,
	// baseline, undecodable_fork, deposit, execution_request) per network
	// per hour. It is the flood guard: when exhausted the service skips -
	// it never queues and never deletes.
	RateCapPerHour uint64 `yaml:"rateCapPerHour" default:"30"`

	// RareCapPerHour is a separate budget for rare-tier promotions
	// (slashing, voluntary_exit, bls_to_execution_change, fork_boundary),
	// so the rarest captures never compete with a participation/gap flood
	// during a stall, while a mass-slashing incident still meets a hard
	// ceiling. Reorg promotions are uncapped (self-limiting per slot).
	RareCapPerHour uint64 `yaml:"rareCapPerHour" default:"120"`

	// BaselineEveryNEpochs promotes the first qualifying slot of every Nth
	// epoch regardless of triggers; a corpus of only pathologies is its own
	// blind spot. Fires on epoch%N == 0 so re-processing selects the same
	// slots.
	BaselineEveryNEpochs uint64 `yaml:"baselineEveryNEpochs" default:"8"`

	// GapTrigger is the minimum run of skipped slots before a block that
	// fires the gap trigger.
	GapTrigger uint64 `yaml:"gapTrigger" default:"2"`

	// SyncParticipationFloor fires the low_participation trigger when the
	// sync committee bit density falls below it. Requires a typed decode.
	SyncParticipationFloor float64 `yaml:"syncParticipationFloor" default:"0.95"`

	// Triggers toggles individual triggers. All default to enabled.
	Triggers TriggersConfig `yaml:"triggers"`

	// Store is the corpus store. It must not be the buffer store, and its
	// credentials should be write-only (no delete): the service never
	// deletes anything, anywhere.
	Store store.Config `yaml:"store"`
}

// TriggersConfig holds per-trigger toggles. A nil value means enabled.
type TriggersConfig struct {
	Slashing             *bool `yaml:"slashing"`
	VoluntaryExit        *bool `yaml:"voluntaryExit"`
	BLSToExecutionChange *bool `yaml:"blsToExecutionChange"`
	Deposit              *bool `yaml:"deposit"`
	ExecutionRequest     *bool `yaml:"executionRequest"`
	Gap                  *bool `yaml:"gap"`
	Reorg                *bool `yaml:"reorg"`
	LowParticipation     *bool `yaml:"lowParticipation"`
	ForkBoundary         *bool `yaml:"forkBoundary"`
	UndecodableFork      *bool `yaml:"undecodableFork"`
	Baseline             *bool `yaml:"baseline"`
}

func triggerEnabled(v *bool) bool {
	return v == nil || *v
}

func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.SlotsPerEpoch == 0 {
		return errors.New("promotion: slotsPerEpoch must be > 0")
	}

	if c.SecondsPerSlot.Duration <= 0 {
		return errors.New("promotion: secondsPerSlot must be > 0")
	}

	if c.CheckInterval.Duration <= 0 {
		return errors.New("promotion: checkInterval must be > 0")
	}

	if c.SyncParticipationFloor < 0 || c.SyncParticipationFloor > 1 {
		return errors.New("promotion: syncParticipationFloor must be within [0, 1]")
	}

	if err := c.Store.Validate(); err != nil {
		return fmt.Errorf("promotion store: %w", err)
	}

	return nil
}

// ValidateRetention refuses to run when the buffer cannot outlive the
// processing lag: the service reads slots lagEpochs behind head from a buffer
// the retention reaper empties, so retention must exceed the lag plus at
// least one epoch of processing margin or promotion silently loses data.
func (c *Config) ValidateRetention(retention time.Duration) error {
	if !c.Enabled {
		return nil
	}

	epoch := time.Duration(c.SlotsPerEpoch) * c.SecondsPerSlot.Duration //nolint:gosec // bounded by config validation

	required := time.Duration(c.LagEpochs+1) * epoch //nolint:gosec // bounded by config validation
	if retention < required {
		return fmt.Errorf(
			"promotion: beacon state/block retention %s is below lag %d epochs + 1 epoch margin (%s); increase retention or reduce lagEpochs",
			retention, c.LagEpochs, required,
		)
	}

	return nil
}

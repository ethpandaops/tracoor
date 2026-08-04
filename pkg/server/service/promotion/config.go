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
	//
	// The window is tumbling, not sliding: it resets an hour after its first
	// promotion, so a burst straddling the boundary can spend two budgets
	// within a few minutes. That is deliberate - the cap bounds sustained
	// volume, not instantaneous burstiness.
	//
	// A cap of 0 promotes nothing in that tier. To drop a class of captures,
	// prefer disabling its trigger: a zero cap is indistinguishable in the
	// metrics from a permanently exhausted budget.
	RateCapPerHour uint64 `yaml:"rateCapPerHour" default:"30"`

	// RareCapPerHour is a separate budget for rare-tier promotions
	// (slashing, voluntary_exit, bls_to_execution_change, fork_boundary),
	// so the rarest captures never compete with a participation/gap flood
	// during a stall, while a mass-slashing incident still meets a hard
	// ceiling.
	RareCapPerHour uint64 `yaml:"rareCapPerHour" default:"120"`

	// ReorgCapPerHour is a deliberately generous ceiling for reorg
	// promotions. Reorgs are self-limiting per slot and the orphaned branch
	// is unobtainable anywhere else, so they get their own budget rather
	// than competing with anything - but "self-limiting per slot" is not a
	// bound in aggregate: an equivocation storm, or an agent inserting many
	// distinct roots per slot, is otherwise unbounded corpus spend.
	ReorgCapPerHour uint64 `yaml:"reorgCapPerHour" default:"1000"`

	// BaselineEveryNEpochs promotes the lowest indexed slot of every Nth
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

	// Manifests record seconds_per_slot as an integer, so a sub-second value
	// would be published as 0 rather than truncated silently. No beacon
	// chain spec uses one.
	if c.SecondsPerSlot.Duration < time.Second {
		return errors.New("promotion: secondsPerSlot must be >= 1s")
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

// Store config fields that decide whether two store configs address the same
// physical destination.
const (
	keyEndpoint   = "endpoint"
	keyBucketName = "bucket_name"
	keyBasePath   = "base_path"
)

var identityKeys = map[store.Type][]string{
	store.S3StoreType: {keyEndpoint, keyBucketName},
	store.FSStoreType: {keyBasePath},
}

// ValidateDistinctFrom refuses a corpus store that addresses the same
// destination as the buffer store. The corpus is append-only and outlives
// every devnet; the buffer is reaped on a schedule. Sharing a bucket between
// them is never intended, and the failure mode - corpus objects sitting in a
// bucket somebody eventually points a lifecycle rule at - is silent.
func (c *Config) ValidateDistinctFrom(buffer store.Config) error {
	if !c.Enabled || c.Store.Type != buffer.Type {
		return nil
	}

	keys, ok := identityKeys[c.Store.Type]
	if !ok {
		return nil
	}

	corpusCfg, err := rawConfigMap(c.Store)
	if err != nil {
		return fmt.Errorf("promotion store: %w", err)
	}

	bufferCfg, err := rawConfigMap(buffer)
	if err != nil {
		return fmt.Errorf("promotion store: %w", err)
	}

	if len(corpusCfg) == 0 || len(bufferCfg) == 0 {
		return nil
	}

	for _, key := range keys {
		if fmt.Sprint(corpusCfg[key]) != fmt.Sprint(bufferCfg[key]) {
			return nil
		}
	}

	return fmt.Errorf(
		"promotion: the corpus store addresses the same %s destination as the buffer store (%v); the corpus must be a separate, append-only location",
		c.Store.Type, keys,
	)
}

func rawConfigMap(conf store.Config) (map[string]any, error) {
	out := map[string]any{}
	if conf.Config.IsZero() {
		return out, nil
	}

	if err := conf.Config.Unmarshal(&out); err != nil {
		return nil, err
	}

	return out, nil
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

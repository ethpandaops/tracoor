package promotion

import (
	"testing"
	"time"

	"github.com/creasty/defaults"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The zero-value config (promotion absent from yaml) must take defaults and
// validate: defaults.Set runs at every server startup.
func TestConfigDefaults(t *testing.T) {
	conf := &Config{}
	require.NoError(t, defaults.Set(conf))

	assert.False(t, conf.Enabled)
	assert.Equal(t, uint64(2), conf.LagEpochs)
	assert.Equal(t, 30*time.Second, conf.CheckInterval.Duration)
	assert.Equal(t, uint64(32), conf.SlotsPerEpoch)
	assert.Equal(t, 12*time.Second, conf.SecondsPerSlot.Duration)
	assert.Equal(t, uint64(30), conf.RateCapPerHour)
	assert.Equal(t, uint64(8), conf.BaselineEveryNEpochs)
	assert.Equal(t, uint64(2), conf.GapTrigger)
	assert.InDelta(t, 0.95, conf.SyncParticipationFloor, 1e-9)

	// Disabled promotion never blocks startup.
	require.NoError(t, conf.Validate())
}

// The service refuses to run when the buffer cannot outlive the processing
// lag plus one epoch of margin.
func TestValidateRetention(t *testing.T) {
	conf := testConfig(t, t.TempDir())
	require.NoError(t, conf.Validate())

	// lag 2 + 1 margin epochs x 32 slots x 2s = 192s required.
	assert.Error(t, conf.ValidateRetention(191*time.Second))
	assert.NoError(t, conf.ValidateRetention(192*time.Second))
	assert.NoError(t, conf.ValidateRetention(2*time.Hour))

	// Disabled service never blocks startup.
	conf.Enabled = false
	assert.NoError(t, conf.ValidateRetention(time.Second))
}

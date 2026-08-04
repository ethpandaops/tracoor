package promotion

import (
	"testing"
	"time"

	"github.com/creasty/defaults"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testEndpoint     = "http://minio:9000"
	testCorpusBucket = "corpus"
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
	assert.Equal(t, uint64(120), conf.RareCapPerHour)
	assert.Equal(t, uint64(1000), conf.ReorgCapPerHour)
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

// Manifests publish seconds_per_slot as an integer, so a sub-second value
// would be published as 0 rather than truncated silently.
func TestValidateRejectsSubSecondSlots(t *testing.T) {
	conf := testConfig(t, t.TempDir())
	conf.SecondsPerSlot.Duration = 500 * time.Millisecond

	assert.Error(t, conf.Validate())
}

// The corpus outlives every devnet; the buffer is reaped. Pointing both at
// one destination is never intended, and the failure is otherwise silent.
func TestValidateDistinctFromBufferStore(t *testing.T) {
	dir := t.TempDir()
	conf := testConfig(t, dir)

	same := storeConfig(t, "fs", map[string]string{keyBasePath: dir})
	other := storeConfig(t, "fs", map[string]string{keyBasePath: t.TempDir()})

	require.Error(t, conf.ValidateDistinctFrom(same))
	require.NoError(t, conf.ValidateDistinctFrom(other))

	// A different backend is a different destination by construction.
	s3 := storeConfig(t, "s3", map[string]string{keyBucketName: "buffer", keyEndpoint: testEndpoint})
	require.NoError(t, conf.ValidateDistinctFrom(s3))

	// Disabled promotion never blocks startup.
	conf.Enabled = false
	require.NoError(t, conf.ValidateDistinctFrom(same))
}

// Two S3 stores are the same destination only when both endpoint and bucket
// match: one bucket per purpose is the intended layout.
func TestValidateDistinctFromS3(t *testing.T) {
	conf := testConfig(t, t.TempDir())
	conf.Store = storeConfig(t, "s3", map[string]string{keyBucketName: testCorpusBucket, keyEndpoint: testEndpoint})

	sameBucket := storeConfig(t, "s3", map[string]string{keyBucketName: testCorpusBucket, keyEndpoint: testEndpoint})
	otherBucket := storeConfig(t, "s3", map[string]string{keyBucketName: "buffer", keyEndpoint: testEndpoint})
	otherEndpoint := storeConfig(t, "s3", map[string]string{keyBucketName: testCorpusBucket, keyEndpoint: "http://other:9000"})

	assert.Error(t, conf.ValidateDistinctFrom(sameBucket))
	assert.NoError(t, conf.ValidateDistinctFrom(otherBucket))
	assert.NoError(t, conf.ValidateDistinctFrom(otherEndpoint))
}

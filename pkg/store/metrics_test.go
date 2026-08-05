package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// One instance per namespace: a process runs both the capture buffer and the
// promotion corpus, and a single global instance attributed every store's
// activity to whichever one was constructed first.
func TestBasicMetricsAreNamespaced(t *testing.T) {
	buffer := GetBasicMetricsInstance("test_namespaced_buffer", string(FSStoreType), false)
	corpus := GetBasicMetricsInstance("test_namespaced_corpus", string(FSStoreType), false)

	assert.NotSame(t, buffer, corpus)
	assert.Same(t, buffer, GetBasicMetricsInstance("test_namespaced_buffer", string(FSStoreType), false))
}

// A namespace first created with metrics disabled must still be exported when
// a later store on the same namespace wants them.
func TestBasicMetricsRegisterOnUpgrade(t *testing.T) {
	const namespace = "test_upgrade"

	disabled := GetBasicMetricsInstance(namespace, string(FSStoreType), false)
	assert.False(t, disabled.registered)

	enabled := GetBasicMetricsInstance(namespace, string(FSStoreType), true)
	assert.Same(t, disabled, enabled)
	assert.True(t, enabled.registered)

	// Registering is idempotent: a third store must not panic on a
	// duplicate collector.
	assert.NotPanics(t, func() { GetBasicMetricsInstance(namespace, string(FSStoreType), true) })
}

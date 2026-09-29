package promotion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// One instance per namespace, like the store metrics: a process could run more
// than one promoter (tests do), and a single global instance would attribute
// every promoter's activity to whichever one was constructed first.
func TestMetricsAreNamespaced(t *testing.T) {
	a := GetMetricsInstance("test_promotion_metrics_a")
	b := GetMetricsInstance("test_promotion_metrics_b")

	assert.NotSame(t, a, b)
	assert.Same(t, a, GetMetricsInstance("test_promotion_metrics_a"))
}

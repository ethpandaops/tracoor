package promotion

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	promotions         *prometheus.CounterVec
	captures           *prometheus.CounterVec
	unpaired           *prometheus.CounterVec
	skips              *prometheus.CounterVec
	errors             *prometheus.CounterVec
	corpusBytes        *prometheus.CounterVec
	lastProcessedEpoch *prometheus.GaugeVec
}

// Skip reasons. Every non-promotion of a candidate that fired a trigger is
// visible, never silent.
const (
	SkipReasonRateCappedCommon = "rate_capped_common"
	SkipReasonRateCappedRare   = "rate_capped_rare"
	SkipReasonMissingObject    = "missing_object"
	SkipReasonMalformed        = "malformed"
	SkipReasonGVRAmbiguous     = "gvr_ambiguous"
	SkipReasonAlreadyPromoted  = "already_promoted"
)

var (
	metricsInstance *Metrics
	metricsOnce     sync.Once
)

// GetMetricsInstance returns the process-wide promotion metrics, registering
// them at most once so multiple service instances (restarts, tests) are safe.
func GetMetricsInstance(namespace string, enabled bool) *Metrics {
	metricsOnce.Do(func() {
		metricsInstance = newMetrics(namespace, enabled)
	})

	return metricsInstance
}

func newMetrics(namespace string, enabled bool) *Metrics {
	m := &Metrics{
		promotions: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "promotions_total",
			Help:      "Number of promotions per trigger (a capture with N triggers counts N times).",
		}, []string{labelNetwork, "trigger"}),
		captures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "captures_total",
			Help:      "Number of captures written to the corpus.",
		}, []string{labelNetwork, "branch"}),
		unpaired: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "unpaired_captures_total",
			Help:      "Number of captures promoted without a pre-state (buffer holes).",
		}, []string{labelNetwork}),
		skips: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "skips_total",
			Help:      "Number of triggered candidates skipped instead of promoted.",
		}, []string{labelNetwork, "reason"}),
		errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "errors_total",
			Help:      "Number of errors while processing candidates.",
		}, []string{labelNetwork}),
		corpusBytes: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "corpus_bytes_total",
			Help:      "Bytes written to the corpus store.",
		}, []string{labelNetwork, "kind"}),
		lastProcessedEpoch: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "last_processed_epoch",
			Help:      "Highest epoch processed per network.",
		}, []string{labelNetwork}),
	}

	if enabled {
		prometheus.MustRegister(
			m.promotions,
			m.captures,
			m.unpaired,
			m.skips,
			m.errors,
			m.corpusBytes,
			m.lastProcessedEpoch,
		)
	}

	return m
}

func (m *Metrics) ObserveTrigger(network, trigger string) {
	m.promotions.WithLabelValues(network, trigger).Inc()
}

func (m *Metrics) ObserveCapture(network, branch string) {
	m.captures.WithLabelValues(network, branch).Inc()
}

func (m *Metrics) ObserveUnpaired(network string) {
	m.unpaired.WithLabelValues(network).Inc()
}

func (m *Metrics) ObserveSkip(network, reason string) {
	m.skips.WithLabelValues(network, reason).Inc()
}

func (m *Metrics) ObserveError(network string) {
	m.errors.WithLabelValues(network).Inc()
}

func (m *Metrics) ObserveCorpusBytes(network, kind string, n int) {
	m.corpusBytes.WithLabelValues(network, kind).Add(float64(n))
}

func (m *Metrics) ObserveLastProcessedEpoch(network string, epoch uint64) {
	m.lastProcessedEpoch.WithLabelValues(network).Set(float64(epoch))
}

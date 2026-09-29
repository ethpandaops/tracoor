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
	networkResets      *prometheus.CounterVec
	lastProcessedEpoch *prometheus.GaugeVec
}

// Skip reasons. Every non-promotion of a candidate that fired a trigger is
// visible, never silent.
const (
	SkipReasonRateCappedCommon = "rate_capped_common"
	SkipReasonRateCappedRare   = "rate_capped_rare"
	SkipReasonRateCappedReorg  = "rate_capped_reorg"
	SkipReasonMissingObject    = "missing_object"
	SkipReasonMalformed        = "malformed"
	SkipReasonGVRAmbiguous     = "gvr_ambiguous"
	SkipReasonAlreadyPromoted  = "already_promoted"
	SkipReasonRowLimit         = "row_limit_truncated"
	SkipReasonImplausibleEpoch = "implausible_head_epoch"
)

// Reasons a network's per-process state was discarded and rebuilt.
const (
	ResetReasonHeightRegression = "height_regression"
	ResetReasonIdentityChange   = "identity_change"
)

// metricsInstances is one Metrics per namespace, so the collectors are
// registered at most once per namespace and multiple service instances
// (restarts, tests) are safe.
var (
	metricsInstances = map[string]*Metrics{}
	metricsMu        sync.Mutex
)

// GetMetricsInstance returns the promotion metrics for a namespace, creating
// and registering them on first use.
func GetMetricsInstance(namespace string) *Metrics {
	metricsMu.Lock()
	defer metricsMu.Unlock()

	if instance, ok := metricsInstances[namespace]; ok {
		return instance
	}

	instance := newMetrics(namespace)
	metricsInstances[namespace] = instance

	return instance
}

func newMetrics(namespace string) *Metrics {
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
		networkResets: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "network_resets_total",
			Help:      "Number of times a network's promotion cursor and identity were discarded and rebuilt.",
		}, []string{labelNetwork, "reason"}),
		lastProcessedEpoch: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "last_processed_epoch",
			Help:      "Highest epoch processed per network.",
		}, []string{labelNetwork}),
	}

	prometheus.MustRegister(
		m.promotions,
		m.captures,
		m.unpaired,
		m.skips,
		m.errors,
		m.corpusBytes,
		m.networkResets,
		m.lastProcessedEpoch,
	)

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

func (m *Metrics) ObserveNetworkReset(network, reason string) {
	m.networkResets.WithLabelValues(network, reason).Inc()
}

func (m *Metrics) ObserveLastProcessedEpoch(network string, epoch uint64) {
	m.lastProcessedEpoch.WithLabelValues(network).Set(float64(epoch))
}

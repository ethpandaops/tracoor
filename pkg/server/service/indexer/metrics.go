package indexer

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	labelKind    = "kind"
	labelReason  = "reason"
	labelNetwork = "network"
	labelCause   = "cause"
)

// Metrics covers what retention does and what it declines to do. The skip reasons matter as
// much as the successes: a collector that keeps deciding not to collect is the failure mode
// that would otherwise be invisible.
type Metrics struct {
	rowsPurged      *prometheus.CounterVec
	purgeDuration   *prometheus.HistogramVec
	purgeBacklog    *prometheus.GaugeVec
	blobsCollected  prometheus.Counter
	blobGCSkipped   *prometheus.CounterVec
	rootDivergence  *prometheus.CounterVec
	archiveHoldback *prometheus.CounterVec
}

var (
	metricsInstance *Metrics
	metricsOnce     sync.Once
)

// NewMetrics returns the process-wide indexer metrics. Collectors are registered once: a
// second indexer in the same process shares them rather than fighting the registry.
func NewMetrics(namespace string) *Metrics {
	metricsOnce.Do(func() {
		m := &Metrics{
			rowsPurged: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "rows_purged_total",
				Help:      "Number of artifact rows removed by retention",
			}, []string{labelKind}),
			purgeDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "purge_duration_seconds",
				Help:      "Time taken by one retention pass over a kind",
				Buckets:   prometheus.ExponentialBuckets(0.01, 3, 8),
			}, []string{labelKind}),
			purgeBacklog: prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "purge_backlog_rows",
				Help:      "Rows still older than their retention window after a pass",
			}, []string{labelKind}),
			blobsCollected: prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "blobs_collected_total",
				Help:      "Number of deduplicated payloads removed by the blob collector",
			}),
			blobGCSkipped: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "blob_gc_skipped_total",
				Help:      "Blob collection candidates left in place, by reason",
			}, []string{labelReason}),
			rootDivergence: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "root_divergence_total",
				Help:      "Distinct slot observations with more than one root, by what explains them",
			}, []string{labelNetwork, labelKind, labelCause}),
			archiveHoldback: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "permanent_archive_holdback_total",
				Help:      "Block rows retention held back from purge because their archive was not confirmed, by reason",
			}, []string{labelReason}),
		}

		prometheus.MustRegister(
			m.rowsPurged,
			m.purgeDuration,
			m.purgeBacklog,
			m.blobsCollected,
			m.blobGCSkipped,
			m.rootDivergence,
			m.archiveHoldback,
		)

		metricsInstance = m
	})

	return metricsInstance
}

func (m *Metrics) ObserveRowsPurged(kind string, count int64) {
	m.rowsPurged.WithLabelValues(kind).Add(float64(count))
}

func (m *Metrics) ObservePurgeDuration(kind string, seconds float64) {
	m.purgeDuration.WithLabelValues(kind).Observe(seconds)
}

func (m *Metrics) SetPurgeBacklog(kind string, rows int64) {
	m.purgeBacklog.WithLabelValues(kind).Set(float64(rows))
}

func (m *Metrics) ObserveBlobCollected() {
	m.blobsCollected.Inc()
}

func (m *Metrics) ObserveBlobGCSkipped(reason string) {
	m.blobGCSkipped.WithLabelValues(reason).Inc()
}

func (m *Metrics) ObserveRootDivergence(network, kind, cause string) {
	m.rootDivergence.WithLabelValues(network, kind, cause).Inc()
}

// ObserveArchiveHoldback records block rows a purge pass left in place because the permanent
// store had not confirmed them. A rising count is the archive falling behind retention — the
// failure mode that silently lost blocks before rows were held back.
func (m *Metrics) ObserveArchiveHoldback(reason string, count int64) {
	if count <= 0 {
		return
	}

	m.archiveHoldback.WithLabelValues(reason).Add(float64(count))
}

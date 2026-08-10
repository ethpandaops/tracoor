package agent

import (
	"context"
	"sync"
	"time"

	"github.com/ethpandaops/tracoor/pkg/observability"
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	queueSize               *prometheus.GaugeVec
	queueItemProcessingTime *prometheus.HistogramVec
	itemExported            *prometheus.CounterVec
	queueItemSkipped        *prometheus.CounterVec
	itemsDropped            *prometheus.CounterVec
	artifactUnsupported     *prometheus.CounterVec
}

type Queue string

var (
	BeaconStateQueue              Queue = "beacon_state"
	BeaconBlockQueue              Queue = "beacon_block"
	ExecutionPayloadEnvelopeQueue Queue = "execution_payload_envelope"
	BeaconBadBlockQueue           Queue = "beacon_bad_block"
	BeaconBadBlobQueue            Queue = "beacon_bad_blob"
	ExecutionBlockTraceQueue      Queue = "execution_block_trace"
	ExecutionBadBlockQueue        Queue = "execution_bad_block"
)

var (
	metricsInstance *Metrics
	once            sync.Once
)

func GetMetricsInstance(namespace string) *Metrics {
	once.Do(func() {
		metricsInstance = &Metrics{
			queueSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "queue_size",
				Help:      "The size of the queue",
			}, []string{labelQueue, labelAgent}),
			queueItemProcessingTime: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "queue_item_processing_time_seconds",
				Help:      "The time it takes to process an item from the queue",
				Buckets:   prometheus.LinearBuckets(0, 3, 10),
			}, []string{labelQueue, labelAgent}),
			itemExported: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "item_exported",
				Help:      "The number of items exported",
			}, []string{labelQueue, labelAgent}),
			queueItemSkipped: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "queue_item_skipped",
				Help:      "The number of items skipped",
			}, []string{labelQueue, labelAgent}),
			itemsDropped: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "items_dropped_total",
				Help:      "The number of items discarded without being indexed",
			}, []string{labelQueue, labelAgent, labelReason}),
			artifactUnsupported: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "artifact_unsupported_total",
				Help:      "The number of times a node was found to be unable to serve an artifact",
			}, []string{labelQueue, labelAgent}),
		}

		prometheus.MustRegister(metricsInstance.queueSize)
		prometheus.MustRegister(metricsInstance.queueItemProcessingTime)
		prometheus.MustRegister(metricsInstance.itemExported)
		prometheus.MustRegister(metricsInstance.queueItemSkipped)
		prometheus.MustRegister(metricsInstance.itemsDropped)
		prometheus.MustRegister(metricsInstance.artifactUnsupported)
	})

	return metricsInstance
}

func (m *Metrics) SetQueueSize(queue Queue, count int, agentName string) {
	m.queueSize.WithLabelValues(string(queue), agentName).Set(float64(count))
}

func (m *Metrics) ObserveQueueItemProcessingTime(queue Queue, duration time.Duration, agentName string) {
	m.queueItemProcessingTime.WithLabelValues(string(queue), agentName).Observe(duration.Seconds())
}

func (m *Metrics) IncrementItemExported(queue Queue, agentName string) {
	m.itemExported.WithLabelValues(string(queue), agentName).Inc()
}

func (m *Metrics) IncrementItemSkipped(queue Queue, agentName string) {
	m.queueItemSkipped.WithLabelValues(string(queue), agentName).Inc()
}

func (m *Metrics) IncrementItemDropped(queue Queue, agentName, reason string) {
	m.itemsDropped.WithLabelValues(string(queue), agentName, reason).Inc()
}

func (m *Metrics) IncrementArtifactUnsupported(queue Queue, agentName string) {
	m.artifactUnsupported.WithLabelValues(string(queue), agentName).Inc()
}

func (m *Metrics) ServeMetrics(ctx context.Context, addr string) {
	observability.StartMetricsServer(ctx, addr)
}

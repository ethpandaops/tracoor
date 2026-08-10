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
	blobCreated             *prometheus.CounterVec
	blobReused              *prometheus.CounterVec
	payloadMismatch         *prometheus.CounterVec
	payloadVerified         *prometheus.CounterVec
	transferIncomplete      *prometheus.CounterVec
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
			blobCreated: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "blob_created_total",
				Help:      "The number of payloads this agent compressed and stored for the first time",
			}, []string{labelQueue, labelAgent}),
			blobReused: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "blob_reused_total",
				Help:      "The number of times a verified payload was linked to bytes somebody else had already stored",
			}, []string{labelQueue, labelAgent}),
			payloadMismatch: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "payload_mismatch_total",
				Help:      "The number of divergences recorded, where a node served bytes that did not match the stored payload",
			}, []string{labelQueue, labelAgent}),
			payloadVerified: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "payload_verified_total",
				Help:      "The number of payloads read from a node in full and hashed",
			}, []string{labelQueue, labelAgent}),
			transferIncomplete: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "transfer_incomplete_total",
				Help:      "The number of payloads abandoned because they did not arrive in full",
			}, []string{labelQueue, labelAgent}),
		}

		prometheus.MustRegister(metricsInstance.queueSize)
		prometheus.MustRegister(metricsInstance.queueItemProcessingTime)
		prometheus.MustRegister(metricsInstance.itemExported)
		prometheus.MustRegister(metricsInstance.queueItemSkipped)
		prometheus.MustRegister(metricsInstance.itemsDropped)
		prometheus.MustRegister(metricsInstance.artifactUnsupported)
		prometheus.MustRegister(metricsInstance.blobCreated)
		prometheus.MustRegister(metricsInstance.blobReused)
		prometheus.MustRegister(metricsInstance.payloadMismatch)
		prometheus.MustRegister(metricsInstance.payloadVerified)
		prometheus.MustRegister(metricsInstance.transferIncomplete)
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

func (m *Metrics) IncrementBlobCreated(queue Queue, agentName string) {
	m.blobCreated.WithLabelValues(string(queue), agentName).Inc()
}

func (m *Metrics) IncrementBlobReused(queue Queue, agentName string) {
	m.blobReused.WithLabelValues(string(queue), agentName).Inc()
}

func (m *Metrics) IncrementPayloadMismatch(queue Queue, agentName string) {
	m.payloadMismatch.WithLabelValues(string(queue), agentName).Inc()
}

func (m *Metrics) IncrementPayloadVerified(queue Queue, agentName string) {
	m.payloadVerified.WithLabelValues(string(queue), agentName).Inc()
}

func (m *Metrics) IncrementTransferIncomplete(queue Queue, agentName string) {
	m.transferIncomplete.WithLabelValues(string(queue), agentName).Inc()
}

func (m *Metrics) ServeMetrics(ctx context.Context, addr string) {
	observability.StartMetricsServer(ctx, addr)
}

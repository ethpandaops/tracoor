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
	queueWaitTime           *prometheus.HistogramVec
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
	lengthUnverified        *prometheus.CounterVec
	flightAbandoned         *prometheus.CounterVec
	fetchBytes              *prometheus.CounterVec
	storedBytes             *prometheus.CounterVec
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
			queueWaitTime: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "queue_wait_seconds",
				Help:      "The time an item spent waiting in the queue before a worker picked it up",
				// A backlog is measured in slots, not seconds, so the buckets have
				// to reach well past a slot without losing resolution beneath one.
				Buckets: prometheus.ExponentialBuckets(0.1, 3, 9),
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
			lengthUnverified: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "payload_length_unverified_total",
				Help:      "The number of payloads that arrived without a length to check them against, so completeness rests on the read ending cleanly",
			}, []string{labelQueue, labelAgent}),
			flightAbandoned: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "dedup_flight_abandoned_total",
				Help:      "The number of times an agent stopped waiting on another node's upload and read its own node instead",
			}, []string{labelQueue, labelAgent}),
			fetchBytes: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "fetch_bytes_total",
				Help:      "The number of raw bytes read from a node",
			}, []string{labelQueue, labelAgent}),
			storedBytes: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "stored_bytes_total",
				Help:      "The number of compressed bytes handed to the store; the gap against fetch_bytes_total is what dedup saved",
			}, []string{labelQueue, labelAgent}),
		}

		prometheus.MustRegister(metricsInstance.queueSize)
		prometheus.MustRegister(metricsInstance.queueWaitTime)
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
		prometheus.MustRegister(metricsInstance.lengthUnverified)
		prometheus.MustRegister(metricsInstance.flightAbandoned)
		prometheus.MustRegister(metricsInstance.fetchBytes)
		prometheus.MustRegister(metricsInstance.storedBytes)
	})

	return metricsInstance
}

func (m *Metrics) SetQueueSize(queue Queue, count int, agentName string) {
	m.queueSize.WithLabelValues(string(queue), agentName).Set(float64(count))
}

func (m *Metrics) ObserveQueueWaitTime(queue Queue, duration time.Duration, agentName string) {
	m.queueWaitTime.WithLabelValues(string(queue), agentName).Observe(duration.Seconds())
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

// IncrementLengthUnverified records a payload the completeness gate could not
// be applied to, which HTTP/2, chunked encoding and transparent decompression
// all produce.
func (m *Metrics) IncrementLengthUnverified(queue Queue, agentName string) {
	m.lengthUnverified.WithLabelValues(string(queue), agentName).Inc()
}

// IncrementFlightAbandoned records a caller that stopped waiting on another
// node's upload. A rising count means the fleet is collapsing less work than it
// could, which is a symptom of slow nodes rather than of this agent.
func (m *Metrics) IncrementFlightAbandoned(queue Queue, agentName string) {
	m.flightAbandoned.WithLabelValues(string(queue), agentName).Inc()
}

// AddFetchedBytes records raw bytes read from a node, whether or not they were
// stored.
func (m *Metrics) AddFetchedBytes(queue Queue, agentName string, bytes int64) {
	if bytes <= 0 {
		return
	}

	m.fetchBytes.WithLabelValues(string(queue), agentName).Add(float64(bytes))
}

// AddStoredBytes records compressed bytes handed to the store. A payload that
// was hashed against one somebody else had already stored contributes nothing,
// which is the whole point of measuring it separately.
func (m *Metrics) AddStoredBytes(queue Queue, agentName string, bytes int64) {
	if bytes <= 0 {
		return
	}

	m.storedBytes.WithLabelValues(string(queue), agentName).Add(float64(bytes))
}

func (m *Metrics) ServeMetrics(ctx context.Context, addr string) {
	observability.StartMetricsServer(ctx, addr)
}

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
}

type Queue string

var (
	BeaconStateQueue         Queue = "beacon_state"
	BeaconBlockQueue         Queue = "beacon_block"
	BeaconBadBlockQueue      Queue = "beacon_bad_block"
	BeaconBadBlobQueue       Queue = "beacon_bad_blob"
	ExecutionBlockTraceQueue Queue = "execution_block_trace"
	ExecutionBadBlockQueue   Queue = "execution_bad_block"
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
			}, []string{"queue", "agent"}),
			queueItemProcessingTime: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "queue_item_processing_time_seconds",
				Help:      "The time it takes to process an item from the queue",
				Buckets:   prometheus.LinearBuckets(0, 3, 10),
			}, []string{"queue", "agent"}),
			itemExported: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "item_exported",
				Help:      "The number of items exported",
			}, []string{"queue", "agent"}),
			queueItemSkipped: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "queue_item_skipped",
				Help:      "The number of items skipped",
			}, []string{"queue", "agent"}),
		}

		prometheus.MustRegister(metricsInstance.queueSize)
		prometheus.MustRegister(metricsInstance.queueItemProcessingTime)
		prometheus.MustRegister(metricsInstance.itemExported)
		prometheus.MustRegister(metricsInstance.queueItemSkipped)
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

func (m *Metrics) ServeMetrics(ctx context.Context, addr string) {
	observability.StartMetricsServer(ctx, addr)
}

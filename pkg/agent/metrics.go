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
	serveOnce               sync.Once
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

func GetMetricsInstance(namespace string, agentName string) *Metrics {
	once.Do(func() {
		metricsInstance = &Metrics{
			queueSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "queue_size",
				Help:      "The size of the queue",
				ConstLabels: prometheus.Labels{
					"agent": agentName,
				},
			}, []string{"queue"}),
			queueItemProcessingTime: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "queue_item_processing_time_seconds",
				Help:      "The time it takes to process an item from the queue",
				Buckets:   prometheus.LinearBuckets(0, 3, 10),
				ConstLabels: prometheus.Labels{
					"agent": agentName,
				},
			}, []string{"queue"}),
			itemExported: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "item_exported",
				Help:      "The number of items exported",
				ConstLabels: prometheus.Labels{
					"agent": agentName,
				},
			}, []string{"queue"}),
		}

		prometheus.MustRegister(metricsInstance.queueSize)
		prometheus.MustRegister(metricsInstance.queueItemProcessingTime)
		prometheus.MustRegister(metricsInstance.itemExported)
	})

	return metricsInstance
}

func (m *Metrics) SetQueueSize(queue Queue, count int) {
	m.queueSize.WithLabelValues(string(queue)).Set(float64(count))
}

func (m *Metrics) ObserveQueueItemProcessingTime(queue Queue, duration time.Duration) {
	m.queueItemProcessingTime.WithLabelValues(string(queue)).Observe(duration.Seconds())
}

func (m *Metrics) IncrementItemExported(queue Queue) {
	m.itemExported.WithLabelValues(string(queue)).Inc()
}

func (m *Metrics) ServeMetrics(ctx context.Context, addr string) {
	observability.StartMetricsServer(ctx, addr)
}

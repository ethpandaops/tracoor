package agent

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	network string

	queueSize               *prometheus.GaugeVec
	queueItemProcessingTime *prometheus.HistogramVec
	itemExported            *prometheus.CounterVec
}

type Queue string

var (
	BeaconStateQueue         Queue = "beacon_state"
	ExecutionBlockTraceQueue Queue = "execution_block_trace"
	ExecutionBadBlockQueue   Queue = "execution_bad_block"
)

func NewMetrics(namespace, network string) *Metrics {
	m := &Metrics{
		network: network,
		queueSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "queue_size",
			Help:      "The size of the queue",
		}, []string{"queue", "network"}),
		queueItemProcessingTime: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "queue_item_processing_time_seconds",
			Help:      "The time it takes to process an item from the queue",
			Buckets:   prometheus.LinearBuckets(0, 3, 10),
		}, []string{"queue", "network"}),
		itemExported: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "item_exported",
			Help:      "The number of items exported",
		}, []string{"queue", "network"}),
	}

	prometheus.MustRegister(m.queueSize)
	prometheus.MustRegister(m.queueItemProcessingTime)
	prometheus.MustRegister(m.itemExported)

	return m
}

func (m *Metrics) SetQueueSize(queue Queue, count int) {
	m.queueSize.WithLabelValues(string(queue), m.network).Set(float64(count))
}

func (m *Metrics) ObserveQueueItemProcessingTime(queue Queue, duration time.Duration) {
	m.queueItemProcessingTime.WithLabelValues(string(queue), m.network).Observe(duration.Seconds())
}

func (m *Metrics) IncrementItemExported(queue Queue) {
	m.itemExported.WithLabelValues(string(queue), m.network).Inc()
}

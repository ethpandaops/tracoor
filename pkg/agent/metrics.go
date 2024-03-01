package agent

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	queueSize               *prometheus.GaugeVec
	queueItemProcessingTime *prometheus.HistogramVec
	itemExported            *prometheus.CounterVec
}

type Queue string

var (
	BeaconStateQueue         Queue = "beacon_state"
	BeaconBlockQueue         Queue = "beacon_block"
	BeaconBadBlockQueue      Queue = "beacon_bad_block"
	ExecutionBlockTraceQueue Queue = "execution_block_trace"
	ExecutionBadBlockQueue   Queue = "execution_bad_block"
)

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		queueSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "queue_size",
			Help:      "The size of the queue",
		}, []string{"queue"}),
		queueItemProcessingTime: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "queue_item_processing_time_seconds",
			Help:      "The time it takes to process an item from the queue",
			Buckets:   prometheus.LinearBuckets(0, 3, 10),
		}, []string{"queue"}),
		itemExported: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "item_exported",
			Help:      "The number of items exported",
		}, []string{"queue"}),
	}

	prometheus.MustRegister(m.queueSize)
	prometheus.MustRegister(m.queueItemProcessingTime)

	return m
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

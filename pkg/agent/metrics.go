package agent

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	queueSize               *prometheus.GaugeVec
	queueItemProcessingTime *prometheus.HistogramVec
}

type Queue string

var (
	BeaconStateQueue         Queue = "beacon_state"
	ExecutionBlockTraceQueue Queue = "execution_block_trace"
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

package store

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type BasicMetrics struct {
	namespace string

	info                        *prometheus.GaugeVec
	itemsAdded                  *prometheus.CounterVec
	itemsAddedBytes             *prometheus.HistogramVec
	itemsAddedBytesUncompressed *prometheus.HistogramVec
	itemsRemoved                *prometheus.CounterVec
	itemsRetreived              *prometheus.CounterVec
	itemsUrlsRetreived          *prometheus.CounterVec
	itemsStored                 *prometheus.GaugeVec

	cacheHit  *prometheus.CounterVec
	cacheMiss *prometheus.CounterVec
}

var (
	instance *BasicMetrics
	once     sync.Once
)

func GetBasicMetricsInstance(namespace, storeType string, enabled bool) *BasicMetrics {
	once.Do(func() {
		instance = &BasicMetrics{
			namespace: namespace,

			info: prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "info",
				Help:      "Information about the implementation of the store",
			}, []string{"implementation"}),

			itemsAdded: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "items_added_count",
				Help:      "Number of items added to the store",
			}, []string{"type"}),
			itemsRemoved: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "items_removed_count",
				Help:      "Number of items removed from the store",
			}, []string{"type"}),
			itemsRetreived: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "items_retrieved_count",
				Help:      "Number of items retreived from the store",
			}, []string{"type"}),
			itemsStored: prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "items_stored_total",
				Help:      "Number of items stored in the store",
			}, []string{"type"}),
			itemsUrlsRetreived: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "items_urls_retrieved_count",
				Help:      "Number of items URLs retreived",
			}, []string{"type"}),
			cacheHit: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_hit_count",
				Help:      "Number of cache hits",
			}, []string{"type"}),
			cacheMiss: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_miss_count",
				Help:      "Number of cache misses",
			}, []string{"type"}),
			itemsAddedBytes: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "items_added_bytes",
				Help:      "Size of items added to the store",
				Buckets:   prometheus.ExponentialBuckets(1024000, 2, 13),
			}, []string{"type"}),
			itemsAddedBytesUncompressed: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "items_added_uncompressed_bytes",
				Help:      "Size of items added to the store when uncompressed",
				Buckets:   prometheus.ExponentialBuckets(1024000, 2, 13),
			}, []string{"type"}),
		}

		if enabled {
			prometheus.MustRegister(instance.info)
			prometheus.MustRegister(instance.itemsAdded)
			prometheus.MustRegister(instance.itemsAddedBytes)
			prometheus.MustRegister(instance.itemsAddedBytesUncompressed)
			prometheus.MustRegister(instance.itemsRemoved)
			prometheus.MustRegister(instance.itemsRetreived)
			prometheus.MustRegister(instance.itemsUrlsRetreived)
			prometheus.MustRegister(instance.itemsStored)
			prometheus.MustRegister(instance.cacheHit)
			prometheus.MustRegister(instance.cacheMiss)
		}

		instance.info.WithLabelValues(storeType).Set(1)
	})

	return instance
}

func (m *BasicMetrics) ObserveItemAdded(itemType string) {
	m.itemsAdded.WithLabelValues(itemType).Inc()
}

func (m *BasicMetrics) ObserveItemAddedBytes(itemType string, size int) {
	m.itemsAddedBytes.WithLabelValues(itemType).Observe(float64(size))
}

func (m *BasicMetrics) ObserveItemAddedBytesUncompressed(itemType string, size int) {
	m.itemsAddedBytesUncompressed.WithLabelValues(itemType).Observe(float64(size))
}

func (m *BasicMetrics) ObserveItemRemoved(itemType string) {
	m.itemsRemoved.WithLabelValues(itemType).Inc()
}

func (m *BasicMetrics) ObserveItemRetreived(itemType string) {
	m.itemsRetreived.WithLabelValues(itemType).Inc()
}

func (m *BasicMetrics) ObserveItemURLRetreived(itemType string) {
	m.itemsUrlsRetreived.WithLabelValues(itemType).Inc()
}

func (m *BasicMetrics) ObserveItemStored(itemType string, count int) {
	m.itemsStored.WithLabelValues(itemType).Set(float64(count))
}

func (m *BasicMetrics) ObserveCacheHit(itemType string) {
	m.cacheHit.WithLabelValues(itemType).Inc()
}

func (m *BasicMetrics) ObserveCacheMiss(itemType string) {
	m.cacheMiss.WithLabelValues(itemType).Inc()
}

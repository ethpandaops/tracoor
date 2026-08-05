package store

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type BasicMetrics struct {
	namespace string

	// registered records whether the collectors reached the prometheus
	// registry, so a namespace first created with metrics disabled can
	// still be registered later by a store that wants them.
	registered bool

	info               *prometheus.GaugeVec
	itemsAdded         *prometheus.CounterVec
	itemsAddedBytes    *prometheus.HistogramVec
	itemsRemoved       *prometheus.CounterVec
	itemsRetreived     *prometheus.CounterVec
	itemsUrlsRetreived *prometheus.CounterVec
	itemsStored        *prometheus.GaugeVec

	cacheHit  *prometheus.CounterVec
	cacheMiss *prometheus.CounterVec
}

// instances is one BasicMetrics per namespace. A process can run more than
// one store - the capture buffer and the promotion corpus, for instance - and
// a single global instance would silently attribute every store's activity to
// whichever one was constructed first.
var (
	instances = map[string]*BasicMetrics{}
	mu        sync.Mutex
)

func GetBasicMetricsInstance(namespace, storeType string, enabled bool) *BasicMetrics {
	mu.Lock()
	defer mu.Unlock()

	if existing, ok := instances[namespace]; ok {
		// A namespace whose first store disabled metrics would otherwise
		// stay unexported for every later store that wants them.
		existing.register(enabled)
		existing.info.WithLabelValues(storeType).Set(1)

		return existing
	}

	instance := &BasicMetrics{
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
		}, []string{labelType}),
		itemsRemoved: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "items_removed_count",
			Help:      "Number of items removed from the store",
		}, []string{labelType}),
		itemsRetreived: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "items_retrieved_count",
			Help:      "Number of items retreived from the store",
		}, []string{labelType}),
		itemsStored: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "items_stored_total",
			Help:      "Number of items stored in the store",
		}, []string{labelType}),
		itemsUrlsRetreived: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "items_urls_retrieved_count",
			Help:      "Number of items URLs retreived",
		}, []string{labelType}),
		cacheHit: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "cache_hit_count",
			Help:      "Number of cache hits",
		}, []string{labelType}),
		cacheMiss: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "cache_miss_count",
			Help:      "Number of cache misses",
		}, []string{labelType}),
		itemsAddedBytes: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "items_added_bytes",
			Help:      "Size of items added to the store",
			Buckets:   prometheus.ExponentialBuckets(1024000, 2, 13),
		}, []string{labelType}),
	}

	instance.register(enabled)
	instance.info.WithLabelValues(storeType).Set(1)

	instances[namespace] = instance

	return instance
}

// register exports the collectors, at most once. Callers hold mu.
func (m *BasicMetrics) register(enabled bool) {
	if !enabled || m.registered {
		return
	}

	prometheus.MustRegister(m.info)
	prometheus.MustRegister(m.itemsAdded)
	prometheus.MustRegister(m.itemsAddedBytes)
	prometheus.MustRegister(m.itemsRemoved)
	prometheus.MustRegister(m.itemsRetreived)
	prometheus.MustRegister(m.itemsUrlsRetreived)
	prometheus.MustRegister(m.itemsStored)
	prometheus.MustRegister(m.cacheHit)
	prometheus.MustRegister(m.cacheMiss)

	m.registered = true
}

func (m *BasicMetrics) ObserveItemAdded(itemType string) {
	m.itemsAdded.WithLabelValues(itemType).Inc()
}

func (m *BasicMetrics) ObserveItemAddedBytes(itemType string, size int) {
	m.itemsAddedBytes.WithLabelValues(itemType).Observe(float64(size))
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

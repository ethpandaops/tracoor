package observability

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var (
	metricsServerInstance *http.Server
	once                  sync.Once
)

// StartMetricsServer starts a Prometheus metrics server if it hasn't been started already.
func StartMetricsServer(ctx context.Context, addr string) {
	once.Do(func() {
		if metricsServerInstance != nil {
			return
		}

		sm := http.NewServeMux()
		sm.Handle("/metrics", promhttp.Handler())

		metricsServerInstance = &http.Server{
			Addr:              addr,
			ReadHeaderTimeout: 15 * time.Second,
			Handler:           sm,
		}

		go func() {
			logrus.Infof("Starting metrics server on %s", addr)

			if err := metricsServerInstance.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logrus.WithError(err).Fatal("Failed to start metrics server")
			}
		}()
	})
}

// StopMetricsServer gracefully stops the Prometheus metrics server.
func StopMetricsServer(ctx context.Context) error {
	if metricsServerInstance != nil {
		if err := metricsServerInstance.Shutdown(ctx); err != nil {
			logrus.WithError(err).Error("Failed to stop metrics server")

			return err
		}
	}

	return nil
}

// RegisterMetrics registers the provided Prometheus collectors.
func RegisterMetrics(collectors ...prometheus.Collector) {
	for _, collector := range collectors {
		prometheus.MustRegister(collector)
	}
}

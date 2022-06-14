package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tereus-project/tereus-transpiler-std/env"
	"github.com/tereus-project/tereus-transpiler-std/submission"
)

type MetricsService struct {
	transpilingDurationHistogram *prometheus.HistogramVec
	server                       *http.Server
}

func NewMetricsService(prometheusNamespace string) (*MetricsService, error) {
	config := env.GetEnv()

	transpilingDurationHistogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: prometheusNamespace,
		Name:      "transpiling_duration_seconds",
		Help:      "Histogram of transpiling duration",
	}, []string{"status"})

	err := prometheus.Register(transpilingDurationHistogram)
	if err != nil {
		return nil, err
	}

	http.Handle("/metrics", promhttp.Handler())

	return &MetricsService{
		transpilingDurationHistogram: transpilingDurationHistogram,
		server: &http.Server{
			Addr: fmt.Sprintf(":%s", config.MetricsPort),
		},
	}, nil
}

func (m *MetricsService) Listen() error {
	if err := m.server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (m *MetricsService) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return m.server.Shutdown(ctx)
}

func (m *MetricsService) ObserveTranspilingDuration(status submission.SubmissionStatus, startTime time.Time) {
	m.transpilingDurationHistogram.WithLabelValues(string(status)).Observe(time.Since(startTime).Seconds())
}

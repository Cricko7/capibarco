// Package metrics exposes Prometheus metrics.
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics groups matching-service Prometheus collectors.
type Metrics struct {
	GRPCRequests   *prometheus.CounterVec
	GRPCDuration   *prometheus.HistogramVec
	HTTPRequests   *prometheus.CounterVec
	KafkaConsumed  *prometheus.CounterVec
	KafkaPublished *prometheus.CounterVec
}

// New creates and registers service metrics.
func New(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		GRPCRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matching_grpc_requests_total",
			Help: "Total gRPC requests.",
		}, []string{"method", "code"}),
		GRPCDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "matching_grpc_request_duration_seconds",
			Help:    "gRPC request duration.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method"}),
		HTTPRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matching_http_requests_total",
			Help: "Total HTTP requests.",
		}, []string{"path", "method", "code"}),
		KafkaConsumed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matching_kafka_consumed_total",
			Help: "Total consumed Kafka messages.",
		}, []string{"topic", "result"}),
		KafkaPublished: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matching_kafka_published_total",
			Help: "Total published Kafka messages.",
		}, []string{"topic", "result"}),
	}
	reg.MustRegister(m.GRPCRequests, m.GRPCDuration, m.HTTPRequests, m.KafkaConsumed, m.KafkaPublished)
	return m
}

// ObserveGRPC records a gRPC request.
func (m *Metrics) ObserveGRPC(method string, code string, started time.Time) {
	if m == nil {
		return
	}
	m.GRPCRequests.WithLabelValues(method, code).Inc()
	m.GRPCDuration.WithLabelValues(method).Observe(time.Since(started).Seconds())
}

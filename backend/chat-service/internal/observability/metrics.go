package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Metrics groups Prometheus collectors used by the service.
type Metrics struct {
	Registry        *prometheus.Registry
	GRPCRequests    *prometheus.CounterVec
	GRPCDuration    *prometheus.HistogramVec
	RealtimeClients prometheus.Gauge
}

// NewMetrics registers base service metrics.
func NewMetrics() *Metrics {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	metrics := &Metrics{
		Registry: registry,
		GRPCRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "chat_grpc_requests_total",
			Help: "Total gRPC requests handled by chat-service.",
		}, []string{"method", "code"}),
		GRPCDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "chat_grpc_request_duration_seconds",
			Help:    "gRPC request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method"}),
		RealtimeClients: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "chat_realtime_clients",
			Help: "Number of active realtime chat streams.",
		}),
	}
	registry.MustRegister(metrics.GRPCRequests, metrics.GRPCDuration, metrics.RealtimeClients)
	return metrics
}

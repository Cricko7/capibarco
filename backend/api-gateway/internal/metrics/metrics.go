// Package metrics contains Prometheus collectors for api-gateway.
package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics groups gateway collectors.
type Metrics struct {
	Requests        *prometheus.CounterVec
	RequestLatency  *prometheus.HistogramVec
	RateLimitHits   *prometheus.CounterVec
	DownstreamCalls *prometheus.CounterVec
}

// New creates and registers gateway metrics.
func New(registry *prometheus.Registry) *Metrics {
	m := &Metrics{
		Requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "api_gateway_http_requests_total",
			Help: "Total HTTP requests handled by api-gateway.",
		}, []string{"method", "route", "status"}),
		RequestLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "api_gateway_http_request_duration_seconds",
			Help:    "HTTP request duration.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route"}),
		RateLimitHits: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "api_gateway_rate_limit_hits_total",
			Help: "Total rejected requests due to rate limiting.",
		}, []string{"bucket"}),
		DownstreamCalls: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "api_gateway_downstream_calls_total",
			Help: "Total downstream gRPC calls.",
		}, []string{"service", "method", "status"}),
	}
	registry.MustRegister(m.Requests, m.RequestLatency, m.RateLimitHits, m.DownstreamCalls)
	return m
}

// ObserveHTTP records an HTTP request.
func (m *Metrics) ObserveHTTP(method string, route string, status int, duration time.Duration) {
	m.Requests.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
	m.RequestLatency.WithLabelValues(method, route).Observe(duration.Seconds())
}

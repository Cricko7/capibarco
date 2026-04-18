package observability

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	EventsIngestedTotal prometheus.Counter
	EventsDuplicate     prometheus.Counter
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		HTTPRequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "analytics",
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		}, []string{"method", "route", "status"}),
		HTTPRequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "analytics",
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request latency",
			Buckets:   prometheus.DefBuckets,
		}, []string{"method", "route"}),
		EventsIngestedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "analytics",
			Name:      "events_ingested_total",
			Help:      "Total ingested events",
		}),
		EventsDuplicate: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "analytics",
			Name:      "events_duplicate_total",
			Help:      "Total duplicate events",
		}),
	}
	reg.MustRegister(m.HTTPRequestsTotal, m.HTTPRequestDuration, m.EventsIngestedTotal, m.EventsDuplicate)
	return m
}

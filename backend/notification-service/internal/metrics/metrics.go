package metrics

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	HTTPRequests *prometheus.CounterVec
	GRPCRequests *prometheus.CounterVec
}

func New(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		HTTPRequests: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "notification_http_requests_total", Help: "HTTP requests."}, []string{"path", "method", "status"}),
		GRPCRequests: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "notification_grpc_requests_total", Help: "gRPC requests."}, []string{"method", "status"}),
	}
	if reg != nil {
		reg.MustRegister(m.HTTPRequests, m.GRPCRequests)
	}
	return m
}

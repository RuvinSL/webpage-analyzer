package metrics

import (
	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusCollector implements metrics collection using Prometheus
type PrometheusCollector struct {
	serviceName string

	// HTTP metrics
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestDuration  *prometheus.HistogramVec
	httpRequestsInFlight prometheus.Gauge

	// Business metrics
	analysisTotal     *prometheus.CounterVec
	analysisDuration  *prometheus.HistogramVec
	linkChecksTotal   *prometheus.CounterVec
	linkCheckDuration *prometheus.HistogramVec
}

// NewPrometheusCollector creates a new Prometheus metrics collector
func NewPrometheusCollector(serviceName string) *PrometheusCollector {
	return &PrometheusCollector{
		serviceName: serviceName,

		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
			[]string{"method", "path", "status"},
		),

		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "http_request_duration_seconds",
				Help: "HTTP request duration in seconds",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "status"},
		),

		httpRequestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Number of HTTP requests currently being processed",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
		),

		analysisTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "webpage_analysis_total",
				Help: "Total number of webpage analyses",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
			[]string{"status"},
		),

		analysisDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "webpage_analysis_duration_seconds",
				Help: "Webpage analysis duration in seconds",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
				Buckets: []float64{0.1, 0.5, 1, 2.5, 5, 10, 30},
			},
			[]string{"status"},
		),

		linkChecksTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "link_checks_total",
				Help: "Total number of link checks",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
			[]string{"status"},
		),

		linkCheckDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "link_check_duration_seconds",
				Help: "Link check duration in seconds",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
				Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2.5, 5},
			},
			[]string{"status"},
		),
	}
}

// GetCollectors returns all Prometheus collectors for registration
func (p *PrometheusCollector) GetCollectors() []prometheus.Collector {
	return []prometheus.Collector{
		p.httpRequestsTotal,
		p.httpRequestDuration,
		p.httpRequestsInFlight,
		p.analysisTotal,
		p.analysisDuration,
		p.linkChecksTotal,
		p.linkCheckDuration,
	}
}

// RecordRequest records HTTP request metrics
func (p *PrometheusCollector) RecordRequest(method, path string, statusCode int, duration float64) {
	status := statusCodeToString(statusCode)

	p.httpRequestsTotal.WithLabelValues(method, path, status).Inc()
	p.httpRequestDuration.WithLabelValues(method, path, status).Observe(duration)
}

// RecordAnalysis records webpage analysis metrics
func (p *PrometheusCollector) RecordAnalysis(success bool, duration float64) {
	status := "success"
	if !success {
		status = "failure"
	}

	p.analysisTotal.WithLabelValues(status).Inc()
	p.analysisDuration.WithLabelValues(status).Observe(duration)
}

// RecordLinkCheck records link check metrics
func (p *PrometheusCollector) RecordLinkCheck(success bool, duration float64) {
	status := "success"
	if !success {
		status = "failure"
	}

	p.linkChecksTotal.WithLabelValues(status).Inc()
	p.linkCheckDuration.WithLabelValues(status).Observe(duration)
}

// IncRequestsInFlight increments the in-flight requests gauge
func (p *PrometheusCollector) IncRequestsInFlight() {
	p.httpRequestsInFlight.Inc()
}

// DecRequestsInFlight decrements the in-flight requests gauge
func (p *PrometheusCollector) DecRequestsInFlight() {
	p.httpRequestsInFlight.Dec()
}

// statusCodeToString converts HTTP status code to string category
func statusCodeToString(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}

// Collector interface implementation
type Collector interface {
	RecordRequest(method, path string, statusCode int, duration float64)
	RecordAnalysis(success bool, duration float64)
	RecordLinkCheck(success bool, duration float64)
	GetCollectors() []prometheus.Collector
}

// Ensure PrometheusCollector implements interfaces.MetricsCollector
var _ interfaces.MetricsCollector = (*PrometheusCollector)(nil)

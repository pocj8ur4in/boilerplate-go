package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	// bucketStart is the start value for the bucket.
	bucketStart = 100.0

	// bucketFactor is the factor for the bucket.
	bucketFactor = 10.0

	// bucketCount is the count for the bucket.
	bucketCount = 8
)

// metricsCollector holds all prometheus metrics collectors.
type metricsCollector struct {
	// requestsTotal is the total number of requests.
	requestsTotal *prometheus.CounterVec

	// requestDuration is the duration of the request.
	requestDuration *prometheus.HistogramVec

	// requestSize is the size of the request.
	requestSize *prometheus.HistogramVec

	// responseSize is the size of the response.
	responseSize *prometheus.HistogramVec

	// requestsInFlight is the number of requests in flight.
	requestsInFlight prometheus.Gauge
}

// MetricsConfig represents configuration for metrics middleware.
type MetricsConfig struct {
	// Enabled is whether metrics collection is enabled.
	Enabled *bool `json:"enabled"`

	// Path is the path for metrics endpoint.
	Path *string `json:"path"`

	// ExcludePaths is a list of paths to exclude from metrics.
	ExcludePaths []string `json:"exclude_paths"`
}

// SetDefault sets default values.
func (c *MetricsConfig) SetDefault() {
	if c.Enabled == nil {
		c.Enabled = &[]bool{true}[0]
	}

	if c.Path == nil {
		c.Path = &[]string{"/metrics"}[0]
	}

	if c.ExcludePaths == nil {
		c.ExcludePaths = []string{"/health", "/status"}
	}
}

// newMetricsCollector creates a new metrics collector.
func newMetricsCollector(registry prometheus.Registerer) *metricsCollector {
	return &metricsCollector{
		requestsTotal: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		requestDuration: promauto.With(registry).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "Duration of HTTP requests in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "status"},
		),
		requestSize: promauto.With(registry).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_size_bytes",
				Help:    "Size of HTTP requests in bytes",
				Buckets: prometheus.ExponentialBuckets(bucketStart, bucketFactor, bucketCount),
			},
			[]string{"method", "path"},
		),
		responseSize: promauto.With(registry).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_size_bytes",
				Help:    "Size of HTTP responses in bytes",
				Buckets: prometheus.ExponentialBuckets(bucketStart, bucketFactor, bucketCount),
			},
			[]string{"method", "path", "status"},
		),
		requestsInFlight: promauto.With(registry).NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Number of HTTP requests currently being processed",
			},
		),
	}
}

// Metrics is a middleware that collects Prometheus metrics.
func Metrics(config *MetricsConfig, registry prometheus.Registerer) func(next http.Handler) http.Handler {
	// set default config
	if config == nil {
		config = &MetricsConfig{}
	}

	config.SetDefault()

	// use default registry if none provided
	if registry == nil {
		registry = prometheus.DefaultRegisterer
	}

	// create collector instance for this middleware
	collector := newMetricsCollector(registry)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if shouldSkipMetrics(config, request) {
				next.ServeHTTP(writer, request)

				return
			}

			processWithMetrics(next, writer, request, collector)
		})
	}
}

// shouldSkipMetrics checks if metrics should be skipped for this request.
func shouldSkipMetrics(config *MetricsConfig, request *http.Request) bool {
	if !*config.Enabled {
		return true
	}

	path := request.URL.Path
	if path == *config.Path {
		return true
	}

	for _, excludePath := range config.ExcludePaths {
		if path == excludePath {
			return true
		}
	}

	return false
}

// processWithMetrics processes the request and collects metrics.
func processWithMetrics(
	next http.Handler,
	writer http.ResponseWriter,
	request *http.Request,
	collector *metricsCollector,
) {
	collector.requestsInFlight.Inc()
	defer collector.requestsInFlight.Dec()

	recordRequestSize(collector, request)

	start := time.Now()
	wrappedWriter := middleware.NewWrapResponseWriter(writer, request.ProtoMajor)

	next.ServeHTTP(wrappedWriter, request)

	recordRequestMetrics(collector, request, wrappedWriter, time.Since(start))
}

// recordRequestSize records the size of the request.
func recordRequestSize(collector *metricsCollector, request *http.Request) {
	if request.ContentLength > 0 {
		collector.requestSize.WithLabelValues(
			request.Method,
			request.URL.Path,
		).Observe(float64(request.ContentLength))
	}
}

// recordRequestMetrics records request metrics after processing.
func recordRequestMetrics(
	collector *metricsCollector,
	request *http.Request,
	wrappedWriter middleware.WrapResponseWriter,
	duration time.Duration,
) {
	status := strconv.Itoa(wrappedWriter.Status())

	collector.requestsTotal.WithLabelValues(
		request.Method,
		request.URL.Path,
		status,
	).Inc()

	collector.requestDuration.WithLabelValues(
		request.Method,
		request.URL.Path,
		status,
	).Observe(duration.Seconds())

	if wrappedWriter.BytesWritten() > 0 {
		collector.responseSize.WithLabelValues(
			request.Method,
			request.URL.Path,
			status,
		).Observe(float64(wrappedWriter.BytesWritten()))
	}
}

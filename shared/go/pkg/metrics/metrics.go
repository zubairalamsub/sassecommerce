// Package metrics provides a shared Prometheus instrumentation layer for all
// Go services in the platform. It exposes a Gin-compatible middleware that
// records standard HTTP RED-style metrics (Rate, Errors, Duration) plus an
// in-flight gauge, and a Handler() func returning the Prometheus HTTP handler
// suitable for serving the /metrics endpoint.
//
// Typical usage:
//
//	metrics.Register("user-service")
//	router.Use(metrics.Middleware("user-service"))
//	router.GET("/metrics", gin.WrapH(metrics.Handler()))
package metrics

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// defaultBuckets are the histogram buckets (seconds) used for the
// http_request_duration_seconds metric — covers fast in-process work
// through multi-second slow paths.
var defaultBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5}

var (
	registerOnce sync.Once

	httpRequestsTotal *prometheus.CounterVec
	httpRequestDur    *prometheus.HistogramVec
	httpInFlight      *prometheus.GaugeVec
)

// Register initialises the default Prometheus registry with standard
// process/Go runtime collectors and the HTTP instrumentation metrics
// scoped by the given service name as a baseline (const) label.
//
// It is safe to call multiple times; subsequent calls are no-ops.
func Register(service string) {
	registerOnce.Do(func() {
		// Standard process + Go runtime collectors. The default registry
		// usually contains these via init(), but we re-register defensively
		// to handle programs that build their own registry. Already-
		// registered errors are swallowed.
		_ = prometheus.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
		_ = prometheus.Register(collectors.NewGoCollector())

		httpRequestsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "http_requests_total",
				Help:        "Total number of HTTP requests handled.",
				ConstLabels: prometheus.Labels{"service": service},
			},
			[]string{"method", "path", "status"},
		)

		httpRequestDur = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "http_request_duration_seconds",
				Help:        "Duration of HTTP requests in seconds.",
				Buckets:     defaultBuckets,
				ConstLabels: prometheus.Labels{"service": service},
			},
			[]string{"method", "path"},
		)

		httpInFlight = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "http_requests_in_flight",
				Help:        "Number of HTTP requests currently being served.",
				ConstLabels: prometheus.Labels{"service": service},
			},
			[]string{},
		)

		prometheus.MustRegister(httpRequestsTotal, httpRequestDur, httpInFlight)
	})
}

// Middleware returns a Gin middleware that records request count, duration
// and in-flight counts for every request. Call Register(service) once at
// startup; Middleware will call it for you if you forget.
//
// The middleware uses c.FullPath() for the path label, which collapses
// dynamic segments like /users/:id into the route template. This is
// important to avoid label cardinality blow-ups from path parameters.
func Middleware(service string) gin.HandlerFunc {
	Register(service)

	return func(c *gin.Context) {
		// Skip self-instrumentation of the metrics endpoint to keep the
		// scrape from polluting its own histograms.
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()

		httpInFlight.WithLabelValues().Inc()
		defer httpInFlight.WithLabelValues().Dec()

		c.Next()

		// Use the matched route template so cardinality stays bounded.
		// Fall back to a constant for unmatched routes (404s) — we still
		// want to count those without leaking arbitrary paths.
		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}

		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method

		httpRequestsTotal.WithLabelValues(method, path, status).Inc()
		httpRequestDur.WithLabelValues(method, path).Observe(time.Since(start).Seconds())
	}
}

// Handler returns the Prometheus HTTP handler for the default registry,
// suitable for mounting at /metrics:
//
//	router.GET("/metrics", gin.WrapH(metrics.Handler()))
func Handler() http.Handler {
	return promhttp.Handler()
}

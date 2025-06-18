package main

import (
	"strconv"
	"time"

	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP request metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "handler", "code"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "handler", "code"},
	)

	httpRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "Size of HTTP requests in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "handler"},
	)

	httpResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "Size of HTTP responses in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "handler", "code"},
	)

	// Business logic metrics
	stockLookups = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stock_lookups_total",
			Help: "Total number of stock symbol lookups",
		},
		[]string{"symbol", "status"},
	)

	cryptoLookups = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "crypto_lookups_total",
			Help: "Total number of crypto symbol lookups",
		},
		[]string{"symbol", "status"},
	)

	watchlistOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "watchlist_operations_total",
			Help: "Total number of watchlist operations",
		},
		[]string{"operation", "status"},
	)
)

// responseWriter wraps http.ResponseWriter to capture status code and response size
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	responseSize int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.responseSize += size
	return size, err
}

// PrometheusMiddleware creates HTTP metrics middleware
func PrometheusMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Get route pattern for better labeling
			route := mux.CurrentRoute(r)
			handler := "unknown"
			if route != nil {
				if pattern, err := route.GetPathTemplate(); err == nil {
					handler = pattern
				}
			}

			// Wrap response writer
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     200, // default status code
			}

			// Process request
			next.ServeHTTP(rw, r)

			// Record metrics
			duration := time.Since(start).Seconds()
			statusCode := strconv.Itoa(rw.statusCode)
			method := r.Method

			// HTTP metrics
			httpRequestsTotal.WithLabelValues(method, handler, statusCode).Inc()
			httpRequestDuration.WithLabelValues(method, handler, statusCode).Observe(duration)

			if r.ContentLength > 0 {
				httpRequestSize.WithLabelValues(method, handler).Observe(float64(r.ContentLength))
			}

			if rw.responseSize > 0 {
				httpResponseSize.WithLabelValues(method, handler, statusCode).Observe(float64(rw.responseSize))
			}
		})
	}
}

// RecordStockLookup records a stock lookup metric
func RecordStockLookup(symbol, status string) {
	stockLookups.WithLabelValues(symbol, status).Inc()
}

// RecordCryptoLookup records a crypto lookup metric
func RecordCryptoLookup(symbol, status string) {
	cryptoLookups.WithLabelValues(symbol, status).Inc()
}

// RecordWatchlistOperation records a watchlist operation metric
func RecordWatchlistOperation(operation, status string) {
	watchlistOperations.WithLabelValues(operation, status).Inc()
}

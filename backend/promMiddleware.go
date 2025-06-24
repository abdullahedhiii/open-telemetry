package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	httpRequestCounter  metric.Int64Counter
	httpRequestDuration metric.Float64Histogram
	httpRequestSize     metric.Float64Histogram
	httpResponseSize    metric.Float64Histogram
)

// InitHTTPMetrics initializes all HTTP metrics instruments
func InitHTTPMetrics() error {
	meter := otel.Meter("stock-tracker-service/http")

	var err error

	if httpRequestCounter, err = meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
	); err != nil {
		return err
	}

	if httpRequestDuration, err = meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("Duration of HTTP requests in seconds"),
	); err != nil {
		return err
	}

	if httpRequestSize, err = meter.Float64Histogram(
		"http_request_size_bytes",
		metric.WithDescription("Size of HTTP requests in bytes"),
	); err != nil {
		return err
	}

	if httpResponseSize, err = meter.Float64Histogram(
		"http_response_size_bytes",
		metric.WithDescription("Size of HTTP responses in bytes"),
	); err != nil {
		return err
	}

	return nil
}

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

func OpenTelemetryMetricsMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Capture route pattern
			handler := "unknown"
			if route := mux.CurrentRoute(r); route != nil {
				if pattern, err := route.GetPathTemplate(); err == nil {
					handler = pattern
				}
			}

			// Wrap response writer
			rw := &responseWriter{ResponseWriter: w, statusCode: 200}

			// Process request
			next.ServeHTTP(rw, r)

			// Collect metrics
			duration := time.Since(start).Seconds()
			statusCode := strconv.Itoa(rw.statusCode)
			method := r.Method
			ctx := r.Context()

			attrs := []attribute.KeyValue{
				attribute.String("http.method", method),
				attribute.String("http.route", handler),
				attribute.String("http.status_code", statusCode),
			}

			httpRequestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
			httpRequestDuration.Record(ctx, duration, metric.WithAttributes(attrs...))

			if r.ContentLength > 0 {
				httpRequestSize.Record(ctx, float64(r.ContentLength), metric.WithAttributes(attrs...))
			}

			if rw.responseSize > 0 {
				httpResponseSize.Record(ctx, float64(rw.responseSize), metric.WithAttributes(attrs...))
			}
		})
	}
}

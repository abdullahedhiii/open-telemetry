package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func instrumentHandler(handler http.HandlerFunc, endpoint string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		ctx, span := startSpan(r.Context(), fmt.Sprintf("HTTP %s", endpoint))
		defer span.End()

		// Add the trace context to the request
		r = r.WithContext(ctx)

		// Create a response wrapper to capture the status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Execute the handler
		handler(rw, r)

		// Record metrics with exemplar (trace ID)
		duration := time.Since(startTime).Seconds()
		apiLatencyHistogram.Record(ctx, duration,
			metric.WithAttributes(
				attribute.String("endpoint", endpoint),
				attribute.Int("status_code", rw.statusCode),
			))

		apiRequestCounter.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("endpoint", endpoint),
				attribute.Int("status_code", rw.statusCode),
			))
	}
}

// responseWriter is a wrapper for http.ResponseWriter that captures the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func main() {
	router := mux.NewRouter()
	fmt.Println("Starting..")

	// Initialize telemetry first
	initTelemetry()
	fmt.Println("Telemetry initialized")

	// Then initialize database
	initDB()
	fmt.Println("Database initialized")

	// Instrument all endpoints
	router.HandleFunc("/stocks/symbols", instrumentHandler(getAllStockSymbols, "/stocks/symbols")).Methods("GET")
	router.HandleFunc("/stocks/{symbol}", instrumentHandler(getStockData, "/stocks/{symbol}")).Methods("GET")
	router.HandleFunc("/crypto/symbols", instrumentHandler(getAllCryptoSymbols, "/crypto/symbols")).Methods("GET")
	router.HandleFunc("/crypto/{symbol}", instrumentHandler(getCryptoData, "/crypto/{symbol}")).Methods("GET")
	router.HandleFunc("/watchlist/add", instrumentHandler(addToWatchlist, "/watchlist/add")).Methods("POST")
	router.HandleFunc("/watchlist/{userId}", instrumentHandler(getWatchlist, "/watchlist/{userId}")).Methods("GET")
	router.HandleFunc("/watchlist/remove/{userId}/{symbol}",
		instrumentHandler(removeFromWatchlist, "/watchlist/remove/{userId}/{symbol}")).Methods("POST")

	fmt.Println("Server starting on :8000")
	log.Fatal(http.ListenAndServe(":8000", router))
}

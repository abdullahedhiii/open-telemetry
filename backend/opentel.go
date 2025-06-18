package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

var (
	httpRequestCount        metric.Int64Counter
	watchlistAddAttempts    metric.Int64Counter
	watchlistFailedAddCount metric.Int64Counter
	externalAPICallDuration metric.Float64Histogram
)

var (
	metricsServer  *http.Server
	tracerProvider *trace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
)

func initTelemetry() (func(), error) {
	ctx := context.Background()

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("stock-tracker-service"),
			semconv.ServiceVersionKey.String("0.1.0"),
			attribute.String("environment", "development"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint("localhost:4318"),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithURLPath("/v1/traces"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	bsp := trace.NewBatchSpanProcessor(traceExporter)

	tracerProvider = trace.NewTracerProvider(
		trace.WithResource(res),
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	promExporter, err := otelprometheus.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}

	meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(promExporter),
	)
	otel.SetMeterProvider(meterProvider)

	meter := otel.Meter("stock-tracker-service")

	httpRequestCount, err = meter.Int64Counter(
		"app.http.request_count",
		metric.WithDescription("Total number of successful HTTP requests handled by the application."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create app.http.request_count instrument: %w", err)
	}

	watchlistAddAttempts, err = meter.Int64Counter(
		"app.watchlist.add_attempts",
		metric.WithDescription("Total attempts to add an item to the watchlist."),
		metric.WithUnit("{attempt}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create app.watchlist.add_attempts instrument: %w", err)
	}

	watchlistFailedAddCount, err = meter.Int64Counter(
		"app.watchlist.failed_add_count",
		metric.WithDescription("Number of failed attempts to add an item to the watchlist."),
		metric.WithUnit("{failure}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create app.watchlist.failed_add_count instrument: %w", err)
	}

	externalAPICallDuration, err = meter.Float64Histogram(
		"app.external.api_call_duration",
		metric.WithDescription("Duration of external stock data API calls."),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.0, 5.0),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create app.external.api_call_duration instrument: %w", err)
	}
	log.Println("Application metrics instruments initialized.")

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	metricsServer = &http.Server{
		Addr:    ":2222",
		Handler: mux,
	}

	go func() {
		log.Printf("Prometheus metrics endpoint starting at %s/metrics", metricsServer.Addr)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Prometheus HTTP server error: %v", err)
		}
	}()

	log.Println("OpenTelemetry initialization complete. Traces go to OTLP, Metrics go to Prometheus endpoint.")

	return func() {

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if tracerProvider != nil {
			log.Println("Shutting down OpenTelemetry Trace Provider...")
			if err := tracerProvider.Shutdown(shutdownCtx); err != nil {
				log.Printf("Error shutting down trace provider: %v", err)
			}
		}

		if meterProvider != nil {
			log.Println("Shutting down OpenTelemetry Meter Provider...")
			if err := meterProvider.Shutdown(shutdownCtx); err != nil {
				log.Printf("Error shutting down meter provider: %v", err)
			}
		}

		if metricsServer != nil {
			log.Println("Shutting down Prometheus metrics server...")
			if err := metricsServer.Shutdown(shutdownCtx); err != nil {
				log.Printf("Error shutting down metrics server: %v", err)
			}
		}

		log.Println("OpenTelemetry shutdown completed.")
	}, nil
}

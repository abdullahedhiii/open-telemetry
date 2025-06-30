package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
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
	dbQueryCount            metric.Int64Counter
	dbQueryDuration         metric.Float64Histogram
	loginAttempts           metric.Int64Counter
	registerAttempts        metric.Int64Counter
	authDuration            metric.Float64Histogram
)

var (
	tracerProvider *trace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	Logger         *slog.Logger
)

func initTelemetry() (func(), error) {
	ctx := context.Background()

	logDir := "/fluentd/log"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log dir: %w", err)
	}
	logFile, err := os.OpenFile(logDir+"/app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	baseHandler := slog.NewJSONHandler(logFile, nil)
	otelHandler := NewOtelHandler(baseHandler)
	Logger = slog.New(otelHandler)
	// Logger = slog.New(slog.NewJSONHandler(logFile, nil))

	Logger.InfoContext(context.TODO(), "Logger initialized", "service", "otel-backend")

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			"",
			semconv.ServiceNameKey.String("stock-tracker-service"),
			semconv.ServiceVersionKey.String("0.1.0"),
			attribute.String("environment", "development"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}
	//http://otel-collector.127.0.0.1.sslip.io/v1/traces
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint("otel-collector-service:4318"),
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

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	// promExporter, err := otelprometheus.New()
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create Prometheus exporter: %w", err)
	// }
	metricExporter, _ := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint("otel-collector-service:4318"),
		otlpmetrichttp.WithInsecure(),
		otlpmetrichttp.WithURLPath("/v1/metrics"),
	)
	meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(meterProvider)

	meter := otel.Meter("stock-tracker-service")

	loginAttempts, err = meter.Int64Counter(
		"app_login_attempts",
		metric.WithDescription("Total number of login attempts made by users."),
		metric.WithUnit("{attempt}"),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create app_login_attempts instrument: %w", err)
	}

	registerAttempts, err = meter.Int64Counter(
		"app_register_attempts",
		metric.WithDescription("Total number of user registration attempts."),
		metric.WithUnit("{attempt}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create app_register_attempts instrument: %w", err)
	}
	authDuration, err = meter.Float64Histogram(
		"app_auth_duration",
		metric.WithDescription("Duration of user authentication operations."),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create app_auth_duration instrument: %w", err)
	}
	httpRequestCount, err = meter.Int64Counter(
		"app_http_request_count",
		metric.WithDescription("Total number of successful HTTP requests handled by the application."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create app.http.request_count instrument: %w", err)
	}

	dbQueryCount, err = meter.Int64Counter(
		"app_db_query_count",
		metric.WithDescription("Total number of database queries executed by the application."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create app.db.query_count instrument: %w", err)
	}

	dbQueryDuration, err = meter.Float64Histogram(
		"app_db_query_duration",
		metric.WithDescription("Duration of database queries executed by the application."),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create app.db.query_duration instrument: %w", err)
	}

	watchlistAddAttempts, err = meter.Int64Counter(
		"app_watchlist_add_attempts",
		metric.WithDescription("Total attempts to add an item to the watchlist."),
		metric.WithUnit("{attempt}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create app.watchlist.add_attempts instrument: %w", err)
	}

	watchlistFailedAddCount, err = meter.Int64Counter(
		"app_watchlist_failed_add_count",
		metric.WithDescription("Number of failed attempts to add an item to the watchlist."),
		metric.WithUnit("{failure}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create app.watchlist.failed_add_count instrument: %w", err)
	}

	externalAPICallDuration, err = meter.Float64Histogram(
		"app_external_api_call_duration",
		metric.WithDescription("Duration of external stock data API calls."),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.0, 5.0),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create app.external.api_call_duration instrument: %w", err)
	}
	log.Println("Application metrics instruments initialized.")

	// mux := http.NewServeMux()
	// mux.Handle("/metrics", promhttp.Handler())

	// metricsServer = &http.Server{
	// 	Addr:    ":2222",
	// 	Handler: mux,
	// }

	// go func() {
	// 	log.Printf("Prometheus metrics endpoint starting at %s/metrics", metricsServer.Addr)
	// 	if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
	// 		log.Printf("Prometheus HTTP server error: %v", err)
	// 	}
	// }()

	log.Println("OpenTelemetry initialization complete. Traces and metrics are exported via OTLP to the Collector.")

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

		// if metricsServer != nil {
		// 	log.Println("Shutting down Prometheus metrics server...")
		// 	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		// 		log.Printf("Error shutting down metrics server: %v", err)
		// 	}
		// }

		log.Println("OpenTelemetry shutdown completed.")
	}, nil
}

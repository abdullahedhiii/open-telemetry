package main

import (
	"context"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer trace.Tracer
	meter  metric.Meter

	// Application Metrics
	watchlistAddCounter   metric.Int64Counter
	watchlistErrorCounter metric.Int64Counter
	apiRequestCounter     metric.Int64Counter
	apiLatencyHistogram   metric.Float64Histogram
	dbOperationCounter    metric.Int64Counter
	dbLatencyHistogram    metric.Float64Histogram

	// System Metrics
	memoryUsageGauge   metric.Float64ObservableGauge
	cpuUsageGauge      metric.Float64ObservableGauge
	appGoroutinesGauge metric.Int64ObservableGauge
	gcPauseLatency     metric.Float64Histogram
)

func initTelemetry() {
	// Create resource
	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("stock-tracker"),
		semconv.ServiceVersion("0.1.0"),
		attribute.String("environment", "development"),
	)

	// Initialize tracer
	traceExporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
		stdouttrace.WithoutTimestamps(),
	)
	if err != nil {
		log.Fatal(err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(traceExporter),
		sdktrace.WithResource(resource),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	tracer = tp.Tracer("stock-tracker")

	// Create Prometheus registry and exporter
	registry := prometheus.NewRegistry()

	// Add Go collectors including GC stats
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	promExporter, err := otelprometheus.New(
		otelprometheus.WithRegisterer(registry),
		otelprometheus.WithoutUnits(),
		otelprometheus.WithoutScopeInfo(),
		otelprometheus.WithoutTargetInfo(),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize metrics with Prometheus
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(promExporter),
		sdkmetric.WithResource(resource),
	)
	otel.SetMeterProvider(mp)
	meter = mp.Meter("stock-tracker")

	// Create application metric instruments
	watchlistAddCounter, err = meter.Int64Counter(
		"watchlist_additions_total",
		metric.WithDescription("Total number of items added to watchlist."),
		metric.WithUnit("1"),
	)
	if err != nil {
		log.Fatal(err)
	}

	watchlistErrorCounter, err = meter.Int64Counter(
		"watchlist_errors_total",
		metric.WithDescription("Total number of errors in watchlist operations."),
		metric.WithUnit("1"),
	)
	if err != nil {
		log.Fatal(err)
	}

	apiRequestCounter, err = meter.Int64Counter(
		"api_requests_total",
		metric.WithDescription("Total number of API requests."),
		metric.WithUnit("1"),
	)
	if err != nil {
		log.Fatal(err)
	}

	apiLatencyHistogram, err = meter.Float64Histogram(
		"api_request_duration_seconds",
		metric.WithDescription("API request latency distribution."),
		metric.WithUnit("s"),
	)
	if err != nil {
		log.Fatal(err)
	}

	dbOperationCounter, err = meter.Int64Counter(
		"db_operations_total",
		metric.WithDescription("Total number of database operations."),
		metric.WithUnit("1"),
	)
	if err != nil {
		log.Fatal(err)
	}

	dbLatencyHistogram, err = meter.Float64Histogram(
		"db_operation_duration_seconds",
		metric.WithDescription("Database operation latency distribution."),
		metric.WithUnit("s"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Create system metric instruments
	memoryUsageGauge, err = meter.Float64ObservableGauge(
		"process_memory_bytes",
		metric.WithDescription("Current memory usage of the process."),
		metric.WithUnit("bytes"),
	)
	if err != nil {
		log.Fatal(err)
	}

	cpuUsageGauge, err = meter.Float64ObservableGauge(
		"process_cpu_usage",
		metric.WithDescription("Current CPU usage percentage."),
		metric.WithUnit("percent"),
	)
	if err != nil {
		log.Fatal(err)
	}

	appGoroutinesGauge, err = meter.Int64ObservableGauge(
		"app_goroutines_total",
		metric.WithDescription("Number of goroutines that currently exist in the application."),
		metric.WithUnit("1"),
	)
	if err != nil {
		log.Fatal(err)
	}

	gcPauseLatency, err = meter.Float64Histogram(
		"gc_pause_latency_seconds",
		metric.WithDescription("GC pause latency distribution."),
		metric.WithUnit("s"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Register callbacks for system metrics with exemplars
	_, err = meter.RegisterCallback(
		func(ctx context.Context, o metric.Observer) error {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			// Record memory metrics with exemplars
			o.ObserveFloat64(memoryUsageGauge, float64(m.Alloc),
				metric.WithAttributes(
					attribute.String("type", "heap"),
					attribute.Int64("gc_num", int64(m.NumGC)),
				))

			// Record CPU usage with exemplars
			cpuUsage := getCPUUsage()
			o.ObserveFloat64(cpuUsageGauge, cpuUsage,
				metric.WithAttributes(
					attribute.Int("num_cpu", runtime.NumCPU()),
				))

			// Record goroutines count with exemplars
			o.ObserveInt64(appGoroutinesGauge, int64(runtime.NumGoroutine()),
				metric.WithAttributes(
					attribute.String("type", "total"),
				))

			// Record GC pause time with exemplars
			if m.NumGC > 0 {
				gcPauseLatency.Record(ctx, float64(m.PauseNs[(m.NumGC+255)%256])/1e9,
					metric.WithAttributes(
						attribute.Int64("gc_num", int64(m.NumGC)),
						attribute.Int64("gc_cpu_fraction", int64(m.GCCPUFraction*100)),
					))
			}

			return nil
		},
		memoryUsageGauge,
		cpuUsageGauge,
		appGoroutinesGauge,
	)
	if err != nil {
		log.Fatal(err)
	}

	// Start Prometheus HTTP endpoint
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		}))
		server := &http.Server{
			Addr:    ":2222",
			Handler: mux,
		}
		log.Printf("Prometheus metrics endpoint started at :2222/metrics")
		if err := server.ListenAndServe(); err != nil {
			log.Printf("Prometheus HTTP server error: %v", err)
		}
	}()

	log.Println("Telemetry initialized with Prometheus metrics endpoint")
}

// Helper function to get CPU usage
func getCPUUsage() float64 {
	var cpuStats runtime.MemStats
	runtime.ReadMemStats(&cpuStats)
	return float64(cpuStats.Sys) / float64(runtime.NumCPU())
}

// Helper functions for instrumentation
func startSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return tracer.Start(ctx, name,
		trace.WithAttributes(
			attribute.String("service.name", "stock-tracker"),
			attribute.String("span.type", "request"),
		),
	)
}

func startChildSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	parentSpan := trace.SpanFromContext(ctx)
	if parentSpan.IsRecording() {
		return tracer.Start(ctx, name,
			trace.WithAttributes(
				attribute.String("parent.name", parentSpan.SpanContext().SpanID().String()),
			),
		)
	}
	return startSpan(ctx, name)
}

func recordApiRequest(ctx context.Context, endpoint string, duration time.Duration, statusCode int) {
	apiRequestCounter.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("endpoint", endpoint),
			attribute.Int("status_code", statusCode),
		),
	)

	apiLatencyHistogram.Record(ctx, float64(duration.Seconds()),
		metric.WithAttributes(
			attribute.String("endpoint", endpoint),
		),
	)
}

func recordDBOperation(ctx context.Context, operation string, duration time.Duration, err error) {
	dbOperationCounter.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.Bool("success", err == nil),
		),
	)

	dbLatencyHistogram.Record(ctx, float64(duration.Seconds()),
		metric.WithAttributes(
			attribute.String("operation", operation),
		),
	)
}

func recordError(ctx context.Context, operation string, err error) {
	watchlistErrorCounter.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.String("error", err.Error()),
		),
	)
}

func instrumentDBCall(ctx context.Context, operation string) (context.Context, trace.Span) {
	ctx, span := startChildSpan(ctx, "db."+operation)
	span.SetAttributes(
		attribute.String("db.type", "postgres"),
		attribute.String("db.operation", operation),
	)
	return ctx, span
}

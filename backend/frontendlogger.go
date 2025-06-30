package main

import (
	"encoding/json"
	"net/http"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var propagator = propagation.TraceContext{}

func logFrontendEvent(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Type      string
		Event     string                 `json:"event"`
		Timestamp int64                  `json:"timestamp"`
		Metadata  map[string]interface{} `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		Logger.ErrorContext(r.Context(), "Invalid frontend log payload", "error", err)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	spanCtx := trace.SpanContextFromContext(ctx)

	if payload.Type == "Error" {
		Logger.ErrorContext(ctx, "Frontend log error",
			"event", payload.Event,
			"metadata", payload.Metadata,
			"traceId", spanCtx.TraceID().String(),
			"spanId", spanCtx.SpanID().String(),
		)
	} else {
		Logger.InfoContext(ctx, "Frontend log info",
			"event", payload.Event,
			"metadata", payload.Metadata,
			"traceId", spanCtx.TraceID().String(),
			"spanId", spanCtx.SpanID().String(),
		)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode("Log written to file")
}

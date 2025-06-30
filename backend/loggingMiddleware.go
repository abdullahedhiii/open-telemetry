package main

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

type OtelHandler struct {
	slog.Handler
}

func NewOtelHandler(base slog.Handler) *OtelHandler {
	return &OtelHandler{Handler: base}
}

func (h *OtelHandler) Handle(ctx context.Context, r slog.Record) error {

	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, r)
}

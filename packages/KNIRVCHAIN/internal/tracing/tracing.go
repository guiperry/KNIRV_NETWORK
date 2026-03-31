package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("knirvchain")

func InitTracer(serviceName, jaegerEndpoint string) error {
	return nil
}

func StartSpan(ctx context.Context, operationName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return tracer.Start(ctx, operationName, trace.WithAttributes(attrs...))
}

func AddSpanEvent(span trace.Span, eventName string, attrs ...attribute.KeyValue) {
	span.AddEvent(eventName, trace.WithAttributes(attrs...))
}

func SetSpanAttribute(span trace.Span, key string, value interface{}) {
	span.SetAttributes(attribute.String(key, value.(string)))
}

func GetTracer() trace.Tracer {
	return tracer
}

func WithSpan(ctx context.Context, name string, fn func(context.Context) error) error {
	ctx, span := StartSpan(ctx, name)
	defer span.End()
	return fn(ctx)
}

package tracing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

func TestInitTracer(t *testing.T) {
	err := InitTracer("test-service", "")
	assert.NoError(t, err)
}

func TestStartSpan(t *testing.T) {
	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test-operation")
	assert.NotNil(t, ctx)
	assert.NotNil(t, span)
	span.End()
}

func TestStartSpanWithAttributes(t *testing.T) {
	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test-operation",
		attribute.String("key", "value"),
		attribute.Int("number", 42),
	)
	assert.NotNil(t, ctx)
	assert.NotNil(t, span)
	span.End()
}

func TestAddSpanEvent(t *testing.T) {
	ctx := context.Background()
	_, span := StartSpan(ctx, "test-operation")
	AddSpanEvent(span, "test-event")
	span.End()
}

func TestSetSpanAttribute(t *testing.T) {
	ctx := context.Background()
	_, span := StartSpan(ctx, "test-operation")
	SetSpanAttribute(span, "test-key", "test-value")
	span.End()
}

func TestGetTracer(t *testing.T) {
	tracer := GetTracer()
	assert.NotNil(t, tracer)
}

func TestWithSpan(t *testing.T) {
	ctx := context.Background()
	err := WithSpan(ctx, "test-operation", func(ctx context.Context) error {
		return nil
	})
	assert.NoError(t, err)
}

func TestWithSpan_Error(t *testing.T) {
	ctx := context.Background()
	err := WithSpan(ctx, "test-operation", func(ctx context.Context) error {
		return assert.AnError
	})
	assert.Equal(t, assert.AnError, err)
}

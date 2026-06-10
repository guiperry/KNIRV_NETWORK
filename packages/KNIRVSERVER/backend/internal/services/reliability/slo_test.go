package reliability

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewSLOBudget(t *testing.T) {
	sb := NewSLOBudget()
	assert.NotNil(t, sb)
}

func TestDefineSLO(t *testing.T) {
	sb := NewSLOBudget()
	config := SLOConfig{
		Name:       "uptime",
		Target:     0.99,
		Window:     5 * time.Minute,
		MetricName: "health_check",
	}
	sb.DefineSLO(config)

	slos := sb.ListSLOs()
	assert.Len(t, slos, 1)
	assert.Equal(t, "uptime", slos[0].Name)
	assert.Equal(t, 0.99, slos[0].Target)
}

func TestGetStatusNoMetrics(t *testing.T) {
	sb := NewSLOBudget()
	sb.DefineSLO(SLOConfig{Name: "latency", Target: 0.95, ComparisonOp: "lte"})

	status := sb.GetStatus("latency")
	assert.NotNil(t, status)
	assert.True(t, status.Compliant)
	assert.Equal(t, 0.0, status.Current)
}

func TestGetStatusNonexistent(t *testing.T) {
	sb := NewSLOBudget()
	status := sb.GetStatus("nonexistent")
	assert.Nil(t, status)
}

func TestGetStatusWithMetrics(t *testing.T) {
	sb := NewSLOBudget()
	sb.DefineSLO(SLOConfig{Name: "accuracy", Target: 0.80, ComparisonOp: "gte"})

	sb.RecordMetric("accuracy", 0.85)
	sb.RecordMetric("accuracy", 0.90)
	sb.RecordMetric("accuracy", 0.95)

	status := sb.GetStatus("accuracy")
	assert.NotNil(t, status)
	assert.True(t, status.Compliant)
	assert.InDelta(t, 0.90, status.Current, 0.01)
}

func TestGetStatusBelowTarget(t *testing.T) {
	sb := NewSLOBudget()
	sb.DefineSLO(SLOConfig{Name: "uptime", Target: 0.95, ComparisonOp: "gte"})

	sb.RecordMetric("uptime", 0.50)
	sb.RecordMetric("uptime", 0.60)

	status := sb.GetStatus("uptime")
	assert.NotNil(t, status)
	assert.False(t, status.Compliant)
	assert.InDelta(t, 0.55, status.Current, 0.01)
}

func TestBurnRate(t *testing.T) {
	sb := NewSLOBudget()
	sb.DefineSLO(SLOConfig{Name: "perf", Target: 1.0, ComparisonOp: "gte"})

	sb.RecordMetric("perf", 1.0)
	sb.RecordMetric("perf", 1.0)

	status := sb.GetStatus("perf")
	assert.InDelta(t, 0.0, status.BurnRate, 0.01)
}

func TestListSLOs(t *testing.T) {
	sb := NewSLOBudget()
	sb.DefineSLO(SLOConfig{Name: "slo1", Target: 0.99})
	sb.DefineSLO(SLOConfig{Name: "slo2", Target: 0.95})

	slos := sb.ListSLOs()
	assert.Len(t, slos, 2)
}

func TestRecordMetricMultipleNames(t *testing.T) {
	sb := NewSLOBudget()
	sb.DefineSLO(SLOConfig{Name: "a", Target: 0.9})
	sb.DefineSLO(SLOConfig{Name: "b", Target: 0.8})

	sb.RecordMetric("a", 0.95)
	sb.RecordMetric("b", 0.70)

	assert.InDelta(t, 0.95, sb.GetStatus("a").Current, 0.01)
	assert.InDelta(t, 0.70, sb.GetStatus("b").Current, 0.01)
}

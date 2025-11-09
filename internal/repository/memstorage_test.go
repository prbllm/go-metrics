package repository

import (
	"testing"

	"github.com/prbllm/go-metrics/internal/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestMemStorage_UpdateMetricsBatch(t *testing.T) {
	tests := []struct {
		name        string
		metrics     []*model.Metrics
		expectError bool
	}{
		{
			name: "valid batch with counter and gauge",
			metrics: []*model.Metrics{
				{ID: "counter1", MType: model.Counter, Delta: func() *int64 { v := int64(10); return &v }()},
				{ID: "gauge1", MType: model.Gauge, Value: func() *float64 { v := 3.14; return &v }()},
			},
			expectError: false,
		},
		{
			name:        "nil metrics slice",
			metrics:     nil,
			expectError: true,
		},
		{
			name:        "empty metrics slice",
			metrics:     []*model.Metrics{},
			expectError: false,
		},
		{
			name: "counter accumulation in batch",
			metrics: []*model.Metrics{
				{ID: "counter1", MType: model.Counter, Delta: func() *int64 { v := int64(5); return &v }()},
				{ID: "counter1", MType: model.Counter, Delta: func() *int64 { v := int64(3); return &v }()},
			},
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testStorage := NewMemStorage(zaptest.NewLogger(t).Sugar())
			err := testStorage.UpdateMetricsBatch(test.metrics)
			if test.expectError {
				require.Error(t, err, "Expected error")
			} else {
				require.NoError(t, err, "UpdateMetricsBatch should succeed")
			}
		})
	}
}

func TestMemStorage_UpdateMetricsBatch_CounterAccumulation(t *testing.T) {
	storage := NewMemStorage(zaptest.NewLogger(t).Sugar())

	const metricName = "test_counter"
	delta1 := int64(5)
	delta2 := int64(3)
	expectedDelta := int64(8)

	metrics1 := []*model.Metrics{
		{ID: metricName, MType: model.Counter, Delta: &delta1},
	}
	err := storage.UpdateMetricsBatch(metrics1)
	require.NoError(t, err, "First batch update failed")

	metrics2 := []*model.Metrics{
		{ID: metricName, MType: model.Counter, Delta: &delta2},
	}
	err = storage.UpdateMetricsBatch(metrics2)
	require.NoError(t, err, "Second batch update failed")

	metric, err := storage.GetMetric(&model.Metrics{ID: metricName, MType: model.Counter})
	require.NoError(t, err, "Get failed")
	require.Equal(t, expectedDelta, *metric.Delta, "Delta should accumulate")
}

func TestMemStorage_UpdateMetricsBatch_MultipleMetrics(t *testing.T) {
	storage := NewMemStorage(zaptest.NewLogger(t).Sugar())

	metrics := []*model.Metrics{
		{ID: "counter1", MType: model.Counter, Delta: func() *int64 { v := int64(10); return &v }()},
		{ID: "gauge1", MType: model.Gauge, Value: func() *float64 { v := 3.14; return &v }()},
		{ID: "counter2", MType: model.Counter, Delta: func() *int64 { v := int64(20); return &v }()},
		{ID: "gauge2", MType: model.Gauge, Value: func() *float64 { v := 5.67; return &v }()},
	}

	err := storage.UpdateMetricsBatch(metrics)
	require.NoError(t, err, "UpdateMetricsBatch should succeed")

	allMetrics := storage.GetAllMetrics()
	require.Equal(t, 4, len(allMetrics), "Should have 4 metrics")

	counter1, err := storage.GetMetric(&model.Metrics{ID: "counter1", MType: model.Counter})
	require.NoError(t, err)
	require.Equal(t, int64(10), *counter1.Delta)

	gauge1, err := storage.GetMetric(&model.Metrics{ID: "gauge1", MType: model.Gauge})
	require.NoError(t, err)
	require.Equal(t, 3.14, *gauge1.Value)
}

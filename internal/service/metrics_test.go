package service

import (
	"strconv"
	"testing"

	"github.com/prbllm/go-metrics/internal/model"
	"github.com/prbllm/go-metrics/internal/repository"
	"github.com/stretchr/testify/require"
)

func TestMetricsService_UpdateMetric(t *testing.T) {
	storage := repository.NewMemStorage()
	service := NewMetricsService(storage)

	tests := []struct {
		name        string
		metricType  string
		metricName  string
		metricValue string
	}{
		{
			name:        "counter",
			metricType:  model.Counter,
			metricName:  "test_counter",
			metricValue: "42",
		},
		{
			name:        "gauge",
			metricType:  model.Gauge,
			metricName:  "test_gauge",
			metricValue: "3.14",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := service.UpdateMetric(test.metricType, test.metricName, test.metricValue)
			require.NoError(t, err, "Update failed")

			metric, err := service.GetMetric(test.metricType, test.metricName)
			require.NoError(t, err, "Get failed")
			if test.metricType == model.Gauge {
				expectedValue, err := strconv.ParseFloat(test.metricValue, 64)
				require.NoError(t, err)
				require.Equal(t, expectedValue, *metric.Value, "Value is not equal to expected")
			} else {
				expectedValue, err := strconv.ParseInt(test.metricValue, 10, 64)
				require.NoError(t, err)
				require.Equal(t, expectedValue, *metric.Delta, "Value is not equal to expected")
			}
		})
	}
}

func TestMetricsService_CounterAccumulation(t *testing.T) {
	storage := repository.NewMemStorage()
	service := NewMetricsService(storage)

	const metricName = "test_counter"
	const metricValue = "5"
	const expectedDelta = int64(10)
	err := service.UpdateMetric(model.Counter, metricName, metricValue)
	require.NoError(t, err, "First update failed")

	err = service.UpdateMetric(model.Counter, metricName, metricValue)
	require.NoError(t, err, "Second update failed")

	metric, err := service.GetMetric(model.Counter, metricName)
	require.NoError(t, err, "Get failed")
	require.Equal(t, expectedDelta, *metric.Delta, "Delta is not equal to expected")
}

func TestMetricsService_GaugeReplacement(t *testing.T) {
	storage := repository.NewMemStorage()
	service := NewMetricsService(storage)

	const metricName = "test_gauge"
	const metricValue = "10.5"
	const newMetricValue = "20.7"

	err := service.UpdateMetric(model.Gauge, metricName, metricValue)
	require.NoError(t, err, "First update failed")

	err = service.UpdateMetric(model.Gauge, metricName, newMetricValue)
	require.NoError(t, err, "Second update failed")

	metric, err := service.GetMetric(model.Gauge, metricName)
	require.NoError(t, err, "Get failed")
	expectedValue, err := strconv.ParseFloat(newMetricValue, 64)
	require.NoError(t, err)
	require.Equal(t, expectedValue, *metric.Value, "Value is not equal to expected")
}

func TestMetricsService_GetAllMetrics(t *testing.T) {
	storage := repository.NewMemStorage()
	service := NewMetricsService(storage)

	expectedValue := float64(10.5)

	expectedMetrics := []*model.Metrics{
		{ID: "test_gauge", MType: model.Gauge, Value: &expectedValue},
		{ID: "test_gauge2", MType: model.Gauge, Value: &expectedValue},
	}
	service.UpdateMetric(model.Gauge, expectedMetrics[0].ID, strconv.FormatFloat(expectedValue, 'f', -1, 64))
	service.UpdateMetric(model.Gauge, expectedMetrics[1].ID, strconv.FormatFloat(expectedValue, 'f', -1, 64))

	metrics, err := service.GetAllMetrics()
	require.NoError(t, err, "Get all metrics failed")
	require.Equal(t, expectedMetrics, metrics, "Metrics is not equal to expected")
}

func TestMetricsService_GetMetric(t *testing.T) {
	storage := repository.NewMemStorage()
	service := NewMetricsService(storage)
	expectedValue := float64(10.5)

	expectedMetric := &model.Metrics{MType: model.Gauge, ID: "test_gauge", Value: &expectedValue}
	service.UpdateMetric(model.Gauge, expectedMetric.ID, strconv.FormatFloat(expectedValue, 'f', -1, 64))
	metric, err := service.GetMetric(model.Gauge, expectedMetric.ID)
	require.NoError(t, err, "Get metric failed")
	require.Equal(t, metric, expectedMetric, "Metric is not equal to expected")

	expectedDelta := int64(10)
	expectedMetric = &model.Metrics{MType: model.Counter, ID: "test_counter", Delta: &expectedDelta}
	service.UpdateMetric(model.Counter, expectedMetric.ID, strconv.FormatInt(expectedDelta, 10))
	metric, err = service.GetMetric(model.Counter, expectedMetric.ID)
	require.NoError(t, err, "Get metric failed")
	require.Equal(t, metric, expectedMetric, "Metric is not equal to expected")
}

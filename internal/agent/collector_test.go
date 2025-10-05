package agent

import (
	"testing"

	"github.com/prbllm/go-metrics/internal/model"
	"github.com/stretchr/testify/require"
)

func getMetricByID(metrics []model.Metrics, id string) *model.Metrics {
	for _, metric := range metrics {
		if metric.ID == id {
			return &metric
		}
	}
	return nil
}

func TestCollectorMetrics(t *testing.T) {
	collector := RuntimeMetricsCollector{}
	metrics := collector.Collect()
	require.NotNil(t, metrics, "Metrics is nil")
	require.NotEmpty(t, metrics, "Metrics is empty")

	metricsNames := []string{
		"Alloc",
		"BuckHashSys",
		"Frees",
		"GCCPUFraction",
		"GCSys",
		"HeapAlloc",
		"HeapIdle",
		"HeapInuse",
		"HeapObjects",
		"HeapReleased",
		"HeapSys",
		"LastGC",
		"Lookups",
		"MCacheInuse",
		"MCacheSys",
		"MSpanInuse",
		"MSpanSys",
		"Mallocs",
		"NextGC",
		"NumForcedGC",
		"NumGC",
		"OtherSys",
		"PauseTotalNs",
		"StackInuse",
		"StackSys",
		"Sys",
		"TotalAlloc",
		"PollCount",
		"RandomValue",
	}

	require.Equal(t, len(metricsNames), len(metrics), "Metrics count is not equal to expected")

	for _, metricName := range metricsNames {
		require.NotNil(t, getMetricByID(metrics, metricName), "Metric is nil: ", metricName)
	}
}

func TestCollectorMetricsPollCountType(t *testing.T) {
	collector := RuntimeMetricsCollector{}
	metrics := collector.Collect()

	const metricName = "PollCount"
	const expectedDelta = int64(1)

	pollCount := getMetricByID(metrics, metricName)
	require.Equal(t, model.Counter, pollCount.MType)
	require.Equal(t, expectedDelta, *pollCount.Delta)

	metrics = collector.Collect()
	pollCount = getMetricByID(metrics, metricName)
	require.Equal(t, model.Counter, pollCount.MType)
	require.Equal(t, expectedDelta, *pollCount.Delta)
}

func TestCollectorMetricsRandomValueType(t *testing.T) {
	collector := RuntimeMetricsCollector{}
	metrics := collector.Collect()

	const metricName = "RandomValue"

	metric := getMetricByID(metrics, metricName)
	require.Equal(t, model.Gauge, metric.MType)
	require.NotNil(t, metric.Value)
	randomValue := *metric.Value

	metrics = collector.Collect()
	metric = getMetricByID(metrics, metricName)
	require.NotNil(t, metric.Value)
	require.NotEqual(t, randomValue, *metric.Value)
}

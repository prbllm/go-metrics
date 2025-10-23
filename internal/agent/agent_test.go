package agent

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/prbllm/go-metrics/internal/model"
	"github.com/stretchr/testify/require"
)

func TestAgentGenerateUrl(t *testing.T) {
	commonValue := float64(1.0)
	commonDelta := int64(1)

	testData := []struct {
		metric      model.Metrics
		expectedURL string
		expectError bool
	}{
		{
			metric:      model.Metrics{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
			expectedURL: "http://localhost:8080/update/gauge/test_metric/1.000000",
			expectError: false,
		},
		{
			metric:      model.Metrics{ID: "test_metric", MType: model.Counter, Delta: &commonDelta},
			expectedURL: "http://localhost:8080/update/counter/test_metric/1",
			expectError: false,
		},
		{
			metric:      model.Metrics{ID: "test_metric", MType: model.Gauge},
			expectedURL: "",
			expectError: true,
		},
		{
			metric:      model.Metrics{ID: "test_metric", MType: model.Counter},
			expectedURL: "",
			expectError: true,
		},
	}

	agent := NewAgent(nil, nil, "http://localhost:8080/update/", 0, 0)
	for _, test := range testData {
		url, err := agent.generateURL(test.metric)
		if test.expectError {
			require.Error(t, err, "Expected error")
		} else {
			require.NoError(t, err, "Failed to generate URL")
		}
		require.Equal(t, test.expectedURL, url, "URL is not equal to expected")
	}
}

func TestAgentSendMetrics(t *testing.T) {
	commonValue := float64(1.0)
	commonDelta := int64(1)

	agent := NewAgent(http.DefaultClient, nil, "http://localhost:8080/update/", 0, 0)
	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
		{ID: "test_metric", MType: model.Counter, Delta: &commonDelta},
	}
	err := agent.sendMetrics(metrics)
	require.NoError(t, err, "Failed to send metrics")
}

func TestAgentSendMetricsJSON(t *testing.T) {
	commonValue := float64(1.0)
	commonDelta := int64(1)

	agent := NewAgent(http.DefaultClient, nil, "http://localhost:8080/update/", 0, 0)
	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
		{ID: "test_metric", MType: model.Counter, Delta: &commonDelta},
	}
	err := agent.SendMetricsJSON(metrics)
	require.NoError(t, err, "Failed to send metrics via JSON")
}

func TestAgentSendMetricsJSONWithNilClient(t *testing.T) {
	commonValue := float64(1.0)
	commonDelta := int64(1)

	agent := NewAgent(nil, nil, "http://localhost:8080/update/", 0, 0)
	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
		{ID: "test_metric", MType: model.Counter, Delta: &commonDelta},
	}
	err := agent.SendMetricsJSON(metrics)
	require.Error(t, err, "Expected error for nil client")
	require.Contains(t, err.Error(), "client is nil")
}

func TestAgentSendMetricsJSONSerialization(t *testing.T) {
	commonValue := float64(1.0)
	commonDelta := int64(1)

	metrics := []model.Metrics{
		{ID: "test_gauge", MType: model.Gauge, Value: &commonValue},
		{ID: "test_counter", MType: model.Counter, Delta: &commonDelta},
	}

	for _, metric := range metrics {
		jsonData, err := json.Marshal(metric)
		require.NoError(t, err, "Failed to marshal metric to JSON")
		require.NotEmpty(t, jsonData, "JSON data should not be empty")

		jsonStr := string(jsonData)
		require.Contains(t, jsonStr, metric.ID, "JSON should contain metric ID")
		require.Contains(t, jsonStr, metric.MType, "JSON should contain metric type")

		if metric.MType == model.Gauge && metric.Value != nil {
			require.Contains(t, jsonStr, "value", "JSON should contain value field for gauge")
		}
		if metric.MType == model.Counter && metric.Delta != nil {
			require.Contains(t, jsonStr, "delta", "JSON should contain delta field for counter")
		}
	}
}

package agent

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
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

	logger := zaptest.NewLogger(t).Sugar()
	cfg := &config.Config{
		ServerHost: "localhost:8080",
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(nil, nil, logger)
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

	logger := zaptest.NewLogger(t).Sugar()
	cfg := &config.Config{
		ServerHost: "localhost:8080",
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(http.DefaultClient, nil, logger)
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

	logger := zaptest.NewLogger(t).Sugar()
	cfg := &config.Config{
		ServerHost: "localhost:8080",
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(http.DefaultClient, nil, logger)
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

	logger := zaptest.NewLogger(t).Sugar()
	cfg := &config.Config{
		ServerHost: "localhost:8080",
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(nil, nil, logger)
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

func TestAgentSendMetricsBatchJSON(t *testing.T) {
	commonValue := float64(1.0)
	commonDelta := int64(1)

	logger := zaptest.NewLogger(t).Sugar()
	cfg := &config.Config{
		ServerHost: "localhost:8080",
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(http.DefaultClient, nil, logger)

	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
		{ID: "test_metric", MType: model.Counter, Delta: &commonDelta},
	}
	_ = agent.SendMetricsBatchJSON(metrics)
}

func TestAgentSendMetricsBatchJSONWithNilClient(t *testing.T) {
	commonValue := float64(1.0)
	commonDelta := int64(1)

	logger := zaptest.NewLogger(t).Sugar()
	cfg := &config.Config{
		ServerHost: "localhost:8080",
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(nil, nil, logger)

	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
		{ID: "test_metric", MType: model.Counter, Delta: &commonDelta},
	}

	err := agent.SendMetricsBatchJSON(metrics)
	require.Error(t, err, "Expected error for nil client")
	require.Contains(t, err.Error(), "client is nil")
}

func TestAgentSendMetricsBatchJSONEmptyBatch(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	cfg := &config.Config{
		ServerHost: "localhost:8080",
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(http.DefaultClient, nil, logger)

	metrics := []model.Metrics{}

	err := agent.SendMetricsBatchJSON(metrics)
	require.NoError(t, err, "Empty batch should not return error")
}

func TestAgentSendMetricsBatchJSONSerialization(t *testing.T) {
	commonValue := float64(1.0)
	commonDelta := int64(1)

	metrics := []model.Metrics{
		{ID: "test_gauge", MType: model.Gauge, Value: &commonValue},
		{ID: "test_counter", MType: model.Counter, Delta: &commonDelta},
	}

	jsonData, err := json.Marshal(metrics)
	require.NoError(t, err, "Failed to marshal metrics batch to JSON")
	require.NotEmpty(t, jsonData, "JSON data should not be empty")

	jsonStr := string(jsonData)
	require.Contains(t, jsonStr, "test_gauge", "JSON should contain first metric ID")
	require.Contains(t, jsonStr, "test_counter", "JSON should contain second metric ID")
	require.Contains(t, jsonStr, "gauge", "JSON should contain gauge type")
	require.Contains(t, jsonStr, "counter", "JSON should contain counter type")
}

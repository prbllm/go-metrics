package agent

import (
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

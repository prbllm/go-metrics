package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/handler"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/prbllm/go-metrics/internal/repository"
	"github.com/prbllm/go-metrics/internal/service"
	"github.com/stretchr/testify/require"
)

func TestFullIntegration(t *testing.T) {
	storage := repository.NewMemStorage()
	metricsService := service.NewMetricsService(storage)
	handlers := handler.NewHandlers(metricsService)

	mux := http.NewServeMux()
	mux.HandleFunc(config.NotFoundPath, handlers.NotFoundHandler)
	mux.HandleFunc(config.UpdatePath, handlers.UpdateHandler)

	server := httptest.NewServer(mux)
	defer server.Close()

	t.Run("counter", func(t *testing.T) {
		const metricName = "test_counter"
		const metricValue = "10"
		req, err := http.NewRequest(http.MethodPost, server.URL+"/update/counter/"+metricName+"/"+metricValue, nil)
		require.NoError(t, err, "Failed to create request")

		req.Header.Set("Content-Type", "text/plain")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "Failed to send request")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Expected status 200, got %d", resp.StatusCode)

		metric, err := storage.GetMetric(&model.Metrics{MType: model.Counter, ID: metricName})
		require.NoError(t, err, "Expected metric to be saved")

		expectedValue, err := strconv.ParseInt(metricValue, 10, 64)
		require.NoError(t, err)
		require.Equal(t, expectedValue, *metric.Delta, "Metric value is not equal to expected")
	})

	t.Run("gauge", func(t *testing.T) {
		const metricName = "test_gauge"
		const metricValue = "3.14"
		const metricValue2 = "132.42"
		req, err := http.NewRequest(http.MethodPost, server.URL+"/update/gauge/"+metricName+"/"+metricValue, nil)
		require.NoError(t, err, "Failed to create request")

		req.Header.Set("Content-Type", "text/plain")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "Failed to send request")

		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Expected status 200, got %d", resp.StatusCode)

		metric, err := storage.GetMetric(&model.Metrics{MType: model.Gauge, ID: metricName})
		require.NoError(t, err, "Expected metric to be saved")
		expectedValue, err := strconv.ParseFloat(metricValue, 64)
		require.NoError(t, err)
		require.Equal(t, expectedValue, *metric.Value, "Metric value is not equal to expected")

		req2, err := http.NewRequest(http.MethodPost, server.URL+"/update/gauge/"+metricName+"/"+metricValue2, nil)
		require.NoError(t, err, "Failed to create request")

		req2.Header.Set("Content-Type", "text/plain")
		resp2, err := http.DefaultClient.Do(req2)
		require.NoError(t, err, "Failed to send request")

		resp2.Body.Close()

		metric, err = storage.GetMetric(&model.Metrics{MType: model.Gauge, ID: metricName})
		require.NoError(t, err, "Expected metric to be saved")
		expectedValue, err = strconv.ParseFloat(metricValue2, 64)
		require.NoError(t, err)
		require.Equal(t, expectedValue, *metric.Value, "Metric value is not equal to expected")
	})

	t.Run("counter accumulation", func(t *testing.T) {
		const metricName = "accumulator"
		const metricValue = "5"
		req1, err := http.NewRequest(http.MethodPost, server.URL+"/update/counter/"+metricName+"/"+metricValue, nil)
		require.NoError(t, err, "Failed to create request")

		req1.Header.Set("Content-Type", "text/plain")
		resp1, err := http.DefaultClient.Do(req1)
		require.NoError(t, err, "Failed to send request")

		resp1.Body.Close()

		req2, err := http.NewRequest(http.MethodPost, server.URL+"/update/counter/"+metricName+"/"+metricValue, nil)
		require.NoError(t, err, "Failed to create request")

		req2.Header.Set("Content-Type", "text/plain")
		resp2, err := http.DefaultClient.Do(req2)
		require.NoError(t, err, "Failed to send request")

		resp2.Body.Close()

		metric, err := storage.GetMetric(&model.Metrics{MType: model.Counter, ID: metricName})
		require.NoError(t, err, "Expected metric to be saved")

		expectedValue, err := strconv.ParseInt(metricValue, 10, 64)
		require.NoError(t, err)
		require.Equal(t, 2*expectedValue, *metric.Delta, "Expected delta is not equal to expected")
	})

	t.Run("error cases", func(t *testing.T) {
		testCases := []struct {
			name           string
			path           string
			method         string
			expectedStatus int
		}{
			{
				name:           "invalid method",
				path:           "/update/counter/test/42",
				method:         http.MethodGet,
				expectedStatus: http.StatusMethodNotAllowed,
			},
			{
				name:           "invalid path",
				path:           "/update/counter/test",
				method:         http.MethodPost,
				expectedStatus: http.StatusNotFound,
			},
			{
				name:           "invalid metric type",
				path:           "/update/invalid/test/42",
				method:         http.MethodPost,
				expectedStatus: http.StatusBadRequest,
			},
			{
				name:           "invalid counter value",
				path:           "/update/counter/test/abc",
				method:         http.MethodPost,
				expectedStatus: http.StatusBadRequest,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req, err := http.NewRequest(tc.method, server.URL+tc.path, nil)
				if err != nil {
					t.Fatalf("Failed to create request: %v", err)
				}
				req.Header.Set("Content-Type", "text/plain")

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Fatalf("Failed to send request: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != tc.expectedStatus {
					t.Errorf("Expected status %d, got %d", tc.expectedStatus, resp.StatusCode)
				}
			})
		}
	})
}

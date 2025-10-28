package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/prbllm/go-metrics/internal/compression"
	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/handler"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/prbllm/go-metrics/internal/repository"
	"github.com/prbllm/go-metrics/internal/service"
	"github.com/stretchr/testify/require"

	"github.com/go-chi/chi/v5"
)

func TestHTTPAPIIntegration(t *testing.T) {
	storage := repository.NewMemStorage()
	metricsService := service.NewMetricsService(storage)
	handlers := handler.NewHandlers(metricsService)

	router := chi.NewRouter()
	router.Use(handler.GzipDecompressMiddleware())
	router.Route(config.CommonPath, func(r chi.Router) {
		r.Get("/", handlers.GetAllMetricsHandlerByURL)
		r.Route(config.UpdatePath, func(r chi.Router) {
			r.Post("/{metricType}/{metricName}/{metricValue}", handlers.UpdateMetricHandlerByURL)
			r.Post("/", handlers.UpdateMetricHandlerByJSON)
		})
		r.Route(config.ValuePath, func(r chi.Router) {
			r.Get("/{metricType}/{metricName}", handlers.GetValueHandlerByURL)
			r.Post("/", handlers.GetValueHandlerByJSON)
		})
	})

	server := httptest.NewServer(router)
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

	t.Run("get all metrics", func(t *testing.T) {
		requestUpdate, err := http.NewRequest(http.MethodPost, server.URL+"/update/counter/test_all_metrics_counter/10", nil)
		require.NoError(t, err, "Failed to create request")
		requestUpdate.Header.Set("Content-Type", "text/plain")

		responseUpdate, err := http.DefaultClient.Do(requestUpdate)
		require.NoError(t, err, "Failed to send request")
		responseUpdate.Body.Close()

		req, err := http.NewRequest(http.MethodGet, server.URL+"/", nil)
		require.NoError(t, err, "Failed to create request")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "Failed to send request")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Expected status 200, got %d", resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read response body")
		require.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"), "Expected content type %s, got %s", "text/html; charset=utf-8", resp.Header.Get("Content-Type"))
		require.Contains(t, string(body), "test_all_metrics_counter: 10")
	})

	t.Run("get value", func(t *testing.T) {
		requestUpdate, err := http.NewRequest(http.MethodPost, server.URL+"/update/counter/test_get_counter/10", nil)
		require.NoError(t, err, "Failed to create request")
		requestUpdate.Header.Set("Content-Type", "text/plain")

		responseUpdate, err := http.DefaultClient.Do(requestUpdate)
		require.NoError(t, err, "Failed to send request")
		responseUpdate.Body.Close()

		require.NoError(t, err, "Failed to send request")

		req, err := http.NewRequest(http.MethodGet, server.URL+"/value/counter/test_get_counter", nil)
		require.NoError(t, err, "Failed to create request")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "Failed to send request")
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode, "Expected status 200, got %d", resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read response body")
		require.Equal(t, "10", string(body), "Expected body 10, got %s", string(body))
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

	t.Run("JSON counter", func(t *testing.T) {
		const metricName = "test_json_counter"
		metricValue := int64(15)

		metric := model.Metrics{
			ID:    metricName,
			MType: model.Counter,
			Delta: &metricValue,
		}

		jsonData, err := json.Marshal(metric)
		require.NoError(t, err, "Failed to marshal metric to JSON")

		req, err := http.NewRequest(http.MethodPost, server.URL+config.UpdatePath, bytes.NewBuffer(jsonData))
		require.NoError(t, err, "Failed to create request")
		req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "Failed to send request")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Expected status 200, got %d", resp.StatusCode)

		savedMetric, err := storage.GetMetric(&model.Metrics{MType: model.Counter, ID: metricName})
		require.NoError(t, err, "Expected metric to be saved")
		require.Equal(t, metricValue, *savedMetric.Delta, "Metric value is not equal to expected")
	})

	t.Run("JSON gauge", func(t *testing.T) {
		const metricName = "test_json_gauge"
		metricValue := 3.14159

		metric := model.Metrics{
			ID:    metricName,
			MType: model.Gauge,
			Value: &metricValue,
		}

		jsonData, err := json.Marshal(metric)
		require.NoError(t, err, "Failed to marshal metric to JSON")

		req, err := http.NewRequest(http.MethodPost, server.URL+config.UpdatePath, bytes.NewBuffer(jsonData))
		require.NoError(t, err, "Failed to create request")
		req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "Failed to send request")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Expected status 200, got %d", resp.StatusCode)

		savedMetric, err := storage.GetMetric(&model.Metrics{MType: model.Gauge, ID: metricName})
		require.NoError(t, err, "Expected metric to be saved")
		require.Equal(t, metricValue, *savedMetric.Value, "Metric value is not equal to expected")
	})

	t.Run("JSON get value", func(t *testing.T) {
		const metricName = "test_json_get_value"
		metricValue := int64(42)

		metric := model.Metrics{
			ID:    metricName,
			MType: model.Counter,
			Delta: &metricValue,
		}

		jsonData, err := json.Marshal(metric)
		require.NoError(t, err, "Failed to marshal metric to JSON")

		req, err := http.NewRequest(http.MethodPost, server.URL+config.UpdatePath, bytes.NewBuffer(jsonData))
		require.NoError(t, err, "Failed to create request")
		req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "Failed to send request")
		resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Expected status 200, got %d", resp.StatusCode)

		queryMetric := model.Metrics{
			ID:    metricName,
			MType: model.Counter,
		}

		queryJSONData, err := json.Marshal(queryMetric)
		require.NoError(t, err, "Failed to marshal query metric to JSON")

		req2, err := http.NewRequest(http.MethodPost, server.URL+config.ValuePath, bytes.NewBuffer(queryJSONData))
		require.NoError(t, err, "Failed to create request")
		req2.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)

		resp2, err := http.DefaultClient.Do(req2)
		require.NoError(t, err, "Failed to send request")
		defer resp2.Body.Close()

		require.Equal(t, http.StatusOK, resp2.StatusCode, "Expected status 200, got %d", resp2.StatusCode)
		require.Equal(t, config.ContentTypeJSON, resp2.Header.Get(config.ContentTypeHeader), "Expected JSON content type")

		var responseMetric model.Metrics
		err = json.NewDecoder(resp2.Body).Decode(&responseMetric)
		require.NoError(t, err, "Failed to decode response JSON")
		require.Equal(t, metricName, responseMetric.ID, "Metric ID should match")
		require.Equal(t, model.Counter, responseMetric.MType, "Metric type should match")
		require.Equal(t, metricValue, *responseMetric.Delta, "Metric value should match")
	})

	t.Run("JSON error cases", func(t *testing.T) {
		testCases := []struct {
			name           string
			jsonData       string
			contentType    string
			expectedStatus int
		}{
			{
				name:           "invalid JSON",
				jsonData:       `{"id": "test", "type": "gauge", "value": 1.0`,
				contentType:    config.ContentTypeJSON,
				expectedStatus: http.StatusBadRequest,
			},
			{
				name:           "invalid content type",
				jsonData:       `{"id": "test", "type": "gauge", "value": 1.0}`,
				contentType:    config.ContentTypeTextPlain,
				expectedStatus: http.StatusBadRequest,
			},
			{
				name:           "invalid metric type",
				jsonData:       `{"id": "test", "type": "invalid", "value": 1.0}`,
				contentType:    config.ContentTypeJSON,
				expectedStatus: http.StatusBadRequest,
			},
			{
				name:           "missing required fields",
				jsonData:       `{"id": "test"}`,
				contentType:    config.ContentTypeJSON,
				expectedStatus: http.StatusBadRequest,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req, err := http.NewRequest(http.MethodPost, server.URL+config.UpdatePath, bytes.NewBufferString(tc.jsonData))
				require.NoError(t, err, "Failed to create request")
				req.Header.Set(config.ContentTypeHeader, tc.contentType)

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err, "Failed to send request")
				defer resp.Body.Close()

				require.Equal(t, tc.expectedStatus, resp.StatusCode, "Expected status %d, got %d", tc.expectedStatus, resp.StatusCode)
			})
		}
	})

	t.Run("gzip compression", func(t *testing.T) {
		testMetrics := []model.Metrics{
			{ID: "test_gzip_counter_1", MType: model.Counter, Delta: func() *int64 { v := int64(100); return &v }()},
			{ID: "test_gzip_counter_2", MType: model.Counter, Delta: func() *int64 { v := int64(200); return &v }()},
			{ID: "test_gzip_gauge_1", MType: model.Gauge, Value: func() *float64 { v := 3.14159; return &v }()},
			{ID: "test_gzip_gauge_2", MType: model.Gauge, Value: func() *float64 { v := 2.71828; return &v }()},
		}

		for _, metric := range testMetrics {
			err := storage.UpdateMetric(&metric)
			require.NoError(t, err, "Failed to add test metric")
		}

		t.Run("response compression with gzip", func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, server.URL+"/", nil)
			require.NoError(t, err, "Failed to create request")
			req.Header.Set(config.AcceptEncodingHeader, config.ContentEncodingGzip)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err, "Failed to send request")
			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode, "Expected status 200, got %d", resp.StatusCode)

			require.Equal(t, config.ContentEncodingGzip, resp.Header.Get(config.ContentEncodingHeader), "Expected Content-Encoding: gzip")
			require.Equal(t, config.AcceptEncodingHeader, resp.Header.Get(config.VaryHeader), "Expected Vary: Accept-Encoding")

			decompressedBody, err := compression.DecompressReader(resp.Body)
			require.NoError(t, err, "Failed to decompress response")

			bodyStr := string(decompressedBody)
			require.Contains(t, bodyStr, "test_gzip_counter_1: 100", "Expected to find counter 1")
			require.Contains(t, bodyStr, "test_gzip_counter_2: 200", "Expected to find counter 2")
			require.Contains(t, bodyStr, "test_gzip_gauge_1: 3.14159", "Expected to find gauge 1")
			require.Contains(t, bodyStr, "test_gzip_gauge_2: 2.71828", "Expected to find gauge 2")
		})

		t.Run("no compression without gzip", func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, server.URL+"/", nil)
			require.NoError(t, err, "Failed to create request")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err, "Failed to send request")
			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode, "Expected status 200, got %d", resp.StatusCode)

			require.Empty(t, resp.Header.Get(config.ContentEncodingHeader), "Expected no Content-Encoding header")

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err, "Failed to read response body")

			bodyStr := string(body)
			require.Contains(t, bodyStr, "test_gzip_counter_1: 100", "Expected to find counter 1")
			require.Contains(t, bodyStr, "test_gzip_counter_2: 200", "Expected to find counter 2")
			require.Contains(t, bodyStr, "test_gzip_gauge_1: 3.14159", "Expected to find gauge 1")
			require.Contains(t, bodyStr, "test_gzip_gauge_2: 2.71828", "Expected to find gauge 2")
		})

		t.Run("JSON response compression", func(t *testing.T) {
			metric := model.Metrics{
				ID:    "test_gzip_json",
				MType: model.Counter,
				Delta: func() *int64 { v := int64(42); return &v }(),
			}

			jsonData, err := json.Marshal(metric)
			require.NoError(t, err, "Failed to marshal metric to JSON")

			req, err := http.NewRequest(http.MethodPost, server.URL+config.UpdatePath, bytes.NewBuffer(jsonData))
			require.NoError(t, err, "Failed to create request")
			req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err, "Failed to send request")
			resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode, "Expected status 200, got %d", resp.StatusCode)

			queryMetric := model.Metrics{
				ID:    "test_gzip_json",
				MType: model.Counter,
			}

			queryJSONData, err := json.Marshal(queryMetric)
			require.NoError(t, err, "Failed to marshal query metric to JSON")

			req2, err := http.NewRequest(http.MethodPost, server.URL+config.ValuePath, bytes.NewBuffer(queryJSONData))
			require.NoError(t, err, "Failed to create request")
			req2.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)
			req2.Header.Set(config.AcceptEncodingHeader, config.ContentEncodingGzip)

			resp2, err := http.DefaultClient.Do(req2)
			require.NoError(t, err, "Failed to send request")
			defer resp2.Body.Close()

			require.Equal(t, http.StatusOK, resp2.StatusCode, "Expected status 200, got %d", resp2.StatusCode)

			require.Equal(t, config.ContentEncodingGzip, resp2.Header.Get(config.ContentEncodingHeader), "Expected Content-Encoding: gzip")
			require.Equal(t, config.AcceptEncodingHeader, resp2.Header.Get(config.VaryHeader), "Expected Vary: Accept-Encoding")

			decompressedBody, err := compression.DecompressReader(resp2.Body)
			require.NoError(t, err, "Failed to decompress response")

			var responseMetric model.Metrics
			err = json.NewDecoder(bytes.NewReader(decompressedBody)).Decode(&responseMetric)
			require.NoError(t, err, "Failed to decode response JSON")
			require.Equal(t, "test_gzip_json", responseMetric.ID, "Expected to find metric name in response")
			require.Equal(t, int64(42), *responseMetric.Delta, "Expected to find metric value in response")
		})
	})
}

type FileStorageTestHelper struct {
	t         *testing.T
	tempFile  *os.File
	storage   *repository.MemStorage
	decorator *repository.FileStorageDecorator
}

func NewFileStorageTestHelper(t *testing.T, pattern string) *FileStorageTestHelper {
	tempFile, err := os.CreateTemp("", pattern)
	require.NoError(t, err, "Failed to create temp file")

	storage := repository.NewMemStorage()
	decorator := repository.NewFileStorageDecorator(storage, tempFile.Name())

	return &FileStorageTestHelper{
		t:         t,
		tempFile:  tempFile,
		storage:   storage,
		decorator: decorator,
	}
}

func (h *FileStorageTestHelper) Close() {
	h.tempFile.Close()
	os.Remove(h.tempFile.Name())
}

func (h *FileStorageTestHelper) GetFilePath() string {
	return h.tempFile.Name()
}

func (h *FileStorageTestHelper) GetDecorator() *repository.FileStorageDecorator {
	return h.decorator
}

func (h *FileStorageTestHelper) GetStorage() *repository.MemStorage {
	return h.storage
}

func (h *FileStorageTestHelper) AddMetric(metric *model.Metrics) {
	err := h.decorator.UpdateMetric(metric)
	require.NoError(h.t, err, "Failed to update metric")
}

func (h *FileStorageTestHelper) SaveToFile() {
	err := h.decorator.SaveToFile()
	require.NoError(h.t, err, "Failed to save to file")
}

func (h *FileStorageTestHelper) LoadFromFile() {
	err := h.decorator.LoadFromFile()
	require.NoError(h.t, err, "Failed to load from file")
}

func (h *FileStorageTestHelper) AssertFileExists() []byte {
	fileContent, err := os.ReadFile(h.tempFile.Name())
	require.NoError(h.t, err, "Failed to read file")
	require.NotEmpty(h.t, fileContent, "File should not be empty")
	return fileContent
}

func (h *FileStorageTestHelper) AssertJSONFormat(fileContent []byte) []*model.Metrics {
	var savedMetrics []*model.Metrics
	err := json.Unmarshal(fileContent, &savedMetrics)
	require.NoError(h.t, err, "Failed to unmarshal JSON")
	return savedMetrics
}

func (h *FileStorageTestHelper) AssertMetricExists(metricType, metricID string, expectedValue interface{}) {
	metric, err := h.storage.GetMetric(&model.Metrics{MType: metricType, ID: metricID})
	require.NoError(h.t, err, "Failed to get metric")

	switch metricType {
	case model.Counter:
		expectedDelta := expectedValue.(int64)
		require.Equal(h.t, expectedDelta, *metric.Delta, "Counter delta should match")
	case model.Gauge:
		expectedVal := expectedValue.(float64)
		require.Equal(h.t, expectedVal, *metric.Value, "Gauge value should match")
	}
}

func (h *FileStorageTestHelper) CreateTestFileWithData(testMetrics []*model.Metrics) {
	jsonData, err := json.Marshal(testMetrics)
	require.NoError(h.t, err, "Failed to marshal test metrics")
	err = os.WriteFile(h.tempFile.Name(), jsonData, 0644)
	require.NoError(h.t, err, "Failed to write test file")
}

func TestFileStorageIntegration(t *testing.T) {

	t.Run("integration_multiple_metrics_save_load", func(t *testing.T) {
		helper := NewFileStorageTestHelper(t, "test_multiple_*.json")
		defer helper.Close()

		metrics := []*model.Metrics{
			{ID: "counter1", MType: model.Counter, Delta: func() *int64 { v := int64(10); return &v }()},
			{ID: "counter2", MType: model.Counter, Delta: func() *int64 { v := int64(20); return &v }()},
			{ID: "gauge1", MType: model.Gauge, Value: func() *float64 { v := 1.5; return &v }()},
			{ID: "gauge2", MType: model.Gauge, Value: func() *float64 { v := 2.5; return &v }()},
		}

		for _, metric := range metrics {
			helper.AddMetric(metric)
		}

		helper.SaveToFile()

		newHelper := NewFileStorageTestHelper(t, "test_multiple_load_*.json")
		defer newHelper.Close()

		fileContent := helper.AssertFileExists()
		err := os.WriteFile(newHelper.GetFilePath(), fileContent, 0644)
		require.NoError(t, err, "Failed to copy file")

		newHelper.LoadFromFile()

		allMetrics := newHelper.GetStorage().GetAllMetrics()
		require.Len(t, allMetrics, 4, "Should have 4 metrics")

		newHelper.AssertMetricExists(model.Counter, "counter1", int64(10))
		newHelper.AssertMetricExists(model.Gauge, "gauge2", 2.5)
	})

	t.Run("integration_error_handling", func(t *testing.T) {
		storage := repository.NewMemStorage()
		fileDecorator := repository.NewFileStorageDecorator(storage, "/invalid/path/that/does/not/exist/metrics.json")

		metric := &model.Metrics{
			ID:    "test_error",
			MType: model.Counter,
			Delta: func() *int64 { v := int64(1); return &v }(),
		}

		err := fileDecorator.UpdateMetric(metric)
		require.NoError(t, err, "UpdateMetric should succeed even if file save fails")

		err = fileDecorator.SaveToFile()
		require.Error(t, err, "SaveToFile should fail with invalid path")
	})

	t.Run("integration_config_sync_behavior", func(t *testing.T) {
		originalConfig := config.GetConfig()

		testConfig := &config.Config{
			ServerHost:          "localhost:8080",
			AgentPollInterval:   2 * time.Second,
			AgentReportInterval: 10 * time.Second,
			StoreInterval:       0,
			FileStoragePath:     "test_sync.json",
			Restore:             false,
		}

		config.SetConfig(testConfig)

		defer config.SetConfig(originalConfig)

		helper := NewFileStorageTestHelper(t, "test_config_sync_*.json")
		defer helper.Close()

		metric := &model.Metrics{
			ID:    "test_config_sync_counter",
			MType: model.Counter,
			Delta: func() *int64 { v := int64(300); return &v }(),
		}

		helper.AddMetric(metric)

		fileContent := helper.AssertFileExists()
		savedMetrics := helper.AssertJSONFormat(fileContent)
		require.Len(t, savedMetrics, 1, "Should have one metric for sync save")
		require.Equal(t, "test_config_sync_counter", savedMetrics[0].ID)
		require.Equal(t, int64(300), *savedMetrics[0].Delta)

		allMetrics := helper.GetStorage().GetAllMetrics()
		require.Len(t, allMetrics, 1, "Should have one metric in memory")
		require.Equal(t, "test_config_sync_counter", allMetrics[0].ID)
	})

	t.Run("integration_config_async_behavior", func(t *testing.T) {
		originalConfig := config.GetConfig()

		testConfig := &config.Config{
			ServerHost:          "localhost:8080",
			AgentPollInterval:   2 * time.Second,
			AgentReportInterval: 10 * time.Second,
			StoreInterval:       1 * time.Second,
			FileStoragePath:     "test_async.json",
			Restore:             false,
		}

		config.SetConfig(testConfig)

		defer config.SetConfig(originalConfig)

		helper := NewFileStorageTestHelper(t, "test_config_async_*.json")
		defer helper.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		helper.GetDecorator().StartPeriodicSave(ctx)

		metric := &model.Metrics{
			ID:    "test_config_async_counter",
			MType: model.Counter,
			Delta: func() *int64 { v := int64(400); return &v }(),
		}

		helper.AddMetric(metric)

		allMetrics := helper.GetStorage().GetAllMetrics()
		require.Len(t, allMetrics, 1, "Should have one metric in memory")

		time.Sleep(1500 * time.Millisecond)

		fileContent, err := os.ReadFile(helper.GetFilePath())
		require.NoError(t, err, "Failed to read file")
		require.NotEmpty(t, fileContent, "File should not be empty for async behavior")

		savedMetrics := helper.AssertJSONFormat(fileContent)
		require.Len(t, savedMetrics, 1, "Should have one metric in file")
		require.Equal(t, "test_config_async_counter", savedMetrics[0].ID)
		require.Equal(t, int64(400), *savedMetrics[0].Delta)
	})
}

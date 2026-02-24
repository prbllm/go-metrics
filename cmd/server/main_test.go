package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/prbllm/go-metrics/internal/audit"
	"github.com/prbllm/go-metrics/internal/compression"
	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/encryption"
	"github.com/prbllm/go-metrics/internal/handler"
	"github.com/prbllm/go-metrics/internal/hash"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/prbllm/go-metrics/internal/repository"
	"github.com/prbllm/go-metrics/internal/service"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/go-chi/chi/v5"
)

func TestHTTPAPIIntegration(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	storage := repository.NewMemStorage(logger)
	metricsService := service.NewMetricsService(storage)
	handlers := handler.NewHandlers(metricsService, logger)

	router := chi.NewRouter()
	router.Use(handler.GzipDecompressMiddleware(logger))
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

		metric, err := storage.GetMetric(context.Background(), &model.Metrics{MType: model.Counter, ID: metricName})
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

		metric, err := storage.GetMetric(context.Background(), &model.Metrics{MType: model.Gauge, ID: metricName})
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

		metric, err = storage.GetMetric(context.Background(), &model.Metrics{MType: model.Gauge, ID: metricName})
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

		metric, err := storage.GetMetric(context.Background(), &model.Metrics{MType: model.Counter, ID: metricName})
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

		savedMetric, err := storage.GetMetric(context.Background(), &model.Metrics{MType: model.Counter, ID: metricName})
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

		savedMetric, err := storage.GetMetric(context.Background(), &model.Metrics{MType: model.Gauge, ID: metricName})
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
			err := storage.UpdateMetric(context.Background(), &metric)
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

	logger := zaptest.NewLogger(t).Sugar()
	storage := repository.NewMemStorage(logger)
	decorator := repository.NewFileStorageDecorator(storage, tempFile.Name(), logger)

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
	err := h.decorator.UpdateMetric(context.Background(), metric)
	require.NoError(h.t, err, "Failed to update metric")
}

func (h *FileStorageTestHelper) SaveToFile() {
	err := h.decorator.SaveToFile(context.Background())
	require.NoError(h.t, err, "Failed to save to file")
}

func (h *FileStorageTestHelper) LoadFromFile() {
	err := h.decorator.LoadFromFile(context.Background())
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
	metric, err := h.storage.GetMetric(context.Background(), &model.Metrics{MType: metricType, ID: metricID})
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

		allMetrics := newHelper.GetStorage().GetAllMetrics(context.Background())
		require.Len(t, allMetrics, 4, "Should have 4 metrics")

		newHelper.AssertMetricExists(model.Counter, "counter1", int64(10))
		newHelper.AssertMetricExists(model.Gauge, "gauge2", 2.5)
	})

	t.Run("integration_error_handling", func(t *testing.T) {
		logger := zaptest.NewLogger(t).Sugar()
		storage := repository.NewMemStorage(logger)
		fileDecorator := repository.NewFileStorageDecorator(storage, "/invalid/path/that/does/not/exist/metrics.json", logger)

		metric := &model.Metrics{
			ID:    "test_error",
			MType: model.Counter,
			Delta: func() *int64 { v := int64(1); return &v }(),
		}

		err := fileDecorator.UpdateMetric(context.Background(), metric)
		require.NoError(t, err, "UpdateMetric should succeed even if file save fails")

		err = fileDecorator.SaveToFile(context.Background())
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
		logger := zaptest.NewLogger(t).Sugar()

		config.SetConfig(testConfig, logger)

		defer config.SetConfig(originalConfig, logger)

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

		allMetrics := helper.GetStorage().GetAllMetrics(context.Background())
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
		logger := zaptest.NewLogger(t).Sugar()
		config.SetConfig(testConfig, logger)

		defer config.SetConfig(originalConfig, logger)

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

		allMetrics := helper.GetStorage().GetAllMetrics(context.Background())
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

func TestBatchUpdatesIntegration(t *testing.T) {
	t.Run("batch update with memory storage", func(t *testing.T) {
		logger := zaptest.NewLogger(t).Sugar()
		storage := repository.NewMemStorage(logger)
		metricsService := service.NewMetricsService(storage)
		handlers := handler.NewHandlers(metricsService, logger)

		router := chi.NewRouter()
		router.Use(handler.GzipDecompressMiddleware(logger))
		router.Route(config.CommonPath, func(r chi.Router) {
			r.Route(config.UpdatesPath, func(r chi.Router) {
				r.Post("/", handlers.UpdateMetricsBatchHandler)
			})
		})

		server := httptest.NewServer(router)
		defer server.Close()

		batchMetrics := []model.Metrics{
			{ID: "batch_counter_1", MType: model.Counter, Delta: func() *int64 { v := int64(100); return &v }()},
			{ID: "batch_counter_2", MType: model.Counter, Delta: func() *int64 { v := int64(200); return &v }()},
			{ID: "batch_gauge_1", MType: model.Gauge, Value: func() *float64 { v := 3.14159; return &v }()},
			{ID: "batch_gauge_2", MType: model.Gauge, Value: func() *float64 { v := 2.71828; return &v }()},
		}

		jsonData, err := json.Marshal(batchMetrics)
		require.NoError(t, err, "Failed to marshal batch metrics")

		req, err := http.NewRequest(http.MethodPost, server.URL+config.UpdatesPath, bytes.NewBuffer(jsonData))
		require.NoError(t, err, "Failed to create request")
		req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "Failed to send request")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Expected status 200, got %d", resp.StatusCode)

		allMetrics := storage.GetAllMetrics(context.Background())
		require.Len(t, allMetrics, 4, "Should have 4 metrics in storage")

		counter1, err := storage.GetMetric(context.Background(), &model.Metrics{ID: "batch_counter_1", MType: model.Counter})
		require.NoError(t, err, "Failed to get counter1")
		require.Equal(t, int64(100), *counter1.Delta, "Counter1 delta should match")

		counter2, err := storage.GetMetric(context.Background(), &model.Metrics{ID: "batch_counter_2", MType: model.Counter})
		require.NoError(t, err, "Failed to get counter2")
		require.Equal(t, int64(200), *counter2.Delta, "Counter2 delta should match")

		gauge1, err := storage.GetMetric(context.Background(), &model.Metrics{ID: "batch_gauge_1", MType: model.Gauge})
		require.NoError(t, err, "Failed to get gauge1")
		require.Equal(t, 3.14159, *gauge1.Value, "Gauge1 value should match")

		gauge2, err := storage.GetMetric(context.Background(), &model.Metrics{ID: "batch_gauge_2", MType: model.Gauge})
		require.NoError(t, err, "Failed to get gauge2")
		require.Equal(t, 2.71828, *gauge2.Value, "Gauge2 value should match")
	})

	t.Run("batch update with file storage", func(t *testing.T) {
		helper := NewFileStorageTestHelper(t, "test_batch_file_*.json")
		defer helper.Close()

		logger := zaptest.NewLogger(t).Sugar()
		metricsService := service.NewMetricsService(helper.GetDecorator())
		handlers := handler.NewHandlers(metricsService, logger)

		router := chi.NewRouter()
		router.Use(handler.GzipDecompressMiddleware(logger))
		router.Route(config.CommonPath, func(r chi.Router) {
			r.Route(config.UpdatesPath, func(r chi.Router) {
				r.Post("/", handlers.UpdateMetricsBatchHandler)
			})
		})

		server := httptest.NewServer(router)
		defer server.Close()

		batchMetrics := []model.Metrics{
			{ID: "file_batch_counter", MType: model.Counter, Delta: func() *int64 { v := int64(500); return &v }()},
			{ID: "file_batch_gauge", MType: model.Gauge, Value: func() *float64 { v := 9.87654; return &v }()},
		}

		jsonData, err := json.Marshal(batchMetrics)
		require.NoError(t, err, "Failed to marshal batch metrics")

		req, err := http.NewRequest(http.MethodPost, server.URL+config.UpdatesPath, bytes.NewBuffer(jsonData))
		require.NoError(t, err, "Failed to create request")
		req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "Failed to send request")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Expected status 200, got %d", resp.StatusCode)

		allMetrics := helper.GetStorage().GetAllMetrics(context.Background())
		require.Len(t, allMetrics, 2, "Should have 2 metrics in storage")

		helper.AssertMetricExists(model.Counter, "file_batch_counter", int64(500))
		helper.AssertMetricExists(model.Gauge, "file_batch_gauge", 9.87654)

		helper.SaveToFile()
		fileContent := helper.AssertFileExists()
		savedMetrics := helper.AssertJSONFormat(fileContent)
		require.Len(t, savedMetrics, 2, "Should have 2 metrics in file")

		foundCounter := false
		foundGauge := false
		for _, m := range savedMetrics {
			if m.ID == "file_batch_counter" && m.MType == model.Counter {
				require.Equal(t, int64(500), *m.Delta, "Counter delta in file should match")
				foundCounter = true
			}
			if m.ID == "file_batch_gauge" && m.MType == model.Gauge {
				require.Equal(t, 9.87654, *m.Value, "Gauge value in file should match")
				foundGauge = true
			}
		}
		require.True(t, foundCounter, "Counter should be found in file")
		require.True(t, foundGauge, "Gauge should be found in file")
	})

	t.Run("batch update with counter accumulation", func(t *testing.T) {
		logger := zaptest.NewLogger(t).Sugar()
		storage := repository.NewMemStorage(logger)
		metricsService := service.NewMetricsService(storage)
		handlers := handler.NewHandlers(metricsService, logger)

		router := chi.NewRouter()
		router.Use(handler.GzipDecompressMiddleware(logger))
		router.Route(config.CommonPath, func(r chi.Router) {
			r.Route(config.UpdatesPath, func(r chi.Router) {
				r.Post("/", handlers.UpdateMetricsBatchHandler)
			})
		})

		server := httptest.NewServer(router)
		defer server.Close()

		batch1 := []model.Metrics{
			{ID: "accum_counter", MType: model.Counter, Delta: func() *int64 { v := int64(50); return &v }()},
		}

		jsonData1, err := json.Marshal(batch1)
		require.NoError(t, err, "Failed to marshal first batch")

		req1, err := http.NewRequest(http.MethodPost, server.URL+config.UpdatesPath, bytes.NewBuffer(jsonData1))
		require.NoError(t, err, "Failed to create first request")
		req1.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)

		resp1, err := http.DefaultClient.Do(req1)
		require.NoError(t, err, "Failed to send first request")
		resp1.Body.Close()

		require.Equal(t, http.StatusOK, resp1.StatusCode, "Expected status 200 for first batch")

		batch2 := []model.Metrics{
			{ID: "accum_counter", MType: model.Counter, Delta: func() *int64 { v := int64(75); return &v }()},
		}

		jsonData2, err := json.Marshal(batch2)
		require.NoError(t, err, "Failed to marshal second batch")

		req2, err := http.NewRequest(http.MethodPost, server.URL+config.UpdatesPath, bytes.NewBuffer(jsonData2))
		require.NoError(t, err, "Failed to create second request")
		req2.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)

		resp2, err := http.DefaultClient.Do(req2)
		require.NoError(t, err, "Failed to send second request")
		resp2.Body.Close()

		require.Equal(t, http.StatusOK, resp2.StatusCode, "Expected status 200 for second batch")

		counter, err := storage.GetMetric(context.Background(), &model.Metrics{ID: "accum_counter", MType: model.Counter})
		require.NoError(t, err, "Failed to get accumulated counter")
		require.Equal(t, int64(125), *counter.Delta, "Counter should accumulate: 50 + 75 = 125")
	})

	t.Run("batch update with gzip compression", func(t *testing.T) {
		logger := zaptest.NewLogger(t).Sugar()
		storage := repository.NewMemStorage(logger)
		metricsService := service.NewMetricsService(storage)
		handlers := handler.NewHandlers(metricsService, logger)

		router := chi.NewRouter()
		router.Use(handler.GzipDecompressMiddleware(logger))
		router.Route(config.CommonPath, func(r chi.Router) {
			r.Route(config.UpdatesPath, func(r chi.Router) {
				r.Post("/", handlers.UpdateMetricsBatchHandler)
			})
		})

		server := httptest.NewServer(router)
		defer server.Close()

		batchMetrics := []model.Metrics{
			{ID: "gzip_batch_counter", MType: model.Counter, Delta: func() *int64 { v := int64(999); return &v }()},
			{ID: "gzip_batch_gauge", MType: model.Gauge, Value: func() *float64 { v := 1.23456; return &v }()},
		}

		jsonData, err := json.Marshal(batchMetrics)
		require.NoError(t, err, "Failed to marshal batch metrics")

		compressedData, err := compression.CompressData(jsonData)
		require.NoError(t, err, "Failed to compress data")

		req, err := http.NewRequest(http.MethodPost, server.URL+config.UpdatesPath, bytes.NewBuffer(compressedData))
		require.NoError(t, err, "Failed to create request")
		req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)
		req.Header.Set(config.ContentEncodingHeader, config.ContentEncodingGzip)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "Failed to send request")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Expected status 200, got %d", resp.StatusCode)

		allMetrics := storage.GetAllMetrics(context.Background())
		require.Len(t, allMetrics, 2, "Should have 2 metrics in storage")

		counter, err := storage.GetMetric(context.Background(), &model.Metrics{ID: "gzip_batch_counter", MType: model.Counter})
		require.NoError(t, err, "Failed to get counter")
		require.Equal(t, int64(999), *counter.Delta, "Counter delta should match")

		gauge, err := storage.GetMetric(context.Background(), &model.Metrics{ID: "gzip_batch_gauge", MType: model.Gauge})
		require.NoError(t, err, "Failed to get gauge")
		require.Equal(t, 1.23456, *gauge.Value, "Gauge value should match")
	})

	t.Run("batch update with PostgreSQL", func(t *testing.T) {
		dsn := os.Getenv("DATABASE_DSN")
		if dsn == "" {
			dsn = "postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable"
		}

		logger := zaptest.NewLogger(t).Sugar()
		postgresRepo, err := repository.NewPostgresRepository(context.Background(), dsn, logger)
		if err != nil {
			t.Skipf("Skipping PostgreSQL test: database not available: %v", err)
			return
		}
		defer postgresRepo.Close()

		ctx := context.Background()
		if err := truncateTableForTest(ctx, dsn); err != nil {
			t.Logf("Warning: failed to truncate table: %v", err)
		}

		metricsService := service.NewMetricsService(postgresRepo)
		handlers := handler.NewHandlers(metricsService, logger)

		router := chi.NewRouter()
		router.Use(handler.GzipDecompressMiddleware(logger))
		router.Route(config.CommonPath, func(r chi.Router) {
			r.Route(config.UpdatesPath, func(r chi.Router) {
				r.Post("/", handlers.UpdateMetricsBatchHandler)
			})
		})

		server := httptest.NewServer(router)
		defer server.Close()

		batchMetrics := []model.Metrics{
			{ID: "pg_batch_counter_1", MType: model.Counter, Delta: func() *int64 { v := int64(1000); return &v }()},
			{ID: "pg_batch_counter_2", MType: model.Counter, Delta: func() *int64 { v := int64(2000); return &v }()},
			{ID: "pg_batch_gauge_1", MType: model.Gauge, Value: func() *float64 { v := 5.55555; return &v }()},
			{ID: "pg_batch_gauge_2", MType: model.Gauge, Value: func() *float64 { v := 6.66666; return &v }()},
		}

		jsonData, err := json.Marshal(batchMetrics)
		require.NoError(t, err, "Failed to marshal batch metrics")

		req, err := http.NewRequest(http.MethodPost, server.URL+config.UpdatesPath, bytes.NewBuffer(jsonData))
		require.NoError(t, err, "Failed to create request")
		req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "Failed to send request")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Expected status 200, got %d", resp.StatusCode)

		allMetrics := postgresRepo.GetAllMetrics(context.Background())
		require.GreaterOrEqual(t, len(allMetrics), 4, "Should have at least 4 metrics in database")

		counter1, err := postgresRepo.GetMetric(context.Background(), &model.Metrics{ID: "pg_batch_counter_1", MType: model.Counter})
		require.NoError(t, err, "Failed to get counter1 from database")
		require.Equal(t, int64(1000), *counter1.Delta, "Counter1 delta should match")

		counter2, err := postgresRepo.GetMetric(context.Background(), &model.Metrics{ID: "pg_batch_counter_2", MType: model.Counter})
		require.NoError(t, err, "Failed to get counter2 from database")
		require.Equal(t, int64(2000), *counter2.Delta, "Counter2 delta should match")

		gauge1, err := postgresRepo.GetMetric(context.Background(), &model.Metrics{ID: "pg_batch_gauge_1", MType: model.Gauge})
		require.NoError(t, err, "Failed to get gauge1 from database")
		require.Equal(t, 5.55555, *gauge1.Value, "Gauge1 value should match")

		gauge2, err := postgresRepo.GetMetric(context.Background(), &model.Metrics{ID: "pg_batch_gauge_2", MType: model.Gauge})
		require.NoError(t, err, "Failed to get gauge2 from database")
		require.Equal(t, 6.66666, *gauge2.Value, "Gauge2 value should match")

		batch2 := []model.Metrics{
			{ID: "pg_batch_counter_1", MType: model.Counter, Delta: func() *int64 { v := int64(500); return &v }()},
		}

		jsonData2, err := json.Marshal(batch2)
		require.NoError(t, err, "Failed to marshal second batch")

		req2, err := http.NewRequest(http.MethodPost, server.URL+config.UpdatesPath, bytes.NewBuffer(jsonData2))
		require.NoError(t, err, "Failed to create second request")
		req2.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)

		resp2, err := http.DefaultClient.Do(req2)
		require.NoError(t, err, "Failed to send second request")
		resp2.Body.Close()

		require.Equal(t, http.StatusOK, resp2.StatusCode, "Expected status 200 for second batch")

		counter1Updated, err := postgresRepo.GetMetric(context.Background(), &model.Metrics{ID: "pg_batch_counter_1", MType: model.Counter})
		require.NoError(t, err, "Failed to get updated counter1 from database")
		require.Equal(t, int64(1500), *counter1Updated.Delta, "Counter1 should accumulate: 1000 + 500 = 1500")
	})
}

func TestHashValidationMiddlewareIntegration(t *testing.T) {
	testKey := "test-hash-key"
	logger := zaptest.NewLogger(t).Sugar()

	config.SetConfig(&config.Config{
		Key: testKey,
	}, logger)

	storage := repository.NewMemStorage(logger)
	metricsService := service.NewMetricsService(storage)
	handlers := handler.NewHandlers(metricsService, logger)

	router := chi.NewRouter()
	router.Use(handler.HashValidationMiddleware(logger))
	router.Route(config.CommonPath, func(r chi.Router) {
		r.Route(config.UpdatePath, func(r chi.Router) {
			r.Post("/", handlers.UpdateMetricHandlerByJSON)
		})
	})

	server := httptest.NewServer(router)
	defer server.Close()

	t.Run("valid hash", func(t *testing.T) {
		metric := model.Metrics{
			ID:    "test_hash_metric",
			MType: model.Gauge,
			Value: func() *float64 { v := 123.45; return &v }(),
		}

		jsonData, err := json.Marshal(metric)
		require.NoError(t, err)

		expectedHash := hash.ComputeHash(testKey, jsonData)

		req, err := http.NewRequest(http.MethodPost, server.URL+config.UpdatePath, bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)
		req.Header.Set(config.HashSHA256Header, expectedHash)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Request should succeed with valid hash")
		require.NotEmpty(t, resp.Header.Get(config.HashSHA256Header), "Response should contain hash header")
	})

	t.Run("invalid hash", func(t *testing.T) {
		metric := model.Metrics{
			ID:    "test_hash_invalid",
			MType: model.Gauge,
			Value: func() *float64 { v := 456.78; return &v }(),
		}

		jsonData, err := json.Marshal(metric)
		require.NoError(t, err)

		invalidHash := "invalid-hash-value"

		req, err := http.NewRequest(http.MethodPost, server.URL+config.UpdatePath, bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)
		req.Header.Set(config.HashSHA256Header, invalidHash)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Request should fail with invalid hash")
		body, _ := io.ReadAll(resp.Body)
		require.Contains(t, string(body), "Invalid hash", "Error message should indicate invalid hash")
	})
}

func TestCryptoAndHashIntegration(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	pubPath, privPath := generateRSAKeyPairFilesForServer(t)

	testKey := "test-crypto-hash-key"
	config.SetConfig(&config.Config{
		Key:       testKey,
		CryptoKey: privPath,
	}, logger)

	storage := repository.NewMemStorage(logger)
	metricsService := service.NewMetricsService(storage)
	handlers := handler.NewHandlers(metricsService, logger)

	router := chi.NewRouter()
	router.Use(
		handler.DecryptCryptoMiddleware(logger),
		handler.HashValidationMiddleware(logger),
	)
	router.Route(config.CommonPath, func(r chi.Router) {
		r.Route(config.UpdatePath, func(r chi.Router) {
			r.Post("/", handlers.UpdateMetricHandlerByJSON)
		})
	})

	server := httptest.NewServer(router)
	defer server.Close()

	metric := model.Metrics{
		ID:    "crypto_hash_metric",
		MType: model.Gauge,
		Value: func() *float64 { v := 99.99; return &v }(),
	}

	jsonData, err := json.Marshal(metric)
	require.NoError(t, err)

	hashValue := hash.ComputeHash(testKey, jsonData)

	encKey, ciphertext, err := encryption.EncryptHybrid(pubPath, jsonData)
	require.NoError(t, err)

	payload := struct {
		Key  string `json:"key"`
		Data string `json:"data"`
	}{
		Key:  base64.StdEncoding.EncodeToString(encKey),
		Data: base64.StdEncoding.EncodeToString(ciphertext),
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, server.URL+config.UpdatePath, bytes.NewBuffer(payloadBytes))
	require.NoError(t, err)
	req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)
	req.Header.Set(config.HashSHA256Header, hashValue)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Request should succeed with valid crypto and hash")

	savedMetric, err := storage.GetMetric(context.Background(), &model.Metrics{
		ID:    "crypto_hash_metric",
		MType: model.Gauge,
	})
	require.NoError(t, err)
	require.NotNil(t, savedMetric.Value)
	require.Equal(t, 99.99, *savedMetric.Value)
}

func generateRSAKeyPairFilesForServer(t *testing.T) (pubPath, privPath string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)
	pubBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}

	tempDir := t.TempDir()
	privPath = tempDir + "/private.pem"
	pubPath = tempDir + "/public.pem"

	require.NoError(t, os.WriteFile(privPath, pem.EncodeToMemory(privBlock), 0600))
	require.NoError(t, os.WriteFile(pubPath, pem.EncodeToMemory(pubBlock), 0644))

	return pubPath, privPath
}

func TestServerAuditIntegration(t *testing.T) {
	t.Run("file audit observer integration", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "audit_integration_*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())
		tempFile.Close()

		logger := zaptest.NewLogger(t).Sugar()
		storage := repository.NewMemStorage(logger)
		metricsService := service.NewMetricsService(storage)

		ctx := context.Background()
		fileObserver := audit.NewFileAuditObserver(ctx, tempFile.Name(), logger)
		metricsService.RegisterObserver(fileObserver)
		defer fileObserver.Close()

		handlers := handler.NewHandlers(metricsService, logger)
		router := chi.NewRouter()
		router.Use(handler.GzipDecompressMiddleware(logger))
		router.Route(config.CommonPath, func(r chi.Router) {
			r.Route(config.UpdatePath, func(r chi.Router) {
				r.Post("/{metricType}/{metricName}/{metricValue}", handlers.UpdateMetricHandlerByURL)
				r.Post("/", handlers.UpdateMetricHandlerByJSON)
			})
			r.Route(config.UpdatesPath, func(r chi.Router) {
				r.Post("/", handlers.UpdateMetricsBatchHandler)
			})
		})

		server := httptest.NewServer(router)
		defer server.Close()

		req, err := http.NewRequest(http.MethodPost, server.URL+"/update/counter/audit_test_counter/10", nil)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "text/plain")
		req.RemoteAddr = "192.168.1.100:12345"

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		time.Sleep(200 * time.Millisecond)
		fileObserver.Close()

		content, err := os.ReadFile(tempFile.Name())
		require.NoError(t, err)
		require.NotEmpty(t, content)

		var auditEvent audit.AuditEvent
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.Len(t, lines, 1)

		err = json.Unmarshal([]byte(lines[0]), &auditEvent)
		require.NoError(t, err)
		require.Contains(t, auditEvent.MetricsIDs, "audit_test_counter")
		require.NotEmpty(t, auditEvent.IPAddress)
		require.NotZero(t, auditEvent.Timestamp)
	})

	t.Run("url audit observer integration", func(t *testing.T) {
		var receivedEvents []audit.AuditEvent
		var mu sync.Mutex

		auditServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, config.ContentTypeJSON, r.Header.Get(config.ContentTypeHeader))

			var event audit.AuditEvent
			err := json.NewDecoder(r.Body).Decode(&event)
			require.NoError(t, err)

			mu.Lock()
			receivedEvents = append(receivedEvents, event)
			mu.Unlock()

			w.WriteHeader(http.StatusOK)
		}))
		defer auditServer.Close()

		logger := zaptest.NewLogger(t).Sugar()
		storage := repository.NewMemStorage(logger)
		metricsService := service.NewMetricsService(storage)

		ctx := context.Background()
		urlObserver, err := audit.NewURLAuditObserver(ctx, auditServer.URL, logger)
		require.NoError(t, err)
		metricsService.RegisterObserver(urlObserver)
		defer urlObserver.Close()

		handlers := handler.NewHandlers(metricsService, logger)
		router := chi.NewRouter()
		router.Use(handler.GzipDecompressMiddleware(logger))
		router.Route(config.CommonPath, func(r chi.Router) {
			r.Route(config.UpdatePath, func(r chi.Router) {
				r.Post("/{metricType}/{metricName}/{metricValue}", handlers.UpdateMetricHandlerByURL)
			})
		})

		server := httptest.NewServer(router)
		defer server.Close()

		req, err := http.NewRequest(http.MethodPost, server.URL+"/update/gauge/audit_test_gauge/42.5", nil)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "text/plain")
		req.RemoteAddr = "10.0.0.1:54321"

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		time.Sleep(300 * time.Millisecond)
		urlObserver.Close()

		mu.Lock()
		require.Len(t, receivedEvents, 1)
		require.Contains(t, receivedEvents[0].MetricsIDs, "audit_test_gauge")
		require.NotEmpty(t, receivedEvents[0].IPAddress)
		require.NotZero(t, receivedEvents[0].Timestamp)
		mu.Unlock()
	})

	t.Run("batch update with file audit", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "audit_batch_*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())
		tempFile.Close()

		logger := zaptest.NewLogger(t).Sugar()
		storage := repository.NewMemStorage(logger)
		metricsService := service.NewMetricsService(storage)

		ctx := context.Background()
		fileObserver := audit.NewFileAuditObserver(ctx, tempFile.Name(), logger)
		metricsService.RegisterObserver(fileObserver)
		defer fileObserver.Close()

		handlers := handler.NewHandlers(metricsService, logger)
		router := chi.NewRouter()
		router.Use(handler.GzipDecompressMiddleware(logger))
		router.Route(config.CommonPath, func(r chi.Router) {
			r.Route(config.UpdatesPath, func(r chi.Router) {
				r.Post("/", handlers.UpdateMetricsBatchHandler)
			})
		})

		server := httptest.NewServer(router)
		defer server.Close()

		batchMetrics := []model.Metrics{
			{ID: "batch_counter_1", MType: model.Counter, Delta: func() *int64 { v := int64(100); return &v }()},
			{ID: "batch_gauge_1", MType: model.Gauge, Value: func() *float64 { v := 3.14; return &v }()},
		}

		jsonData, err := json.Marshal(batchMetrics)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPost, server.URL+config.UpdatesPath, bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)
		req.RemoteAddr = "172.16.0.1:8080"

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		time.Sleep(200 * time.Millisecond)
		fileObserver.Close()

		content, err := os.ReadFile(tempFile.Name())
		require.NoError(t, err)

		var auditEvent audit.AuditEvent
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.Len(t, lines, 1)

		err = json.Unmarshal([]byte(lines[0]), &auditEvent)
		require.NoError(t, err)
		require.Len(t, auditEvent.MetricsIDs, 2)
		require.Contains(t, auditEvent.MetricsIDs, "batch_counter_1")
		require.Contains(t, auditEvent.MetricsIDs, "batch_gauge_1")
		require.NotEmpty(t, auditEvent.IPAddress)
	})

	t.Run("multiple observers integration", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "audit_multi_*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())
		tempFile.Close()

		var urlEvents []audit.AuditEvent
		var mu sync.Mutex

		auditServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var event audit.AuditEvent
			json.NewDecoder(r.Body).Decode(&event)

			mu.Lock()
			urlEvents = append(urlEvents, event)
			mu.Unlock()

			w.WriteHeader(http.StatusOK)
		}))
		defer auditServer.Close()

		logger := zaptest.NewLogger(t).Sugar()
		storage := repository.NewMemStorage(logger)
		metricsService := service.NewMetricsService(storage)

		ctx := context.Background()
		fileObserver := audit.NewFileAuditObserver(ctx, tempFile.Name(), logger)
		urlObserver, err := audit.NewURLAuditObserver(ctx, auditServer.URL, logger)
		require.NoError(t, err)
		metricsService.RegisterObserver(fileObserver)
		metricsService.RegisterObserver(urlObserver)
		defer fileObserver.Close()
		defer urlObserver.Close()

		handlers := handler.NewHandlers(metricsService, logger)
		router := chi.NewRouter()
		router.Use(handler.GzipDecompressMiddleware(logger))
		router.Route(config.CommonPath, func(r chi.Router) {
			r.Route(config.UpdatePath, func(r chi.Router) {
				r.Post("/{metricType}/{metricName}/{metricValue}", handlers.UpdateMetricHandlerByURL)
			})
		})

		server := httptest.NewServer(router)
		defer server.Close()

		req, err := http.NewRequest(http.MethodPost, server.URL+"/update/counter/multi_test/5", nil)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "text/plain")
		req.RemoteAddr = "192.168.0.50:9999"

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		time.Sleep(300 * time.Millisecond)
		fileObserver.Close()
		urlObserver.Close()

		content, err := os.ReadFile(tempFile.Name())
		require.NoError(t, err)
		require.NotEmpty(t, content)

		var fileEvent audit.AuditEvent
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.Len(t, lines, 1)
		json.Unmarshal([]byte(lines[0]), &fileEvent)

		mu.Lock()
		require.Len(t, urlEvents, 1)
		urlEvent := urlEvents[0]
		mu.Unlock()

		require.Contains(t, fileEvent.MetricsIDs, "multi_test")
		require.Contains(t, urlEvent.MetricsIDs, "multi_test")
		require.Equal(t, fileEvent.IPAddress, urlEvent.IPAddress)
		require.NotEmpty(t, fileEvent.IPAddress)
	})
}

func truncateTableForTest(ctx context.Context, dsn string) error {
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return err
	}

	db := stdlib.OpenDB(*config)
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return err
	}

	var exists bool
	err = db.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'metrics')").Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	_, err = db.ExecContext(ctx, "TRUNCATE TABLE metrics")
	return err
}

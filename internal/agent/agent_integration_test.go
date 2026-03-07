package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/prbllm/go-metrics/internal/compression"
	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/prbllm/go-metrics/internal/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestAgentJSONIntegration(t *testing.T) {
	receivedMetrics := make([]model.Metrics, 0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method, "Expected POST method")
		require.Equal(t, config.ContentTypeJSON, r.Header.Get(config.ContentTypeHeader), "Expected JSON content type")
		require.Equal(t, config.ContentEncodingGzip, r.Header.Get(config.ContentEncodingHeader), "Expected gzip content encoding")
		require.Equal(t, config.UpdatePath, r.URL.Path, "Expected /update path")

		decompressedBody, err := compression.DecompressReader(r.Body)
		require.NoError(t, err, "Failed to decompress gzip data")

		var metric model.Metrics
		err = json.NewDecoder(bytes.NewReader(decompressedBody)).Decode(&metric)
		require.NoError(t, err, "Failed to decode gzipped JSON metric")

		require.NotEmpty(t, metric.ID, "Metric ID should not be empty")
		require.NotEmpty(t, metric.MType, "Metric type should not be empty")

		receivedMetrics = append(receivedMetrics, metric)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t).Sugar()
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost:          serverURL.Host,
		AgentPollInterval:   time.Duration(1) * time.Second,
		AgentReportInterval: time.Duration(2) * time.Second,
	}
	config.SetConfig(cfg, logger)
	collector := NewRuntimeMetricsCollector(logger)
	ag := NewAgent(http.DefaultClient, collector, logger)

	metrics := collector.CollectRuntimeMetrics()
	err := ag.sendMetricsJSON(context.Background(), metrics)
	require.NoError(t, err, "Failed to send metrics via JSON")

	require.NotEmpty(t, receivedMetrics, "Should have received some metrics")

	hasGauge := false
	hasCounter := false
	for _, metric := range receivedMetrics {
		if metric.MType == model.Gauge {
			hasGauge = true
		}
		if metric.MType == model.Counter {
			hasCounter = true
		}
	}
	require.True(t, hasGauge, "Should have received gauge metrics")
	require.True(t, hasCounter, "Should have received counter metrics")
}

func TestAgentJSONIntegration_WithHashHeader(t *testing.T) {
	testKey := "integration-test-key"
	receivedMetrics := make([]model.Metrics, 0)
	var receivedHashes []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method, "Expected POST method")
		require.Equal(t, config.ContentTypeJSON, r.Header.Get(config.ContentTypeHeader), "Expected JSON content type")
		require.Equal(t, config.ContentEncodingGzip, r.Header.Get(config.ContentEncodingHeader), "Expected gzip content encoding")
		require.Equal(t, config.UpdatePath, r.URL.Path, "Expected /update path")

		hashHeader := r.Header.Get(config.HashSHA256Header)
		require.NotEmpty(t, hashHeader, "HashSHA256 header should be present when key is set")
		receivedHashes = append(receivedHashes, hashHeader)

		decompressedBody, err := compression.DecompressReader(r.Body)
		require.NoError(t, err, "Failed to decompress gzip data")

		var metric model.Metrics
		err = json.NewDecoder(bytes.NewReader(decompressedBody)).Decode(&metric)
		require.NoError(t, err, "Failed to decode gzipped JSON metric")

		require.NotEmpty(t, metric.ID, "Metric ID should not be empty")
		require.NotEmpty(t, metric.MType, "Metric type should not be empty")

		expectedHash := testutil.MustHashFromJSON(t, testKey, metric)
		require.Equal(t, expectedHash, hashHeader, "Hash should be computed correctly based on key and original JSON body")

		receivedMetrics = append(receivedMetrics, metric)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t).Sugar()
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost:          serverURL.Host,
		AgentPollInterval:   time.Duration(1) * time.Second,
		AgentReportInterval: time.Duration(2) * time.Second,
		Key:                 testKey,
	}
	config.SetConfig(cfg, logger)
	collector := NewRuntimeMetricsCollector(logger)
	ag := NewAgent(http.DefaultClient, collector, logger)

	metrics := collector.CollectRuntimeMetrics()
	err := ag.sendMetricsJSON(context.Background(), metrics)
	require.NoError(t, err, "Failed to send metrics via JSON")

	require.NotEmpty(t, receivedMetrics, "Should have received some metrics")
	require.NotEmpty(t, receivedHashes, "Should have received hash headers")
	require.Equal(t, len(receivedMetrics), len(receivedHashes), "Should have hash header for each metric")
}

func TestAgentBatchJSONIntegration(t *testing.T) {
	receivedMetrics := make([]model.Metrics, 0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method, "Expected POST method")
		require.Equal(t, config.ContentTypeJSON, r.Header.Get(config.ContentTypeHeader), "Expected JSON content type")
		require.Equal(t, config.ContentEncodingGzip, r.Header.Get(config.ContentEncodingHeader), "Expected gzip content encoding")
		require.Equal(t, config.UpdatesPath, r.URL.Path, "Expected /updates path")

		decompressedBody, err := compression.DecompressReader(r.Body)
		require.NoError(t, err, "Failed to decompress gzip data")

		var metrics []model.Metrics
		err = json.NewDecoder(bytes.NewReader(decompressedBody)).Decode(&metrics)
		require.NoError(t, err, "Failed to decode gzipped JSON metrics batch")

		require.NotEmpty(t, metrics, "Should have received metrics")
		require.Greater(t, len(metrics), 0, "Should have received at least one metric")

		receivedMetrics = append(receivedMetrics, metrics...)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t).Sugar()
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost:          serverURL.Host,
		AgentPollInterval:   time.Duration(1) * time.Second,
		AgentReportInterval: time.Duration(2) * time.Second,
	}
	config.SetConfig(cfg, logger)
	collector := NewRuntimeMetricsCollector(logger)
	ag := NewAgent(http.DefaultClient, collector, logger)

	metrics := collector.CollectRuntimeMetrics()
	err := ag.sendMetricsBatchJSON(context.Background(), metrics)
	require.NoError(t, err, "Failed to send metrics batch via JSON")

	require.NotEmpty(t, receivedMetrics, "Should have received some metrics")

	hasGauge := false
	hasCounter := false
	for _, metric := range receivedMetrics {
		if metric.MType == model.Gauge {
			hasGauge = true
		}
		if metric.MType == model.Counter {
			hasCounter = true
		}
	}
	require.True(t, hasGauge, "Should have received gauge metrics")
	require.True(t, hasCounter, "Should have received counter metrics")
}

func TestAgentBatchJSONIntegration_WithHashHeader(t *testing.T) {
	testKey := "integration-batch-test-key"
	receivedMetrics := make([]model.Metrics, 0)
	var receivedHash string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method, "Expected POST method")
		require.Equal(t, config.ContentTypeJSON, r.Header.Get(config.ContentTypeHeader), "Expected JSON content type")
		require.Equal(t, config.ContentEncodingGzip, r.Header.Get(config.ContentEncodingHeader), "Expected gzip content encoding")
		require.Equal(t, config.UpdatesPath, r.URL.Path, "Expected /updates path")

		receivedHash = r.Header.Get(config.HashSHA256Header)
		require.NotEmpty(t, receivedHash, "HashSHA256 header should be present when key is set")

		decompressedBody, err := compression.DecompressReader(r.Body)
		require.NoError(t, err, "Failed to decompress gzip data")

		var metrics []model.Metrics
		err = json.NewDecoder(bytes.NewReader(decompressedBody)).Decode(&metrics)
		require.NoError(t, err, "Failed to decode gzipped JSON metrics batch")

		require.NotEmpty(t, metrics, "Should have received metrics")
		require.Greater(t, len(metrics), 0, "Should have received at least one metric")

		expectedHash := testutil.MustHashFromJSON(t, testKey, metrics)
		require.Equal(t, expectedHash, receivedHash, "Hash should be computed correctly based on key and original JSON body")

		receivedMetrics = append(receivedMetrics, metrics...)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t).Sugar()
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost:          serverURL.Host,
		AgentPollInterval:   time.Duration(1) * time.Second,
		AgentReportInterval: time.Duration(2) * time.Second,
		Key:                 testKey,
	}
	config.SetConfig(cfg, logger)
	collector := NewRuntimeMetricsCollector(logger)
	ag := NewAgent(http.DefaultClient, collector, logger)

	metrics := collector.CollectRuntimeMetrics()
	err := ag.sendMetricsBatchJSON(context.Background(), metrics)
	require.NoError(t, err, "Failed to send metrics batch via JSON")

	require.NotEmpty(t, receivedMetrics, "Should have received some metrics")
	require.NotEmpty(t, receivedHash, "Should have received hash header")
}

func TestAgentBatchJSONWrongStatusCode(t *testing.T) {
	commonValue := float64(1.0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t).Sugar()
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost:          serverURL.Host,
		AgentPollInterval:   time.Duration(1) * time.Second,
		AgentReportInterval: time.Duration(2) * time.Second,
	}
	config.SetConfig(cfg, logger)
	ag := NewAgent(http.DefaultClient, nil, logger)

	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
	}

	err := ag.sendMetricsBatchJSON(context.Background(), metrics)
	require.Error(t, err, "Expected error for retriable status code (500)")
	require.Contains(t, err.Error(), "error sending metrics batch")
}

func TestAgentJSONIntegration_RetryOnTemporaryNetworkError(t *testing.T) {
	receivedMetrics := make([]model.Metrics, 0)
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		require.Equal(t, http.MethodPost, r.Method, "Expected POST method")
		require.Equal(t, config.ContentTypeJSON, r.Header.Get(config.ContentTypeHeader), "Expected JSON content type")
		require.Equal(t, config.ContentEncodingGzip, r.Header.Get(config.ContentEncodingHeader), "Expected gzip content encoding")
		require.Equal(t, config.UpdatePath, r.URL.Path, "Expected /update path")

		decompressedBody, err := compression.DecompressReader(r.Body)
		require.NoError(t, err, "Failed to decompress gzip data")

		var metric model.Metrics
		err = json.NewDecoder(bytes.NewReader(decompressedBody)).Decode(&metric)
		require.NoError(t, err, "Failed to decode gzipped JSON metric")

		require.NotEmpty(t, metric.ID, "Metric ID should not be empty")
		require.NotEmpty(t, metric.MType, "Metric type should not be empty")

		receivedMetrics = append(receivedMetrics, metric)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t).Sugar()
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost:          serverURL.Host,
		AgentPollInterval:   time.Duration(1) * time.Second,
		AgentReportInterval: time.Duration(2) * time.Second,
	}
	config.SetConfig(cfg, logger)
	ag := NewAgent(http.DefaultClient, nil, logger)

	commonValue := float64(1.0)
	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
	}

	err := ag.sendMetricsJSON(context.Background(), metrics)
	require.NoError(t, err, "Should succeed after retry")

	require.Equal(t, 2, attempts, "Should have retried once (total 2 attempts: 1 initial + 1 retry)")
	require.Len(t, receivedMetrics, 1, "Should have received one metric after retry")
	require.Equal(t, "test_metric", receivedMetrics[0].ID, "Received metric ID should match")
}

func TestAgentBatchJSONIntegration_RetryOnTemporaryNetworkError(t *testing.T) {
	receivedMetrics := make([]model.Metrics, 0)
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}

		require.Equal(t, http.MethodPost, r.Method, "Expected POST method")
		require.Equal(t, config.ContentTypeJSON, r.Header.Get(config.ContentTypeHeader), "Expected JSON content type")
		require.Equal(t, config.ContentEncodingGzip, r.Header.Get(config.ContentEncodingHeader), "Expected gzip content encoding")
		require.Equal(t, config.UpdatesPath, r.URL.Path, "Expected /updates path")

		decompressedBody, err := compression.DecompressReader(r.Body)
		require.NoError(t, err, "Failed to decompress gzip data")

		var metrics []model.Metrics
		err = json.NewDecoder(bytes.NewReader(decompressedBody)).Decode(&metrics)
		require.NoError(t, err, "Failed to decode gzipped JSON metrics batch")

		require.NotEmpty(t, metrics, "Should have received metrics")
		require.Greater(t, len(metrics), 0, "Should have received at least one metric")

		receivedMetrics = append(receivedMetrics, metrics...)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t).Sugar()
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost:          serverURL.Host,
		AgentPollInterval:   time.Duration(1) * time.Second,
		AgentReportInterval: time.Duration(2) * time.Second,
	}
	config.SetConfig(cfg, logger)
	collector := NewRuntimeMetricsCollector(logger)
	ag := NewAgent(http.DefaultClient, collector, logger)

	metrics := collector.CollectRuntimeMetrics()
	err := ag.sendMetricsBatchJSON(context.Background(), metrics)
	require.NoError(t, err, "Should succeed after retry")

	require.Equal(t, 3, attempts, "Should have retried 2 times (total 3 attempts)")
	require.NotEmpty(t, receivedMetrics, "Should have received metrics after retry")

	hasGauge := false
	hasCounter := false
	for _, metric := range receivedMetrics {
		if metric.MType == model.Gauge {
			hasGauge = true
		}
		if metric.MType == model.Counter {
			hasCounter = true
		}
	}
	require.True(t, hasGauge, "Should have received gauge metrics")
	require.True(t, hasCounter, "Should have received counter metrics")
}

func TestAgentJSONIntegration_RetryExhausted(t *testing.T) {
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t).Sugar()
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost:          serverURL.Host,
		AgentPollInterval:   time.Duration(1) * time.Second,
		AgentReportInterval: time.Duration(2) * time.Second,
	}
	config.SetConfig(cfg, logger)
	ag := NewAgent(http.DefaultClient, nil, logger)

	commonValue := float64(1.0)
	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
	}

	err := ag.sendMetricsJSON(context.Background(), metrics)
	require.NoError(t, err, "Should not return error (skips metrics after retry exhaustion)")

	require.Equal(t, 4, attempts, "Should have exhausted all retries (1 initial + 3 retries)")
}

func TestAgentBatchJSONIntegration_RetryExhausted(t *testing.T) {
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusGatewayTimeout)
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t).Sugar()
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost:          serverURL.Host,
		AgentPollInterval:   time.Duration(1) * time.Second,
		AgentReportInterval: time.Duration(2) * time.Second,
	}
	config.SetConfig(cfg, logger)
	collector := NewRuntimeMetricsCollector(logger)
	ag := NewAgent(http.DefaultClient, collector, logger)

	metrics := collector.CollectRuntimeMetrics()
	err := ag.sendMetricsBatchJSON(context.Background(), metrics)
	require.Error(t, err, "Should return error after retry exhaustion")
	require.Contains(t, err.Error(), "error sending metrics batch")

	require.Equal(t, 4, attempts, "Should have exhausted all retries (1 initial + 3 retries)")
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/prbllm/go-metrics/internal/agent"
	"github.com/prbllm/go-metrics/internal/compression"
	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestFullIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	context, cancel := context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
	defer cancel()

	logger := zaptest.NewLogger(t).Sugar()
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost:          serverURL.Host,
		AgentPollInterval:   time.Duration(1) * time.Second,
		AgentReportInterval: time.Duration(2) * time.Second,
	}
	config.SetConfig(cfg, logger)
	collector := agent.NewRuntimeMetricsCollector(logger)
	agent := agent.NewAgent(http.DefaultClient, collector, logger)
	go agent.Start(context)
	<-context.Done()
}

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
	collector := agent.NewRuntimeMetricsCollector(logger)
	agent := agent.NewAgent(http.DefaultClient, collector, logger)

	metrics := collector.Collect()
	err := agent.SendMetricsJSON(context.Background(), metrics)
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
	collector := agent.NewRuntimeMetricsCollector(logger)
	agent := agent.NewAgent(http.DefaultClient, collector, logger)

	metrics := collector.Collect()
	err := agent.SendMetricsBatchJSON(context.Background(), metrics)
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
	agent := agent.NewAgent(http.DefaultClient, nil, logger)

	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
	}

	err := agent.SendMetricsBatchJSON(context.Background(), metrics)
	require.Error(t, err, "Expected error for wrong status code")
	require.Contains(t, err.Error(), "unexpected status code")
}

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prbllm/go-metrics/internal/agent"
	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/stretchr/testify/require"
)

func TestFullIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	context, cancel := context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
	defer cancel()

	collector := &agent.RuntimeMetricsCollector{}
	agent := agent.NewAgent(http.DefaultClient, collector, server.URL+"/update/", time.Duration(1)*time.Second, time.Duration(2)*time.Second)
	go agent.Start(context)
	<-context.Done()
}

func TestAgentJSONIntegration(t *testing.T) {
	receivedMetrics := make([]model.Metrics, 0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method, "Expected POST method")
		require.Equal(t, config.ContentTypeJSON, r.Header.Get(config.ContentTypeHeader), "Expected JSON content type")
		require.Equal(t, config.UpdatePath, r.URL.Path, "Expected /update path")

		var metric model.Metrics
		err := json.NewDecoder(r.Body).Decode(&metric)
		require.NoError(t, err, "Failed to decode JSON metric")

		require.NotEmpty(t, metric.ID, "Metric ID should not be empty")
		require.NotEmpty(t, metric.MType, "Metric type should not be empty")

		receivedMetrics = append(receivedMetrics, metric)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	collector := &agent.RuntimeMetricsCollector{}
	agent := agent.NewAgent(http.DefaultClient, collector, server.URL+config.UpdatePath, time.Duration(1)*time.Second, time.Duration(2)*time.Second)

	metrics := collector.Collect()
	err := agent.SendMetricsJSON(metrics)
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

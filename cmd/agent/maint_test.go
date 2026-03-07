package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/prbllm/go-metrics/internal/agent"
	"github.com/prbllm/go-metrics/internal/config"
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
	ag := agent.NewAgent(http.DefaultClient, collector, logger)
	go ag.Start(context)
	<-context.Done()
}

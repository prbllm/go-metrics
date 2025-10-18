package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prbllm/go-metrics/internal/agent"
	"github.com/stretchr/testify/require"
)

func TestFullIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		fmt.Println("Request: ", r.URL.Path)
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

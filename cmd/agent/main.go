package main

import (
	"net/http"
	"time"

	"github.com/prbllm/go-metrics/internal/agent"
	"github.com/prbllm/go-metrics/internal/config"
)

func main() {
	client := &http.Client{}
	collector := agent.RuntimeMetricsCollector{}
	agent := agent.NewAgent(client, collector, "http://"+config.ServerAddress+config.ServerPort+config.UpdatePath, time.Duration(config.AgentPollIntervalSeconds)*time.Second, time.Duration(config.AgentReportIntervalSeconds)*time.Second)
	agent.Start()
}

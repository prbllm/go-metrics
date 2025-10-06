package main

import (
	"context"
	"net/http"

	"github.com/prbllm/go-metrics/internal/agent"
	"github.com/prbllm/go-metrics/internal/config"
)

func main() {
	err := config.InitConfig("agent")
	if err != nil {
		panic(err)
	}

	collector := &agent.RuntimeMetricsCollector{}
	agent := agent.NewAgent(http.DefaultClient, collector, "http://"+config.GetConfig().ServerHost+config.CommonPath+config.UpdatePath, config.GetConfig().AgentPollInterval, config.GetConfig().AgentReportInterval)
	agent.Start(context.Background())
}

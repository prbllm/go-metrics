package main

import (
	"context"
	"net/http"
	"time"

	"github.com/prbllm/go-metrics/internal/agent"
	"github.com/prbllm/go-metrics/internal/config"
)

func main() {
	err := config.InitConfig("agent")
	if err != nil {
		panic(err)
	}

	collector := &agent.RuntimeMetricsCollector{}
	agent := agent.NewAgent(http.DefaultClient, collector, "http://"+config.GetConfig().ServerHost+config.CommonPath+config.UpdatePath, time.Duration(config.GetConfig().AgentPollInterval)*time.Second, time.Duration(config.GetConfig().AgentReportInterval)*time.Second)
	agent.Start(context.Background())
}

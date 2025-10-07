package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/prbllm/go-metrics/internal/agent"
	"github.com/prbllm/go-metrics/internal/config"
)

func main() {
	err := config.InitConfig("agent")
	if err != nil {
		fmt.Println("Error initializing config: ", err)
		os.Exit(1)
	}

	collector := &agent.RuntimeMetricsCollector{}
	agent := agent.NewAgent(http.DefaultClient, collector, "http://"+config.GetConfig().ServerHost+config.UpdatePath, config.GetConfig().AgentPollInterval, config.GetConfig().AgentReportInterval)
	agent.Start(context.Background())
}

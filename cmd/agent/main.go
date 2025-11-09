package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/prbllm/go-metrics/internal/agent"
	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/logger"
)

func main() {
	appLogger, err := logger.NewZapLogger()
	if err != nil {
		fmt.Println("Error initializing config logger: ", err)
		os.Exit(1)
	}

	err = config.InitConfig(config.AgentFlagsSet, appLogger)
	if err != nil {
		fmt.Println("Error initializing config: ", err)
		os.Exit(1)
	}

	collector := agent.NewRuntimeMetricsCollector(appLogger)
	agent := agent.NewAgent(http.DefaultClient, collector, appLogger)
	agent.Start(context.Background())
}

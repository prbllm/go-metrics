package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prbllm/go-metrics/internal/agent"
	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/logger"
	"github.com/prbllm/go-metrics/internal/versions"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
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

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	collector := agent.NewRuntimeMetricsCollector(appLogger)
	agent := agent.NewAgent(http.DefaultClient, collector, appLogger)
	versions.LogBuildInfo(buildVersion, buildDate, buildCommit, appLogger)
	agent.Start(ctx)
}

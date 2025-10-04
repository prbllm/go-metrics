package main

import (
	"net/http"

	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/handler"
	"github.com/prbllm/go-metrics/internal/repository"
	"github.com/prbllm/go-metrics/internal/service"
)

func main() {
	storage := repository.NewMemStorage()
	metricsService := service.NewMetricsService(storage)
	handlers := handler.NewHandlers(metricsService)

	mux := http.NewServeMux()
	mux.HandleFunc(config.NotFoundPath, handlers.NotFoundHandler)
	mux.HandleFunc(config.UpdatePath, handlers.UpdateHandler)

	err := http.ListenAndServe(config.ServerAddress, mux)
	if err != nil {
		panic(err)
	}
}

package main

import (
	"fmt"
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

	fmt.Println("Server starting on ", config.ServerPort)
	err := http.ListenAndServe(config.ServerPort, mux)
	if err != nil {
		panic(err)
	}
}

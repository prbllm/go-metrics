package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/handler"
	"github.com/prbllm/go-metrics/internal/logger"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/prbllm/go-metrics/internal/repository"
	"github.com/prbllm/go-metrics/internal/service"
)

func setupExampleServer() (*httptest.Server, *handler.Handlers) {
	logger, _ := logger.NewZapLogger()
	storage := repository.NewMemStorage(logger)
	metricsService := service.NewMetricsService(storage)
	handlers := handler.NewHandlers(metricsService, logger)

	router := chi.NewRouter()
	router.Get(config.PingPath, handlers.PingHandler)
	router.Route(config.CommonPath, func(r chi.Router) {
		r.Get("/", handlers.GetAllMetricsHandlerByURL)
		r.Route(config.UpdatePath, func(r chi.Router) {
			r.Post("/{metricType}/{metricName}/{metricValue}", handlers.UpdateMetricHandlerByURL)
			r.Post("/", handlers.UpdateMetricHandlerByJSON)
		})
		r.Route(config.ValuePath, func(r chi.Router) {
			r.Get("/{metricType}/{metricName}", handlers.GetValueHandlerByURL)
			r.Post("/", handlers.GetValueHandlerByJSON)
		})
	})

	server := httptest.NewServer(router)
	return server, handlers
}

// ExampleHandlers_UpdateMetricHandlerByURL демонстрирует обновление метрики через URL параметры.
func ExampleHandlers_UpdateMetricHandlerByURL() {
	server, _ := setupExampleServer()
	defer server.Close()

	// Обновление gauge метрики
	resp, err := http.Post(server.URL+"/update/gauge/Alloc/12345.67", "text/plain", nil)
	if err == nil {
		defer resp.Body.Close()
		fmt.Printf("Status: %d\n", resp.StatusCode)
	}

	// Обновление counter метрики
	resp, err = http.Post(server.URL+"/update/counter/PollCount/1", "text/plain", nil)
	if err == nil {
		defer resp.Body.Close()
		fmt.Printf("Status: %d\n", resp.StatusCode)
	}

	// Output:
	// Status: 200
	// Status: 200
}

// ExampleHandlers_UpdateMetricHandlerByJSON демонстрирует обновление метрики через JSON.
func ExampleHandlers_UpdateMetricHandlerByJSON() {
	server, _ := setupExampleServer()
	defer server.Close()

	// Обновление gauge метрики
	metric := model.Metrics{
		ID:    "Alloc",
		MType: model.Gauge,
		Value: func() *float64 { v := 12345.67; return &v }(),
	}
	body, _ := json.Marshal(metric)
	resp, err := http.Post(server.URL+"/update/", "application/json", bytes.NewReader(body))
	if err == nil {
		defer resp.Body.Close()
		fmt.Printf("Status: %d\n", resp.StatusCode)
	}

	// Обновление counter метрики
	counterMetric := model.Metrics{
		ID:    "PollCount",
		MType: model.Counter,
		Delta: func() *int64 { v := int64(1); return &v }(),
	}
	body, _ = json.Marshal(counterMetric)
	resp, err = http.Post(server.URL+"/update/", "application/json", bytes.NewReader(body))
	if err == nil {
		defer resp.Body.Close()
		fmt.Printf("Status: %d\n", resp.StatusCode)
	}

	// Output:
	// Status: 200
	// Status: 200
}

// ExampleHandlers_GetValueHandlerByURL демонстрирует получение значения метрики через URL параметры.
func ExampleHandlers_GetValueHandlerByURL() {
	server, _ := setupExampleServer()
	defer server.Close()

	// Сначала создаем метрику
	resp, _ := http.Post(server.URL+"/update/gauge/Alloc/12345.67", "text/plain", nil)
	if resp != nil {
		resp.Body.Close()
	}

	// Получаем значение метрики
	resp, err := http.Get(server.URL + "/value/gauge/Alloc")
	if err == nil {
		defer resp.Body.Close()

		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)
		fmt.Printf("Value: %s\n", buf.String())
	}

	// Output:
	// Value: 12345.67
}

// ExampleHandlers_GetValueHandlerByJSON демонстрирует получение значения метрики через JSON.
func ExampleHandlers_GetValueHandlerByJSON() {
	server, _ := setupExampleServer()
	defer server.Close()

	// Сначала создаем метрику
	metric := model.Metrics{
		ID:    "Alloc",
		MType: model.Gauge,
		Value: func() *float64 { v := 12345.67; return &v }(),
	}
	body, _ := json.Marshal(metric)
	resp, _ := http.Post(server.URL+"/update/", "application/json", bytes.NewReader(body))
	if resp != nil {
		resp.Body.Close()
	}

	// Получаем значение метрики
	requestMetric := model.Metrics{
		ID:    "Alloc",
		MType: model.Gauge,
	}
	body, _ = json.Marshal(requestMetric)
	resp, err := http.Post(server.URL+"/value/", "application/json", bytes.NewReader(body))
	if err == nil {
		defer resp.Body.Close()

		var result model.Metrics
		json.NewDecoder(resp.Body).Decode(&result)
		fmt.Printf("Value: %f\n", *result.Value)
	}

	// Output:
	// Value: 12345.670000
}

// ExampleHandlers_GetAllMetricsHandlerByURL демонстрирует получение всех метрик в HTML формате.
func ExampleHandlers_GetAllMetricsHandlerByURL() {
	server, _ := setupExampleServer()
	defer server.Close()

	// Создаем несколько метрик
	resp, _ := http.Post(server.URL+"/update/gauge/Alloc/12345.67", "text/plain", nil)
	if resp != nil {
		resp.Body.Close()
	}
	resp, _ = http.Post(server.URL+"/update/counter/PollCount/5", "text/plain", nil)
	if resp != nil {
		resp.Body.Close()
	}

	// Получаем все метрики
	resp, err := http.Get(server.URL + "/")
	if err == nil {
		defer resp.Body.Close()

		fmt.Printf("Status: %d\n", resp.StatusCode)
		fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))
	}

	// Output:
	// Status: 200
	// Content-Type: text/html; charset=utf-8
}

// ExampleHandlers_PingHandler демонстрирует проверку доступности хранилища метрик.
// Для in-memory хранилища Ping не поддерживается, поэтому возвращается ошибка.
func ExampleHandlers_PingHandler() {
	server, _ := setupExampleServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/ping")
	if err == nil {
		defer resp.Body.Close()

		fmt.Printf("Status: %d\n", resp.StatusCode)
	}

	// Output:
	// Status: 500
}

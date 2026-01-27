package benchmarks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/handler"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/prbllm/go-metrics/internal/repository"
	"github.com/prbllm/go-metrics/internal/service"
	"go.uber.org/zap/zaptest"
)

func setupBenchmarkRouter(b *testing.B) (*chi.Mux, *handler.Handlers) {
	logger := zaptest.NewLogger(b).Sugar()
	storage := repository.NewMemStorage(logger)
	svc := service.NewMetricsService(storage)
	handlers := handler.NewHandlers(svc, logger)

	router := chi.NewRouter()
	router.Route(config.CommonPath, func(r chi.Router) {
		r.Get("/", handlers.GetAllMetricsHandlerByURL)
		r.Route(config.UpdatePath, func(r chi.Router) {
			r.Post("/", handlers.UpdateMetricHandlerByJSON)
		})
		r.Route(config.UpdatesPath, func(r chi.Router) {
			r.Post("/", handlers.UpdateMetricsBatchHandler)
		})
	})

	return router, handlers
}

func BenchmarkHandler_UpdateMetricByJSON(b *testing.B) {
	router, _ := setupBenchmarkRouter(b)

	value := 123.45
	metric := model.Metrics{
		MType: model.Gauge,
		ID:    "test_gauge",
		Value: &value,
	}

	body, _ := json.Marshal(metric)

	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, config.UpdatePath, bytes.NewReader(body))
		req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
	}
}

func BenchmarkHandler_GetAllMetrics(b *testing.B) {
	ctx := context.Background()

	sizes := []int{10, 100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			testStorage := repository.NewMemStorage(zaptest.NewLogger(b).Sugar())
			testSvc := service.NewMetricsService(testStorage)
			testHandlers := handler.NewHandlers(testSvc, zaptest.NewLogger(b).Sugar())

			testRouter := chi.NewRouter()
			testRouter.Route(config.CommonPath, func(r chi.Router) {
				r.Get("/", testHandlers.GetAllMetricsHandlerByURL)
			})

			metrics := GenerateMetrics(size)
			for _, m := range metrics {
				_ = testSvc.UpdateMetricByStruct(ctx, m)
			}

			for b.Loop() {
				req := httptest.NewRequest(http.MethodGet, config.CommonPath, nil)
				rr := httptest.NewRecorder()
				testRouter.ServeHTTP(rr, req)
			}
		})
	}
}

func BenchmarkHandler_UpdateMetricsBatch(b *testing.B) {
	router, _ := setupBenchmarkRouter(b)

	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("batch_%d", size), func(b *testing.B) {
			metrics := GenerateMetricsSlice(size)
			body, _ := json.Marshal(metrics)

			for b.Loop() {
				req := httptest.NewRequest(http.MethodPost, config.UpdatesPath, bytes.NewReader(body))
				req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)
				rr := httptest.NewRecorder()
				router.ServeHTTP(rr, req)
			}
		})
	}
}

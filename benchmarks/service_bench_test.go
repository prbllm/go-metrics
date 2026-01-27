package benchmarks

import (
	"context"
	"fmt"
	"testing"

	"github.com/prbllm/go-metrics/internal/model"
	"github.com/prbllm/go-metrics/internal/repository"
	"github.com/prbllm/go-metrics/internal/service"
	"go.uber.org/zap/zaptest"
)

func BenchmarkService_UpdateMetric(b *testing.B) {
	storage := repository.NewMemStorage(zaptest.NewLogger(b).Sugar())
	svc := service.NewMetricsService(storage)
	ctx := context.Background()

	for b.Loop() {
		_ = svc.UpdateMetric(ctx, model.Gauge, "test_gauge", "123.45")
	}
}

func BenchmarkService_UpdateMetricByStruct(b *testing.B) {
	storage := repository.NewMemStorage(zaptest.NewLogger(b).Sugar())
	svc := service.NewMetricsService(storage)
	ctx := context.Background()

	value := 123.45
	metric := &model.Metrics{
		MType: model.Gauge,
		ID:    "test_gauge",
		Value: &value,
	}

	for b.Loop() {
		_ = svc.UpdateMetricByStruct(ctx, metric)
	}
}

func BenchmarkService_GetMetric(b *testing.B) {
	storage := repository.NewMemStorage(zaptest.NewLogger(b).Sugar())
	svc := service.NewMetricsService(storage)
	ctx := context.Background()

	_ = svc.UpdateMetric(ctx, model.Gauge, "test_gauge", "123.45")

	for b.Loop() {
		_, _ = svc.GetMetric(ctx, model.Gauge, "test_gauge")
	}
}

func BenchmarkService_GetAllMetrics(b *testing.B) {
	ctx := context.Background()

	sizes := []int{10, 100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			testStorage := repository.NewMemStorage(zaptest.NewLogger(b).Sugar())
			testSvc := service.NewMetricsService(testStorage)

			metrics := GenerateMetrics(size)
			for _, m := range metrics {
				_ = testSvc.UpdateMetricByStruct(ctx, m)
			}

			for b.Loop() {
				_, _ = testSvc.GetAllMetrics(ctx)
			}
		})
	}
}

func BenchmarkService_UpdateMetricsBatchByStruct(b *testing.B) {
	storage := repository.NewMemStorage(zaptest.NewLogger(b).Sugar())
	svc := service.NewMetricsService(storage)
	ctx := context.Background()

	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("batch_%d", size), func(b *testing.B) {
			metrics := GenerateMetrics(size)

			for b.Loop() {
				_ = svc.UpdateMetricsBatchByStruct(ctx, metrics)
			}
		})
	}
}

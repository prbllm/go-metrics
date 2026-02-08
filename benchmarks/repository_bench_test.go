package benchmarks

import (
	"context"
	"fmt"
	"testing"

	"github.com/prbllm/go-metrics/internal/model"
	"github.com/prbllm/go-metrics/internal/repository"
	"go.uber.org/zap/zaptest"
)

func BenchmarkMemStorage_UpdateMetric(b *testing.B) {
	storage := repository.NewMemStorage(zaptest.NewLogger(b).Sugar())
	ctx := context.Background()

	value := 123.45
	metric := &model.Metrics{
		MType: model.Gauge,
		ID:    "test_metric",
		Value: &value,
	}

	for b.Loop() {
		_ = storage.UpdateMetric(ctx, metric)
	}
}

func BenchmarkMemStorage_UpdateMetric_Counter(b *testing.B) {
	storage := repository.NewMemStorage(zaptest.NewLogger(b).Sugar())
	ctx := context.Background()

	delta := int64(10)
	metric := &model.Metrics{
		MType: model.Counter,
		ID:    "test_counter",
		Delta: &delta,
	}

	for b.Loop() {
		_ = storage.UpdateMetric(ctx, metric)
	}
}

func BenchmarkMemStorage_GetMetric(b *testing.B) {
	storage := repository.NewMemStorage(zaptest.NewLogger(b).Sugar())
	ctx := context.Background()

	metrics := GenerateMetrics(1000)
	for _, m := range metrics {
		_ = storage.UpdateMetric(ctx, m)
	}

	lookupMetric := &model.Metrics{
		MType: model.Gauge,
		ID:    "metric_A0",
	}

	for b.Loop() {
		_, _ = storage.GetMetric(ctx, lookupMetric)
	}
}

func BenchmarkMemStorage_GetAllMetrics(b *testing.B) {
	ctx := context.Background()

	sizes := []int{10, 100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			testStorage := repository.NewMemStorage(zaptest.NewLogger(b).Sugar())
			metrics := GenerateMetrics(size)
			for _, m := range metrics {
				_ = testStorage.UpdateMetric(ctx, m)
			}

			for b.Loop() {
				_ = testStorage.GetAllMetrics(ctx)
			}
		})
	}
}

func BenchmarkMemStorage_UpdateMetricsBatch(b *testing.B) {
	storage := repository.NewMemStorage(zaptest.NewLogger(b).Sugar())
	ctx := context.Background()

	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("batch_%d", size), func(b *testing.B) {
			metrics := GenerateMetrics(size)

			for b.Loop() {
				_ = storage.UpdateMetricsBatch(ctx, metrics)
			}
		})
	}
}

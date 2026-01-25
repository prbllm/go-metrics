package benchmarks

import (
	"fmt"
	"testing"

	"github.com/prbllm/go-metrics/internal/agent"
	"github.com/prbllm/go-metrics/internal/logger"
	"github.com/prbllm/go-metrics/internal/model"
)

func BenchmarkCollector_CollectRuntimeMetrics(b *testing.B) {
	logger, _ := logger.NewZapLogger()
	collector := agent.NewRuntimeMetricsCollector(logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = collector.CollectRuntimeMetrics()
	}
}

func BenchmarkCollector_CollectGopsutilMetrics(b *testing.B) {
	logger, _ := logger.NewZapLogger()
	collector := agent.NewRuntimeMetricsCollector(logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = collector.CollectGopsutilMetrics()
	}
}

func BenchmarkAgent_CombineMetrics(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			runtimeMetrics := GenerateMetricsSlice(size)
			gopsutilMetrics := GenerateMetricsSlice(size / 2)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = model.CombineMetrics(runtimeMetrics, gopsutilMetrics)
			}
		})
	}
}

package benchmarks

import (
	"math/rand/v2"

	"github.com/prbllm/go-metrics/internal/model"
)

func GenerateMetrics(count int) []*model.Metrics {
	metrics := make([]*model.Metrics, 0, count)

	for i := 0; i < count; i++ {
		if i%2 == 0 {
			value := rand.Float64() * 1000
			metrics = append(metrics, &model.Metrics{
				ID:    generateMetricName(i),
				MType: model.Gauge,
				Value: &value,
			})
		} else {
			delta := rand.Int64N(100)
			metrics = append(metrics, &model.Metrics{
				ID:    generateMetricName(i),
				MType: model.Counter,
				Delta: &delta,
			})
		}
	}

	return metrics
}

func GenerateMetricsSlice(count int) []model.Metrics {
	metrics := make([]model.Metrics, 0, count)

	for i := 0; i < count; i++ {
		if i%2 == 0 {
			value := rand.Float64() * 1000
			metrics = append(metrics, model.Metrics{
				ID:    generateMetricName(i),
				MType: model.Gauge,
				Value: &value,
			})
		} else {
			delta := rand.Int64N(100)
			metrics = append(metrics, model.Metrics{
				ID:    generateMetricName(i),
				MType: model.Counter,
				Delta: &delta,
			})
		}
	}

	return metrics
}

func generateMetricName(index int) string {
	return "metric_" + string(rune('A'+(index%26))) + string(rune('0'+(index%10)))
}

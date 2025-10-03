package repository

import (
	"context"
	"fmt"

	"github.com/prbllm/go-metrics/internal/model"
)

type MemStorage struct {
	metrics map[string]*model.Metrics
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		metrics: make(map[string]*model.Metrics),
	}
}

func (m *MemStorage) generateKey(metricType, name string) string {
	return fmt.Sprintf("%s:%s", metricType, name)
}

func (m *MemStorage) UpdateMetric(ctx context.Context, metric *model.Metrics) error {
	key := m.generateKey(metric.MType, metric.ID)

	if metric.MType == model.Counter {
		if existing, exists := m.metrics[key]; exists && existing.Delta != nil {
			newDelta := *existing.Delta + *metric.Delta
			metric.Delta = &newDelta
		}
	}
	m.metrics[key] = metric
	return nil
}

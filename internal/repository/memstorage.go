package repository

import (
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

func (m *MemStorage) UpdateMetric(metric *model.Metrics) error {
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

func (m *MemStorage) GetMetric(metric *model.Metrics) (*model.Metrics, error) {
	key := m.generateKey(metric.MType, metric.ID)
	metric, ok := m.metrics[key]
	if !ok {
		return nil, fmt.Errorf("metric %s not found", key)
	}
	return metric, nil
}

package repository

import (
	"fmt"

	"github.com/prbllm/go-metrics/internal/config"
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
	config.GetLogger().Debugf("Updating metric: %s", metric.String())
	m.metrics[key] = metric
	return nil
}

func (m *MemStorage) GetMetric(metric *model.Metrics) (*model.Metrics, error) {
	if metric == nil {
		return nil, fmt.Errorf("metric is nil")
	}

	key := m.generateKey(metric.MType, metric.ID)
	val, ok := m.metrics[key]
	if !ok {
		return nil, fmt.Errorf("metric %s not found", key)
	}
	config.GetLogger().Debugf("Getting metric: %s", val.String())
	return val, nil
}

func (m *MemStorage) GetAllMetrics() []*model.Metrics {
	metrics := make([]*model.Metrics, 0, len(m.metrics))
	for _, metric := range m.metrics {
		metrics = append(metrics, metric)
	}
	config.GetLogger().Debugf("Getting all metrics (%s)...", len(metrics))
	return metrics
}

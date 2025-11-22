package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/prbllm/go-metrics/internal/logger"
	"github.com/prbllm/go-metrics/internal/model"
)

type MemStorage struct {
	metrics map[string]*model.Metrics
	mu      sync.RWMutex
	logger  logger.Logger
}

func NewMemStorage(logger logger.Logger) *MemStorage {
	return &MemStorage{
		metrics: make(map[string]*model.Metrics),
		logger:  logger,
	}
}

func (m *MemStorage) generateKey(metricType, name string) string {
	return fmt.Sprintf("%s:%s", metricType, name)
}

func (m *MemStorage) UpdateMetric(ctx context.Context, metric *model.Metrics) error {
	key := m.generateKey(metric.MType, metric.ID)

	m.mu.Lock()
	defer m.mu.Unlock()

	if metric.MType == model.Counter {
		if existing, exists := m.metrics[key]; exists && existing.Delta != nil {
			newDelta := *existing.Delta + *metric.Delta
			metric.Delta = &newDelta
		}
	}
	m.logger.Debugf("Updating metric: %s", metric.String())
	m.metrics[key] = metric
	return nil
}

func (m *MemStorage) UpdateMetricsBatch(ctx context.Context, metrics []*model.Metrics) error {
	if metrics == nil {
		return fmt.Errorf("metrics are nil")
	}

	for _, metric := range metrics {
		if err := m.UpdateMetric(ctx, metric); err != nil {
			return fmt.Errorf("failed to update metric %s: %w", metric.String(), err)
		}
	}

	m.logger.Debugf("Updated batch of %d metrics", len(metrics))
	return nil
}

func (m *MemStorage) GetMetric(ctx context.Context, metric *model.Metrics) (*model.Metrics, error) {
	if metric == nil {
		return nil, fmt.Errorf("metric is nil")
	}

	key := m.generateKey(metric.MType, metric.ID)
	m.mu.RLock()
	val, ok := m.metrics[key]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("metric %s not found", key)
	}
	m.logger.Debugf("Getting metric: %s", val.String())
	return val, nil
}

func (m *MemStorage) GetAllMetrics(ctx context.Context) []*model.Metrics {
	m.mu.RLock()
	metrics := make([]*model.Metrics, 0, len(m.metrics))
	for _, metric := range m.metrics {
		metrics = append(metrics, metric)
	}
	m.mu.RUnlock()
	m.logger.Debugf("Getting all metrics (%d)...", len(metrics))
	return metrics
}

func (m *MemStorage) Ping(ctx context.Context) error {
	return fmt.Errorf("not supported")
}

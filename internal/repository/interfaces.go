// Package repository предоставляет интерфейсы и реализации хранилищ метрик.
package repository

//go:generate mockgen -source=interfaces.go -destination=../mocks/mock_repository.go -package=mocks

import (
	"context"

	"github.com/prbllm/go-metrics/internal/model"
)

// MetricsRepository определяет интерфейс для работы с хранилищем метрик.
type MetricsRepository interface {
	// UpdateMetric обновляет метрику в хранилище.
	UpdateMetric(ctx context.Context, metric *model.Metrics) error

	// UpdateMetricsBatch обновляет пакет метрик в хранилище.
	UpdateMetricsBatch(ctx context.Context, metrics []*model.Metrics) error

	// GetMetric возвращает метрику из хранилища.
	GetMetric(ctx context.Context, metric *model.Metrics) (*model.Metrics, error)

	// GetAllMetrics возвращает все метрики из хранилища.
	GetAllMetrics(ctx context.Context) []model.Metrics

	// Ping проверяет доступность хранилища.
	Ping(ctx context.Context) error
}

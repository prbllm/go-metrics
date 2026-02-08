// Package service предоставляет интерфейсы и реализацию бизнес-логики работы с метриками.
package service

//go:generate mockgen -source=interfaces.go -destination=../mocks/mock_service.go -package=mocks

import (
	"context"

	"github.com/prbllm/go-metrics/internal/model"
)

// Service определяет интерфейс для работы с метриками.
type Service interface {
	// UpdateMetric обновляет метрику по типу, имени и значению в виде строки.
	UpdateMetric(ctx context.Context, metricType, metricName, metricValue string) error

	// UpdateMetricByStruct обновляет метрику по структуре Metrics.
	UpdateMetricByStruct(ctx context.Context, metric *model.Metrics) error

	// UpdateMetricsBatchByStruct обновляет пакет метрик.
	UpdateMetricsBatchByStruct(ctx context.Context, metrics []*model.Metrics) error

	// GetMetric возвращает метрику по типу и имени.
	GetMetric(ctx context.Context, metricType, metricName string) (*model.Metrics, error)

	// GetAllMetrics возвращает все метрики.
	GetAllMetrics(ctx context.Context) ([]model.Metrics, error)

	// Ping проверяет доступность хранилища метрик.
	Ping(ctx context.Context) error
}

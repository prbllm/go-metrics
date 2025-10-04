package repository

import (
	"github.com/prbllm/go-metrics/internal/model"
)

type MetricsRepository interface {
	UpdateMetric(metric *model.Metrics) error
	GetMetric(metric *model.Metrics) (*model.Metrics, error)
}

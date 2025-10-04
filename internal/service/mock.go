package service

import "github.com/prbllm/go-metrics/internal/model"

// MockMetricsService for testing
type MockMetricsService struct {
	Error error
}

func (m *MockMetricsService) UpdateMetric(metricType, metricName, metricValue string) error {
	return m.Error
}

func (m *MockMetricsService) GetMetric(metricType, metricName string) (*model.Metrics, error) {
	return nil, m.Error
}

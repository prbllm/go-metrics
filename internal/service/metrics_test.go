package service

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"testing"

	"github.com/prbllm/go-metrics/internal/mocks"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/prbllm/go-metrics/internal/repository"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap/zaptest"
)

func TestMetricsService_UpdateMetric(t *testing.T) {
	storage := repository.NewMemStorage(zaptest.NewLogger(t).Sugar())
	service := NewMetricsService(storage)

	tests := []struct {
		name        string
		metricType  string
		metricName  string
		metricValue string
	}{
		{
			name:        "counter",
			metricType:  model.Counter,
			metricName:  "test_counter",
			metricValue: "42",
		},
		{
			name:        "gauge",
			metricType:  model.Gauge,
			metricName:  "test_gauge",
			metricValue: "3.14",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := service.UpdateMetric(context.Background(), test.metricType, test.metricName, test.metricValue)
			require.NoError(t, err, "Update failed")

			metric, err := service.GetMetric(context.Background(), test.metricType, test.metricName)
			require.NoError(t, err, "Get failed")
			if test.metricType == model.Gauge {
				expectedValue, err := strconv.ParseFloat(test.metricValue, 64)
				require.NoError(t, err)
				require.Equal(t, expectedValue, *metric.Value, "Value is not equal to expected")
			} else {
				expectedValue, err := strconv.ParseInt(test.metricValue, 10, 64)
				require.NoError(t, err)
				require.Equal(t, expectedValue, *metric.Delta, "Value is not equal to expected")
			}
		})
	}
}

func TestMetricsService_Ping(t *testing.T) {
	tests := []struct {
		name        string
		repository  repository.MetricsRepository
		expectError bool
	}{
		{
			name:        "ping with repository",
			repository:  repository.NewMemStorage(zaptest.NewLogger(t).Sugar()),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewMetricsService(tt.repository)

			err := svc.Ping(context.Background())
			if tt.expectError {
				require.Error(t, err, "Expected error")
			} else {
				require.NoError(t, err, "Ping should succeed")
			}
		})
	}
}

func TestMetricsService_CounterAccumulation(t *testing.T) {
	storage := repository.NewMemStorage(zaptest.NewLogger(t).Sugar())
	service := NewMetricsService(storage)

	const metricName = "test_counter"
	const metricValue = "5"
	const expectedDelta = int64(10)
	err := service.UpdateMetric(context.Background(), model.Counter, metricName, metricValue)
	require.NoError(t, err, "First update failed")

	err = service.UpdateMetric(context.Background(), model.Counter, metricName, metricValue)
	require.NoError(t, err, "Second update failed")

	metric, err := service.GetMetric(context.Background(), model.Counter, metricName)
	require.NoError(t, err, "Get failed")
	require.Equal(t, expectedDelta, *metric.Delta, "Delta is not equal to expected")
}

func TestMetricsService_GaugeReplacement(t *testing.T) {
	storage := repository.NewMemStorage(zaptest.NewLogger(t).Sugar())
	service := NewMetricsService(storage)

	const metricName = "test_gauge"
	const metricValue = "10.5"
	const newMetricValue = "20.7"

	err := service.UpdateMetric(context.Background(), model.Gauge, metricName, metricValue)
	require.NoError(t, err, "First update failed")

	err = service.UpdateMetric(context.Background(), model.Gauge, metricName, newMetricValue)
	require.NoError(t, err, "Second update failed")

	metric, err := service.GetMetric(context.Background(), model.Gauge, metricName)
	require.NoError(t, err, "Get failed")
	expectedValue, err := strconv.ParseFloat(newMetricValue, 64)
	require.NoError(t, err)
	require.Equal(t, expectedValue, *metric.Value, "Value is not equal to expected")
}

func TestMetricsService_GetAllMetrics(t *testing.T) {
	storage := repository.NewMemStorage(zaptest.NewLogger(t).Sugar())
	service := NewMetricsService(storage)

	expectedValue := float64(10.5)
	expectedDelta := int64(10)
	expectedMetrics := []*model.Metrics{
		{ID: "test_gauge", MType: model.Gauge, Value: &expectedValue},
		{ID: "test_counter", MType: model.Counter, Delta: &expectedDelta},
	}
	service.UpdateMetric(context.Background(), model.Gauge, expectedMetrics[0].ID, strconv.FormatFloat(expectedValue, 'f', -1, 64))
	service.UpdateMetric(context.Background(), model.Counter, expectedMetrics[1].ID, strconv.FormatInt(expectedDelta, 10))

	metrics, err := service.GetAllMetrics(context.Background())
	require.NoError(t, err, "Get all metrics failed")
	require.Equal(t, len(expectedMetrics), len(metrics), "Metrics count is not equal to expected")

	sort.Slice(expectedMetrics, func(i, j int) bool {
		return expectedMetrics[i].ID < expectedMetrics[j].ID
	})
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].ID < metrics[j].ID
	})
	for i := range expectedMetrics {
		require.Equal(t, expectedMetrics[i].MType, metrics[i].MType, "Metric type is not equal to expected")
		require.Equal(t, expectedMetrics[i].ID, metrics[i].ID, "Metric ID is not equal to expected")
		require.Equal(t, expectedMetrics[i].Delta, metrics[i].Delta, "Metric delta is not equal to expected")
		require.Equal(t, expectedMetrics[i].Value, metrics[i].Value, "Metric value is not equal to expected")
	}
}

func TestMetricsService_GetMetric(t *testing.T) {
	storage := repository.NewMemStorage(zaptest.NewLogger(t).Sugar())
	service := NewMetricsService(storage)
	expectedValue := float64(10.5)

	expectedMetric := &model.Metrics{MType: model.Gauge, ID: "test_gauge", Value: &expectedValue}
	service.UpdateMetric(context.Background(), model.Gauge, expectedMetric.ID, strconv.FormatFloat(expectedValue, 'f', -1, 64))
	metric, err := service.GetMetric(context.Background(), model.Gauge, expectedMetric.ID)
	require.NoError(t, err, "Get metric failed")
	require.Equal(t, metric, expectedMetric, "Metric is not equal to expected")

	expectedDelta := int64(10)
	expectedMetric = &model.Metrics{MType: model.Counter, ID: "test_counter", Delta: &expectedDelta}
	service.UpdateMetric(context.Background(), model.Counter, expectedMetric.ID, strconv.FormatInt(expectedDelta, 10))
	metric, err = service.GetMetric(context.Background(), model.Counter, expectedMetric.ID)
	require.NoError(t, err, "Get metric failed")
	require.Equal(t, metric, expectedMetric, "Metric is not equal to expected")
}

func TestMetricsService_UpdateMetric_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMetricsRepository(ctrl)
	service := NewMetricsService(mockRepo)

	expectedError := errors.New("repository error")
	mockRepo.EXPECT().UpdateMetric(gomock.Any(), gomock.Any()).Return(expectedError).Times(1)

	err := service.UpdateMetric(context.Background(), model.Counter, "test_counter", "42")
	require.Error(t, err, "Expected error from repository")
	require.Equal(t, expectedError, err, "Error should be propagated from repository")
}

func TestMetricsService_GetMetric_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMetricsRepository(ctrl)
	service := NewMetricsService(mockRepo)

	expectedError := errors.New("repository error")
	mockRepo.EXPECT().GetMetric(gomock.Any(), gomock.Any()).Return(nil, expectedError).Times(1)

	metric, err := service.GetMetric(context.Background(), model.Counter, "test_counter")
	require.Error(t, err, "Expected error from repository")
	require.Nil(t, metric, "Metric should be nil on error")
	require.Equal(t, expectedError, err, "Error should be propagated from repository")
}

func TestMetricsService_GetAllMetrics_RepositoryCall(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMetricsRepository(ctrl)
	service := NewMetricsService(mockRepo)

	expectedMetrics := []*model.Metrics{
		{ID: "test1", MType: model.Gauge, Value: func() *float64 { v := 1.5; return &v }()},
		{ID: "test2", MType: model.Counter, Delta: func() *int64 { v := int64(10); return &v }()},
	}

	mockRepo.EXPECT().GetAllMetrics(gomock.Any()).Return(expectedMetrics).Times(1)

	metrics, err := service.GetAllMetrics(context.Background())
	require.NoError(t, err, "GetAllMetrics should not return error")
	require.Equal(t, expectedMetrics, metrics, "Metrics should match repository response")
}

func TestMetricsService_UpdateMetricByStruct_RepositoryCall(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMetricsRepository(ctrl)
	service := NewMetricsService(mockRepo)

	metric := &model.Metrics{
		ID:    "test_metric",
		MType: model.Gauge,
		Value: func() *float64 { v := 3.14; return &v }(),
	}

	mockRepo.EXPECT().UpdateMetric(gomock.Any(), metric).Return(nil).Times(1)

	err := service.UpdateMetricByStruct(context.Background(), metric)
	require.NoError(t, err, "UpdateMetricByStruct should succeed")
}

func TestMetricsService_UpdateMetricByStruct_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMetricsRepository(ctrl)
	service := NewMetricsService(mockRepo)

	metric := &model.Metrics{
		ID:    "test_metric",
		MType: model.Counter,
		Delta: func() *int64 { v := int64(42); return &v }(),
	}

	expectedError := errors.New("repository update failed")
	mockRepo.EXPECT().UpdateMetric(gomock.Any(), metric).Return(expectedError).Times(1)

	err := service.UpdateMetricByStruct(context.Background(), metric)
	require.Error(t, err, "Expected error from repository")
	require.Equal(t, expectedError, err, "Error should be propagated from repository")
}

func TestMetricsService_Ping_RepositoryCall(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMetricsRepository(ctrl)
	service := NewMetricsService(mockRepo)

	mockRepo.EXPECT().Ping(gomock.Any()).Return(nil).Times(1)

	err := service.Ping(context.Background())
	require.NoError(t, err, "Ping should succeed")
}

func TestMetricsService_Ping_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMetricsRepository(ctrl)
	service := NewMetricsService(mockRepo)

	expectedError := errors.New("ping failed")
	mockRepo.EXPECT().Ping(gomock.Any()).Return(expectedError).Times(1)

	err := service.Ping(context.Background())
	require.Error(t, err, "Expected error from repository")
	require.Equal(t, expectedError, err, "Error should be propagated from repository")
}

func TestMetricsService_UpdateMetric_Counter_CallsRepositoryOnce(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMetricsRepository(ctrl)
	service := NewMetricsService(mockRepo)

	mockRepo.EXPECT().UpdateMetric(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, metric *model.Metrics) error {
		require.Equal(t, model.Counter, metric.MType, "Metric type should be counter")
		require.NotNil(t, metric.Delta, "Delta should not be nil")
		require.Equal(t, int64(42), *metric.Delta, "Delta should be 42")
		return nil
	}).Times(1)

	err := service.UpdateMetric(context.Background(), model.Counter, "test_counter", "42")
	require.NoError(t, err, "UpdateMetric should succeed")
}

func TestMetricsService_UpdateMetric_Gauge_CallsRepositoryOnce(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMetricsRepository(ctrl)
	service := NewMetricsService(mockRepo)

	mockRepo.EXPECT().UpdateMetric(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, metric *model.Metrics) error {
		require.Equal(t, model.Gauge, metric.MType, "Metric type should be gauge")
		require.NotNil(t, metric.Value, "Value should not be nil")
		require.Equal(t, 3.14, *metric.Value, "Value should be 3.14")
		return nil
	}).Times(1)

	err := service.UpdateMetric(context.Background(), model.Gauge, "test_gauge", "3.14")
	require.NoError(t, err, "UpdateMetric should succeed")
}

func TestMetricsService_UpdateMetricsBatchByStruct(t *testing.T) {
	storage := repository.NewMemStorage(zaptest.NewLogger(t).Sugar())
	service := NewMetricsService(storage)

	tests := []struct {
		name          string
		metrics       []*model.Metrics
		expectError   bool
		expectedError string
	}{
		{
			name: "valid batch with counter and gauge",
			metrics: []*model.Metrics{
				{ID: "counter1", MType: model.Counter, Delta: func() *int64 { v := int64(10); return &v }()},
				{ID: "gauge1", MType: model.Gauge, Value: func() *float64 { v := 3.14; return &v }()},
			},
			expectError: false,
		},
		{
			name:        "nil metrics slice",
			metrics:     nil,
			expectError: true,
		},
		{
			name:        "empty metrics slice",
			metrics:     []*model.Metrics{},
			expectError: false,
		},
		{
			name: "invalid metric - missing delta for counter",
			metrics: []*model.Metrics{
				{ID: "counter1", MType: model.Counter, Delta: nil},
			},
			expectError: true,
		},
		{
			name: "invalid metric - missing value for gauge",
			metrics: []*model.Metrics{
				{ID: "gauge1", MType: model.Gauge, Value: nil},
			},
			expectError: true,
		},
		{
			name: "invalid metric type",
			metrics: []*model.Metrics{
				{ID: "invalid", MType: "invalid_type", Value: func() *float64 { v := 1.0; return &v }()},
			},
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := service.UpdateMetricsBatchByStruct(context.Background(), test.metrics)
			if test.expectError {
				require.Error(t, err, "Expected error")
			} else {
				require.NoError(t, err, "UpdateMetricsBatchByStruct should succeed")
			}
		})
	}
}

func TestMetricsService_UpdateMetricsBatchByStruct_RepositoryCall(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMetricsRepository(ctrl)
	service := NewMetricsService(mockRepo)

	metrics := []*model.Metrics{
		{ID: "counter1", MType: model.Counter, Delta: func() *int64 { v := int64(10); return &v }()},
		{ID: "gauge1", MType: model.Gauge, Value: func() *float64 { v := 3.14; return &v }()},
	}

	mockRepo.EXPECT().UpdateMetricsBatch(gomock.Any(), metrics).Return(nil).Times(1)

	err := service.UpdateMetricsBatchByStruct(context.Background(), metrics)
	require.NoError(t, err, "UpdateMetricsBatchByStruct should succeed")
}

func TestMetricsService_UpdateMetricsBatchByStruct_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMetricsRepository(ctrl)
	service := NewMetricsService(mockRepo)

	metrics := []*model.Metrics{
		{ID: "counter1", MType: model.Counter, Delta: func() *int64 { v := int64(10); return &v }()},
	}

	expectedError := errors.New("repository update failed")
	mockRepo.EXPECT().UpdateMetricsBatch(gomock.Any(), metrics).Return(expectedError).Times(1)

	err := service.UpdateMetricsBatchByStruct(context.Background(), metrics)
	require.Error(t, err, "Expected error from repository")
}

func TestMetricsService_UpdateMetricsBatchByStruct_CounterAccumulation(t *testing.T) {
	storage := repository.NewMemStorage(zaptest.NewLogger(t).Sugar())
	service := NewMetricsService(storage)

	const metricName = "test_counter"
	delta1 := int64(5)
	delta2 := int64(3)
	expectedDelta := int64(8)

	metrics1 := []*model.Metrics{
		{ID: metricName, MType: model.Counter, Delta: &delta1},
	}
	err := service.UpdateMetricsBatchByStruct(context.Background(), metrics1)
	require.NoError(t, err, "First batch update failed")

	metrics2 := []*model.Metrics{
		{ID: metricName, MType: model.Counter, Delta: &delta2},
	}
	err = service.UpdateMetricsBatchByStruct(context.Background(), metrics2)
	require.NoError(t, err, "Second batch update failed")

	metric, err := service.GetMetric(context.Background(), model.Counter, metricName)
	require.NoError(t, err, "Get failed")
	require.Equal(t, expectedDelta, *metric.Delta, "Delta should accumulate")
}

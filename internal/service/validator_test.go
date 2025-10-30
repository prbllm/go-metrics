package service

import (
	"testing"

	"github.com/prbllm/go-metrics/internal/model"
	"github.com/stretchr/testify/require"
)

func TestValidateMetricType(t *testing.T) {
	tests := []struct {
		name        string
		metricType  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid counter type",
			metricType:  model.Counter,
			expectError: false,
		},
		{
			name:        "valid gauge type",
			metricType:  model.Gauge,
			expectError: false,
		},
		{
			name:        "invalid type - empty string",
			metricType:  "",
			expectError: true,
			errorMsg:    "invalid metric type",
		},
		{
			name:        "invalid type - random string",
			metricType:  "invalid",
			expectError: true,
			errorMsg:    "invalid metric type",
		},
		{
			name:        "invalid type - case sensitive",
			metricType:  "COUNTER",
			expectError: true,
			errorMsg:    "invalid metric type",
		},
		{
			name:        "invalid type - partial match",
			metricType:  "counter_extra",
			expectError: true,
			errorMsg:    "invalid metric type",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateMetricType(test.metricType)

			if test.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateMetricValue(t *testing.T) {
	tests := []struct {
		name        string
		metricType  string
		value       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid counter - positive integer",
			metricType:  model.Counter,
			value:       "42",
			expectError: false,
		},
		{
			name:        "valid counter - zero",
			metricType:  model.Counter,
			value:       "0",
			expectError: false,
		},
		{
			name:        "valid counter - negative integer",
			metricType:  model.Counter,
			value:       "-10",
			expectError: false,
		},
		{
			name:        "valid counter - large integer",
			metricType:  model.Counter,
			value:       "9223372036854775807",
			expectError: false,
		},
		{
			name:        "invalid counter - float",
			metricType:  model.Counter,
			value:       "3.14",
			expectError: true,
			errorMsg:    "counter value must be integer",
		},
		{
			name:        "invalid counter - string",
			metricType:  model.Counter,
			value:       "abc",
			expectError: true,
			errorMsg:    "counter value must be integer",
		},
		{
			name:        "invalid counter - empty string",
			metricType:  model.Counter,
			value:       "",
			expectError: true,
			errorMsg:    "counter value must be integer",
		},
		{
			name:        "valid gauge - positive float",
			metricType:  model.Gauge,
			value:       "3.14",
			expectError: false,
		},
		{
			name:        "valid gauge - integer as float",
			metricType:  model.Gauge,
			value:       "42",
			expectError: false,
		},
		{
			name:        "valid gauge - zero",
			metricType:  model.Gauge,
			value:       "0.0",
			expectError: false,
		},
		{
			name:        "valid gauge - negative float",
			metricType:  model.Gauge,
			value:       "-1.5",
			expectError: false,
		},
		{
			name:        "valid gauge - scientific notation",
			metricType:  model.Gauge,
			value:       "1.23e-4",
			expectError: false,
		},
		{
			name:        "invalid gauge - string",
			metricType:  model.Gauge,
			value:       "abc",
			expectError: true,
			errorMsg:    "gauge value must be float",
		},
		{
			name:        "invalid gauge - empty string",
			metricType:  model.Gauge,
			value:       "",
			expectError: true,
			errorMsg:    "gauge value must be float",
		},
		{
			name:        "invalid metric type",
			metricType:  "invalid",
			value:       "42",
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateMetricValue(test.metricType, test.value)

			if test.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateMetric(t *testing.T) {
	tests := []struct {
		name        string
		metric      *model.Metrics
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid counter metric",
			metric: &model.Metrics{
				ID:    "test_counter",
				MType: model.Counter,
				Delta: func() *int64 { v := int64(42); return &v }(),
			},
			expectError: false,
		},
		{
			name: "valid gauge metric",
			metric: &model.Metrics{
				ID:    "test_gauge",
				MType: model.Gauge,
				Value: func() *float64 { v := 3.14; return &v }(),
			},
			expectError: false,
		},
		{
			name: "valid counter with zero delta",
			metric: &model.Metrics{
				ID:    "test_counter",
				MType: model.Counter,
				Delta: func() *int64 { v := int64(0); return &v }(),
			},
			expectError: false,
		},
		{
			name: "valid gauge with zero value",
			metric: &model.Metrics{
				ID:    "test_gauge",
				MType: model.Gauge,
				Value: func() *float64 { v := 0.0; return &v }(),
			},
			expectError: false,
		},
		{
			name: "invalid metric type",
			metric: &model.Metrics{
				ID:    "test_metric",
				MType: "invalid",
				Value: func() *float64 { v := 1.0; return &v }(),
			},
			expectError: true,
			errorMsg:    "invalid metric type",
		},
		{
			name: "empty metric type",
			metric: &model.Metrics{
				ID:    "test_metric",
				MType: "",
				Value: func() *float64 { v := 1.0; return &v }(),
			},
			expectError: true,
			errorMsg:    "invalid metric type",
		},
		{
			name: "counter without delta",
			metric: &model.Metrics{
				ID:    "test_counter",
				MType: model.Counter,
				Delta: nil,
			},
			expectError: true,
			errorMsg:    "test_counter has no delta",
		},
		{
			name: "gauge without value",
			metric: &model.Metrics{
				ID:    "test_gauge",
				MType: model.Gauge,
				Value: nil,
			},
			expectError: true,
			errorMsg:    "test_gauge has no value",
		},
		// Edge cases
		{
			name: "counter with both delta and value (should be valid)",
			metric: &model.Metrics{
				ID:    "test_counter",
				MType: model.Counter,
				Delta: func() *int64 { v := int64(42); return &v }(),
				Value: func() *float64 { v := 3.14; return &v }(),
			},
			expectError: false,
		},
		{
			name: "gauge with both value and delta (should be valid)",
			metric: &model.Metrics{
				ID:    "test_gauge",
				MType: model.Gauge,
				Value: func() *float64 { v := 3.14; return &v }(),
				Delta: func() *int64 { v := int64(42); return &v }(),
			},
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateMetric(test.metric)

			if test.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

package service

import (
	"fmt"
	"strconv"

	"github.com/prbllm/go-metrics/internal/model"
)

func ValidateMetricType(metricType string) error {
	if metricType != model.Counter && metricType != model.Gauge {
		return fmt.Errorf("invalid metric type")
	}
	return nil
}

func ValidateMetricValue(metricType, value string) error {
	switch metricType {
	case model.Counter:
		_, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("%s value must be integer", metricType)
		}
	case model.Gauge:
		_, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("%s value must be float", metricType)
		}
	}
	return nil
}

func ValidateMetric(metric *model.Metrics) error {
	if err := ValidateMetricType(metric.MType); err != nil {
		return err
	}
	switch metric.MType {
	case model.Counter:
		if metric.Delta == nil {
			return fmt.Errorf("%s has no delta", metric.ID)
		}
	case model.Gauge:
		if metric.Value == nil {
			return fmt.Errorf("%s has no value", metric.ID)
		}
	}
	return nil
}

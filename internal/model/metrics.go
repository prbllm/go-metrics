// Package model предоставляет модели данных для метрик.
package model

import (
	"fmt"
	"strings"
)

const (
	// Counter - тип метрики-счетчика (целочисленное значение).
	Counter = "counter"

	// Gauge - тип метрики-измерителя (значение с плавающей точкой).
	Gauge = "gauge"
)

// Metrics представляет метрику системы.
// Delta и Value объявлены через указатели, чтобы отличать значение "0" от не заданного значения.
type Metrics struct {
	ID    string   `json:"id"`              // Идентификатор метрики
	MType string   `json:"type"`            // Тип метрики (counter или gauge)
	Delta *int64   `json:"delta,omitempty"` // Значение для счетчика
	Value *float64 `json:"value,omitempty"` // Значение для измерителя
	Hash  string   `json:"hash,omitempty"`  // Хеш для проверки целостности
}

// String возвращает строковое представление метрики.
func (m *Metrics) String() string {
	var b strings.Builder
	b.WriteString("Metric{ID: ")
	b.WriteString(m.ID)
	b.WriteString(", MType: ")
	b.WriteString(m.MType)
	b.WriteString(", ")

	if m.Delta != nil {
		b.WriteString("Delta: ")
		b.WriteString(fmt.Sprintf("%d", *m.Delta))
		b.WriteString(", ")
	} else {
		b.WriteString("Delta: nil, ")
	}

	if m.Value != nil {
		b.WriteString("Value: ")
		b.WriteString(fmt.Sprintf("%f", *m.Value))
		b.WriteString(", ")
	} else {
		b.WriteString("Value: nil, ")
	}

	if m.Hash != "" {
		b.WriteString("Hash: ")
		b.WriteString(m.Hash)
	} else {
		b.WriteString("Hash: nil")
	}

	b.WriteString("}")
	return b.String()
}

// CombineMetrics объединяет два списка метрик, суммируя значения счетчиков и заменяя значения измерителей.
func CombineMetrics(metrics []Metrics, metricsToAdd []Metrics) []Metrics {
	metricsMap := make(map[string]*Metrics, len(metrics))

	for i := range metrics {
		metricsMap[metrics[i].ID] = &metrics[i]
	}

	for _, metric := range metricsToAdd {
		existing, found := metricsMap[metric.ID]
		if found {
			switch metric.MType {
			case Counter:
				if existing.Delta != nil && metric.Delta != nil {
					newDelta := *existing.Delta + *metric.Delta
					existing.Delta = &newDelta
				} else if metric.Delta != nil {
					existing.Delta = metric.Delta
				}
			case Gauge:
				if metric.Value != nil {
					if existing.Value == nil {
						existing.Value = new(float64)
					}
					*existing.Value = *metric.Value
				}
			}
		} else {
			metrics = append(metrics, metric)
			metricsMap[metric.ID] = &metrics[len(metrics)-1]
		}
	}

	return metrics
}

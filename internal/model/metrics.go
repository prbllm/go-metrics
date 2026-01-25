package model

import (
	"fmt"
	"strings"
)

const (
	Counter = "counter"
	Gauge   = "gauge"
)

// NOTE: Не усложняем пример, вводя иерархическую вложенность структур.
// Органичиваясь плоской моделью.
// Delta и Value объявлены через указатели,
// что бы отличать значение "0", от не заданного значения
// и соответственно не кодировать в структуру.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

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

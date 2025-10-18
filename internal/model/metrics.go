package model

import "fmt"

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
	metricString := fmt.Sprintf("Metric{ID: %s, MType: %s, ", m.ID, m.MType)
	if m.Delta != nil {
		metricString += fmt.Sprintf("Delta: %d, ", *m.Delta)
	} else {
		metricString += "Delta: nil, "
	}
	if m.Value != nil {
		metricString += fmt.Sprintf("Value: %f, ", *m.Value)
	} else {
		metricString += "Value: nil, "
	}
	if m.Hash != "" {
		metricString += fmt.Sprintf("Hash: %s", m.Hash)
	} else {
		metricString += "Hash: nil"
	}
	metricString += "}"
	return metricString
}

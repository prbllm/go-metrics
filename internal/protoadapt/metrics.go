package protoadapt

import (
	"github.com/prbllm/go-metrics/internal/model"
	metricsv1 "github.com/prbllm/go-metrics/internal/proto/metrics/v1"
)

// ModelToUpdateMetricsRequest конвертирует []model.Metrics в *metricsv1.UpdateMetricsRequest.
func ModelToUpdateMetricsRequest(metrics []model.Metrics) *metricsv1.UpdateMetricsRequest {
	protoMetrics := make([]*metricsv1.Metric, 0, len(metrics))
	for i := range metrics {
		m := &metrics[i]
		pm := &metricsv1.Metric{
			Id: m.ID,
		}
		if m.MType == model.Counter {
			pm.Type = metricsv1.Metric_COUNTER
			if m.Delta != nil {
				pm.Delta = *m.Delta
			}
		} else {
			pm.Type = metricsv1.Metric_GAUGE
			if m.Value != nil {
				pm.Value = *m.Value
			}
		}
		protoMetrics = append(protoMetrics, pm)
	}
	return &metricsv1.UpdateMetricsRequest{Metrics: protoMetrics}
}

// ProtoToModel конвертирует срез proto-метрик в []*model.Metrics.
func ProtoToModel(protoMetrics []*metricsv1.Metric) []*model.Metrics {
	out := make([]*model.Metrics, 0, len(protoMetrics))
	for _, m := range protoMetrics {
		if m == nil {
			continue
		}
		mm := &model.Metrics{
			ID:    m.GetId(),
			MType: protoMTypeToModel(m.GetType()),
		}
		switch m.GetType() {
		case metricsv1.Metric_COUNTER:
			d := m.GetDelta()
			mm.Delta = &d
		case metricsv1.Metric_GAUGE:
			v := m.GetValue()
			mm.Value = &v
		}
		out = append(out, mm)
	}
	return out
}

func protoMTypeToModel(t metricsv1.Metric_MType) string {
	switch t {
	case metricsv1.Metric_COUNTER:
		return model.Counter
	default:
		return model.Gauge
	}
}

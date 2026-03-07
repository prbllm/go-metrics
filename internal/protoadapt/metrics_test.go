package protoadapt

import (
	"testing"

	"github.com/prbllm/go-metrics/internal/model"
	"github.com/stretchr/testify/require"
)

func TestModelToUpdateMetricsRequest_ProtoToModel_Roundtrip(t *testing.T) {
	delta := int64(10)
	value := 2.5
	metrics := []model.Metrics{
		{ID: "c1", MType: model.Counter, Delta: &delta},
		{ID: "g1", MType: model.Gauge, Value: &value},
	}
	req := ModelToUpdateMetricsRequest(metrics)
	require.Len(t, req.GetMetrics(), 2)
	back := ProtoToModel(req.GetMetrics())
	require.Len(t, back, 2)
	require.Equal(t, "c1", back[0].ID)
	require.Equal(t, model.Counter, back[0].MType)
	require.NotNil(t, back[0].Delta)
	require.Equal(t, int64(10), *back[0].Delta)
	require.Equal(t, "g1", back[1].ID)
	require.Equal(t, model.Gauge, back[1].MType)
	require.NotNil(t, back[1].Value)
	require.Equal(t, 2.5, *back[1].Value)
}

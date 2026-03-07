package grpc

import (
	"context"

	metricsv1 "github.com/prbllm/go-metrics/internal/proto/metrics/v1"
	"github.com/prbllm/go-metrics/internal/protoadapt"
	"github.com/prbllm/go-metrics/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MetricsServer реализует metricsv1.MetricsServer поверх service.MetricsService.
type MetricsServer struct {
	metricsv1.UnimplementedMetricsServer
	svc *service.MetricsService
}

// NewMetricsServer создаёт gRPC-сервер метрик.
func NewMetricsServer(svc *service.MetricsService) *MetricsServer {
	return &MetricsServer{svc: svc}
}

// UpdateMetrics принимает батч метрик и сохраняет их через MetricsService.
func (s *MetricsServer) UpdateMetrics(ctx context.Context, req *metricsv1.UpdateMetricsRequest) (*metricsv1.UpdateMetricsResponse, error) {
	if req == nil || len(req.GetMetrics()) == 0 {
		return &metricsv1.UpdateMetricsResponse{}, nil
	}
	metrics := protoadapt.ProtoToModel(req.GetMetrics())
	if err := s.svc.UpdateMetricsBatchByStruct(ctx, metrics); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &metricsv1.UpdateMetricsResponse{}, nil
}

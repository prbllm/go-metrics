package grpcserver

import (
	"context"

	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/logger"
	"github.com/prbllm/go-metrics/internal/network"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TrustedSubnetUnaryInterceptor возвращает UnaryInterceptor, проверяющий,
// что IP из метаданных (x-real-ip) входит в доверенную подсеть (trusted_subnet).
// Если trusted_subnet не задан, запросы проходят без проверки.
func TrustedSubnetUnaryInterceptor(logger logger.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		invoker grpc.UnaryHandler,
	) (interface{}, error) {
		cfg := config.GetConfig()
		ipStr := ""
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			vals := md.Get(config.GRPCRealIPMetadataKey)
			if len(vals) > 0 {
				ipStr = vals[0]
			}
		}
		allowed, err := network.IsIPInTrustedSubnet(ipStr, cfg.TrustedSubnet)
		if err != nil {
			logger.Errorf("invalid trusted subnet configuration %q: %v", cfg.TrustedSubnet, err)
			return nil, status.Error(codes.Internal, "invalid trusted subnet configuration")
		}
		if !allowed {
			return nil, status.Error(codes.PermissionDenied, "IP not in trusted subnet")
		}
		return invoker(ctx, req)
	}
}

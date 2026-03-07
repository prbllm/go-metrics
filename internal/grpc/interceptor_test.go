package grpc

import (
	"context"
	"testing"

	"github.com/prbllm/go-metrics/internal/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestTrustedSubnetUnaryInterceptor(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	interceptor := TrustedSubnetUnaryInterceptor(logger)

	invoker := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}

	t.Run("no trusted subnet - passes without metadata", func(t *testing.T) {
		base := config.GetConfig()
		cfg := *base
		cfg.TrustedSubnet = ""
		config.SetConfig(&cfg, logger)

		ctx := context.Background()
		resp, err := interceptor(ctx, nil, nil, invoker)
		require.NoError(t, err)
		require.Equal(t, "ok", resp)
	})

	t.Run("trusted subnet - IP in subnet", func(t *testing.T) {
		base := config.GetConfig()
		cfg := *base
		cfg.TrustedSubnet = "127.0.0.0/8"
		config.SetConfig(&cfg, logger)

		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(config.GRPCRealIPMetadataKey, "127.0.0.1"))
		resp, err := interceptor(ctx, nil, nil, invoker)
		require.NoError(t, err)
		require.Equal(t, "ok", resp)
	})

	t.Run("trusted subnet - IP outside subnet returns PermissionDenied", func(t *testing.T) {
		base := config.GetConfig()
		cfg := *base
		cfg.TrustedSubnet = "127.0.0.0/8"
		config.SetConfig(&cfg, logger)

		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(config.GRPCRealIPMetadataKey, "10.0.0.1"))
		resp, err := interceptor(ctx, nil, nil, invoker)
		require.Error(t, err)
		require.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.PermissionDenied, st.Code())
	})

	t.Run("trusted subnet - no metadata returns PermissionDenied", func(t *testing.T) {
		base := config.GetConfig()
		cfg := *base
		cfg.TrustedSubnet = "127.0.0.0/8"
		config.SetConfig(&cfg, logger)

		ctx := context.Background()
		resp, err := interceptor(ctx, nil, nil, invoker)
		require.Error(t, err)
		require.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.PermissionDenied, st.Code())
	})
}

package audit

//go:generate mockgen -source=interfaces.go -destination=../mocks/mock_audit.go -package=mocks

import (
	"context"
)

type AuditEvent struct {
	Timestamp  int64    `json:"ts"`
	MetricsIDs []string `json:"metrics"`
	IPAddress  string   `json:"ip_address"`
}

type MetricsObserver interface {
	Process(ctx context.Context, event AuditEvent)
	Close()
}

type clientIPKey struct{}

func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPKey{}, ip)
}

func GetClientIP(ctx context.Context) string {
	if ip, ok := ctx.Value(clientIPKey{}).(string); ok {
		return ip
	}
	return ""
}

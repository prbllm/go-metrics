// Package audit предоставляет интерфейсы и реализации для аудита метрик.
package audit

//go:generate mockgen -source=interfaces.go -destination=../mocks/mock_audit.go -package=mocks

import (
	"context"
)

// AuditEvent представляет событие аудита с информацией о метриках и IP-адресе клиента.
type AuditEvent struct {
	Timestamp  int64    `json:"ts"`          // Временная метка события
	MetricsIDs []string `json:"metrics"`     // Список идентификаторов метрик
	IPAddress  string   `json:"ip_address"`  // IP-адрес клиента
}

// MetricsObserver определяет интерфейс для наблюдателей событий аудита.
type MetricsObserver interface {
	// Process обрабатывает событие аудита.
	Process(ctx context.Context, event AuditEvent)

	// Close закрывает ресурсы наблюдателя.
	Close()
}

type clientIPKey struct{}

// WithClientIP добавляет IP-адрес клиента в контекст.
func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPKey{}, ip)
}

// GetClientIP извлекает IP-адрес клиента из контекста.
func GetClientIP(ctx context.Context) string {
	if ip, ok := ctx.Value(clientIPKey{}).(string); ok {
		return ip
	}
	return ""
}

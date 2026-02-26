package testutil

import (
	"encoding/json"
	"testing"

	"github.com/prbllm/go-metrics/internal/hash"
)

// MustHashFromJSON вычисляет хэш по JSON-представлению значения v.
// В случае ошибки маршалинга завершает тест с фатальной ошибкой.
func MustHashFromJSON(tb testing.TB, key string, v any) string {
	tb.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		tb.Fatalf("failed to marshal value to JSON for hash: %v", err)
	}

	return hash.ComputeHash(key, data)
}


package versions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func Test_defaultNA(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", "N/A"},
		{"non-empty", "v1.0.0", "v1.0.0"},
		{"whitespace only", "  ", "  "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultNA(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestLogBuildInfo(t *testing.T) {
	tests := []struct {
		name    string
		version string
		date    string
		commit  string
	}{
		{"all empty", "", "", ""},
		{"all filled", "1.2.3", "2025-02-09", "abc123"},
		{"mixed - version empty", "", "2025-02-09", "abc123"},
		{"mixed - date empty", "1.0.0", "", "def456"},
		{"mixed - commit empty", "2.0.0", "2025-01-01", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t).Sugar()
			assert.NotPanics(t, func() {
				LogBuildInfo(tt.version, tt.date, tt.commit, logger)
			})
		})
	}
}

func TestLogBuildInfo_NilLogger_NoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		LogBuildInfo("", "", "", nil)
	})
}

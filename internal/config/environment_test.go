package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEnvironment(t *testing.T) {
	tests := []struct {
		name        string
		envKey      string
		envValue    string
		expected    string
		expectError bool
	}{
		{
			name:        "existing environment variable",
			envKey:      "TEST_VAR",
			envValue:    "test_value",
			expected:    "test_value",
			expectError: false,
		},
		{
			name:        "non-existing environment variable",
			envKey:      "NON_EXISTING_VAR",
			envValue:    "",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.envKey)

			if tt.envValue != "" {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			}

			result, err := GetEnvironment(tt.envKey)

			if tt.expectError {
				require.Error(t, err, "expected error for non-existing environment variable")
				assert.Empty(t, result)
			} else {
				require.NoError(t, err, "expected no error for existing environment variable")
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetEnvironmentInt(t *testing.T) {
	tests := []struct {
		name        string
		envKey      string
		envValue    string
		expected    int
		expectError bool
	}{
		{
			name:        "valid integer",
			envKey:      "TEST_INT",
			envValue:    "42",
			expected:    42,
			expectError: false,
		},
		{
			name:        "invalid integer",
			envKey:      "TEST_INVALID_INT",
			envValue:    "not_a_number",
			expected:    0,
			expectError: true,
		},
		{
			name:        "non-existing variable",
			envKey:      "NON_EXISTING_INT",
			envValue:    "",
			expected:    0,
			expectError: true,
		},
		{
			name:        "negative integer",
			envKey:      "TEST_NEGATIVE",
			envValue:    "-5",
			expected:    -5,
			expectError: false,
		},
		{
			name:        "zero",
			envKey:      "TEST_ZERO",
			envValue:    "0",
			expected:    0,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.envKey)

			if tt.envValue != "" {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			}

			result, err := GetEnvironmentInt(tt.envKey)

			if tt.expectError {
				require.Error(t, err, "expected error for invalid environment variable")
				assert.Equal(t, 0, result)
			} else {
				require.NoError(t, err, "expected no error for valid environment variable")
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

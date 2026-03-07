package network

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsIPInTrustedSubnet(t *testing.T) {
	tests := []struct {
		name          string
		ipStr         string
		trustedSubnet string
		wantAllowed   bool
		wantErr       bool
	}{
		{"empty subnet - allows any", "192.168.1.1", "", true, false},
		{"empty subnet - empty IP allowed", "", "", true, false},
		{"in subnet", "127.0.0.1", "127.0.0.0/8", true, false},
		{"outside subnet", "10.0.0.1", "127.0.0.0/8", false, false},
		{"empty IP with subnet", "", "127.0.0.0/8", false, false},
		{"invalid IP", "not-an-ip", "127.0.0.0/8", false, false},
		{"invalid CIDR", "127.0.0.1", "invalid", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := IsIPInTrustedSubnet(tt.ipStr, tt.trustedSubnet)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantAllowed, allowed)
		})
	}
}

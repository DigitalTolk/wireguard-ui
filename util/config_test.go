package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseBasePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"just slash", "/", ""},
		{"no leading slash", "app", "/app"},
		{"trailing slash", "/app/", "/app"},
		{"both missing and trailing", "app/", "/app"},
		{"proper path", "/app", "/app"},
		{"nested path", "/app/v1", "/app/v1"},
		{"nested with trailing", "/app/v1/", "/app/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseBasePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSubnetRanges(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedCount int
		expectedOrder []string
	}{
		{"empty string", "", 0, nil},
		{"single range", "LAN:10.0.0.0/24", 1, []string{"LAN"}},
		{"multiple CIDRs in one range", "LAN:10.0.0.0/24,10.1.0.0/24", 1, []string{"LAN"}},
		{"multiple ranges", "LAN:10.0.0.0/24;REMOTE:192.168.0.0/24", 2, []string{"LAN", "REMOTE"}},
		{"invalid CIDR skipped", "LAN:invalid", 0, nil},
		{"bad format skipped", "NOCOLON", 0, nil},
		{"whitespace handling", " LAN : 10.0.0.0/24 ; REMOTE : 192.168.0.0/24 ", 2, []string{"LAN", "REMOTE"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// reset global state
			SubnetRangesOrder = nil
			result := ParseSubnetRanges(tt.input)
			assert.Equal(t, tt.expectedCount, len(result))
			if tt.expectedOrder != nil {
				assert.Equal(t, tt.expectedOrder, SubnetRangesOrder)
			}
		})
	}
}

func TestParseSubnetRanges_DuplicateCIDR(t *testing.T) {
	SubnetRangesOrder = nil
	result := ParseSubnetRanges("A:10.0.0.0/24;B:10.0.0.0/24")
	// duplicate CIDR should be skipped in the second range
	assert.Equal(t, 1, len(result["A"]))
	// B should have been removed since its only CIDR was a duplicate
	_, hasB := result["B"]
	assert.False(t, hasB)
}

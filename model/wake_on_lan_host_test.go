package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveResourceName(t *testing.T) {
	tests := []struct {
		name      string
		mac       string
		expected  string
		expectErr bool
	}{
		{"valid colon format", "aa:bb:cc:dd:ee:ff", "AA-BB-CC-DD-EE-FF", false},
		{"valid dash format", "AA-BB-CC-DD-EE-FF", "AA-BB-CC-DD-EE-FF", false},
		{"lowercase colon", "01:02:03:04:05:06", "01-02-03-04-05-06", false},
		{"empty mac", "", "", true},
		{"whitespace only", "   ", "", true},
		{"invalid mac", "not-a-mac", "", true},
		{"with leading whitespace", "  aa:bb:cc:dd:ee:ff  ", "AA-BB-CC-DD-EE-FF", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := WakeOnLanHost{MacAddress: tt.mac}
			result, err := host.ResolveResourceName()
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

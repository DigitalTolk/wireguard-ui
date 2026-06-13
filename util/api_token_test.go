package util

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateAPIToken_ShapeAndUniqueness(t *testing.T) {
	a, err := GenerateAPIToken()
	assert.NoError(t, err)
	b, err := GenerateAPIToken()
	assert.NoError(t, err)

	assert.True(t, strings.HasPrefix(a, APITokenPrefix))
	// prefix + 48 hex chars (24 bytes)
	assert.Len(t, a, len(APITokenPrefix)+48)
	assert.NotEqual(t, a, b, "two generated tokens must differ")
	assert.True(t, LooksLikeAPIToken(a))
}

func TestHashAPIToken_DeterministicAndDiffersFromInput(t *testing.T) {
	plain := "wgui_0123456789abcdef0123456789abcdef0123456789abcdef"
	h1 := HashAPIToken(plain)
	h2 := HashAPIToken(plain)
	assert.Equal(t, h1, h2, "hash must be deterministic")
	assert.NotEqual(t, plain, h1, "hash must not echo the plaintext back")
	assert.Len(t, h1, 64) // sha256 hex
}

func TestLooksLikeAPIToken_Rejections(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"empty", "", false},
		{"missing prefix", "0123456789abcdef0123456789abcdef0123456789abcdef", false},
		{"short hex", "wgui_deadbeef", false},
		{"non-hex chars", "wgui_zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz", false},
		{"valid", "wgui_0123456789abcdef0123456789abcdef0123456789abcdef", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, LooksLikeAPIToken(tc.in))
		})
	}
}

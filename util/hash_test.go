package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("mypassword")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	// hash is base64 encoded
	assert.Greater(t, len(hash), 20)

	// different calls produce different hashes (bcrypt salt)
	hash2, err := HashPassword("mypassword")
	require.NoError(t, err)
	assert.NotEqual(t, hash, hash2)
}

func TestVerifyHash(t *testing.T) {
	hash, err := HashPassword("testpassword")
	require.NoError(t, err)

	match, err := VerifyHash(hash, "testpassword")
	assert.NoError(t, err)
	assert.True(t, match)

	match, err = VerifyHash(hash, "wrongpassword")
	assert.NoError(t, err)
	assert.False(t, match)
}

func TestVerifyHash_InvalidBase64(t *testing.T) {
	_, err := VerifyHash("not-base64!!!", "password")
	assert.Error(t, err)
}

func TestVerifyHash_InvalidBcryptHash(t *testing.T) {
	// Valid base64 but not a valid bcrypt hash
	_, err := VerifyHash("aGVsbG93b3JsZA==", "password")
	assert.Error(t, err)
}

func TestHashPassword_EmptyPassword(t *testing.T) {
	hash, err := HashPassword("")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
}

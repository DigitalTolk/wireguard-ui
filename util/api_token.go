package util

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// APITokenPrefix marks the string as an API token so secret-scanners (and
// humans) can recognize it at a glance. Any change here breaks existing
// tokens — so don't change it.
const APITokenPrefix = "wgui_"

// apiTokenEntropyBytes is the count of random bytes encoded in each token
// (24 bytes = 192 bits). Far beyond any plausible brute-force budget for an
// indexed equality lookup.
const apiTokenEntropyBytes = 24

// GenerateAPIToken returns a new plaintext API token. The caller MUST persist
// only HashAPIToken(plaintext) and surface the plaintext to the user exactly
// once at creation.
func GenerateAPIToken() (string, error) {
	buf := make([]byte, apiTokenEntropyBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return APITokenPrefix + hex.EncodeToString(buf), nil
}

// HashAPIToken returns the hex-encoded SHA-256 of the token. The hash is
// what's stored in the api_tokens table and what we look up on each request.
// SHA-256 is fine here because the input is 192 bits of CSPRNG output —
// bcrypt-style adaptive hashing buys nothing against a high-entropy secret.
func HashAPIToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// LooksLikeAPIToken is a cheap pre-check the middleware uses so a malformed
// Authorization header skips the DB lookup entirely.
func LooksLikeAPIToken(token string) bool {
	if !strings.HasPrefix(token, APITokenPrefix) {
		return false
	}
	rest := strings.TrimPrefix(token, APITokenPrefix)
	if len(rest) != apiTokenEntropyBytes*2 {
		return false
	}
	if _, err := hex.DecodeString(rest); err != nil {
		return false
	}
	return true
}

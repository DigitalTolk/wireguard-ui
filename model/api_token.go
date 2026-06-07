package model

import "time"

// APIToken is a long-lived bearer token used by external automation to call the
// programmatic API (provision-client, delete-by-email). Tokens are admin-equivalent.
// Only the SHA-256 hash of the token is persisted; the plaintext is shown to the
// admin exactly once at creation time.
type APIToken struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	CreatedBy  string     `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
}

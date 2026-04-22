package model

import "time"

// User model
type User struct {
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	// PasswordHash takes precedence over Password.
	PasswordHash string    `json:"password_hash,omitempty"`
	Email        string    `json:"email"`
	DisplayName  string    `json:"display_name"`
	OIDCSub      string    `json:"oidc_sub,omitempty"`
	Admin        bool      `json:"admin"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

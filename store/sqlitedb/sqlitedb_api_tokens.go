package sqlitedb

import (
	"database/sql"
	"errors"
	"time"

	"github.com/DigitalTolk/wireguard-ui/model"
	"github.com/DigitalTolk/wireguard-ui/store"
)

// CreateAPIToken persists a new token. The plaintext is never stored — the
// caller passes the SHA-256 hash. The unique constraint on token_hash
// surfaces hash collisions (effectively impossible) as a SaveClient-style
// error rather than silent overwrite.
func (o *SqliteDB) CreateAPIToken(token model.APIToken, tokenHash string) error {
	_, err := o.db.Exec(
		`INSERT INTO api_tokens (id, name, token_hash, created_by, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		token.ID, token.Name, tokenHash, token.CreatedBy, token.CreatedAt,
	)
	return err
}

// ListAPITokens returns every token (revoked included) so admins can see the
// full audit history. Plaintext / hash are never returned.
func (o *SqliteDB) ListAPITokens() ([]model.APIToken, error) {
	rows, err := o.db.Query(
		`SELECT id, name, created_by, created_at, last_used_at, revoked_at
		 FROM api_tokens ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.APIToken
	for rows.Next() {
		t, err := scanAPIToken(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetAPITokenByHash looks up a token by its SHA-256 hash. Returns
// store.ErrAPITokenNotFound when no row matches; revoked tokens are returned as-is
// so the caller can distinguish "unknown" from "revoked" in the audit log.
func (o *SqliteDB) GetAPITokenByHash(tokenHash string) (model.APIToken, error) {
	row := o.db.QueryRow(
		`SELECT id, name, created_by, created_at, last_used_at, revoked_at
		 FROM api_tokens WHERE token_hash = ?`,
		tokenHash,
	)
	t, err := scanAPIToken(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.APIToken{}, store.ErrAPITokenNotFound
	}
	return t, err
}

// RevokeAPIToken stamps revoked_at on the token. Idempotent: revoking an
// already-revoked token keeps the original revocation timestamp.
func (o *SqliteDB) RevokeAPIToken(id string) error {
	res, err := o.db.Exec(
		`UPDATE api_tokens SET revoked_at = COALESCE(revoked_at, ?) WHERE id = ?`,
		time.Now().UTC(), id,
	)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return store.ErrAPITokenNotFound
	}
	return nil
}

// TouchAPITokenLastUsed records that the token was used. Used by the auth
// middleware on each successful request. Errors are surfaced to the caller
// but middleware should swallow them so a write failure doesn't break the
// real request path.
func (o *SqliteDB) TouchAPITokenLastUsed(id string, when time.Time) error {
	_, err := o.db.Exec(
		`UPDATE api_tokens SET last_used_at = ? WHERE id = ?`, when, id,
	)
	return err
}

func scanAPIToken(s scanner) (model.APIToken, error) {
	var t model.APIToken
	var lastUsed, revoked sql.NullTime
	if err := s.Scan(&t.ID, &t.Name, &t.CreatedBy, &t.CreatedAt, &lastUsed, &revoked); err != nil {
		return t, err
	}
	if lastUsed.Valid {
		t.LastUsedAt = &lastUsed.Time
	}
	if revoked.Valid {
		t.RevokedAt = &revoked.Time
	}
	return t, nil
}

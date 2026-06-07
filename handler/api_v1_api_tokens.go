package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/rs/xid"

	"github.com/DigitalTolk/wireguard-ui/model"
	"github.com/DigitalTolk/wireguard-ui/store"
	"github.com/DigitalTolk/wireguard-ui/util"
)

// APITokenAuth is the middleware for programmatic endpoints (provision-client,
// delete-by-email). Requires `Authorization: Bearer <token>`. On success it
// stamps the request as admin-equivalent via context keys consumed by
// currentUser / isAdmin, then asynchronously updates last_used_at.
//
// Tokens are admin-level — there are no scopes today. If scope-based access
// becomes a need, this is the seam to extend.
func APITokenAuth(db store.IStore) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			const prefix = "Bearer "
			if !strings.HasPrefix(header, prefix) {
				return apiUnauthorized(c, "Missing or malformed Authorization header")
			}
			plain := strings.TrimSpace(strings.TrimPrefix(header, prefix))
			if !util.LooksLikeAPIToken(plain) {
				// Cheap shape check — avoid a DB hit for obvious garbage.
				return apiUnauthorized(c, "Invalid token")
			}

			token, err := db.GetAPITokenByHash(util.HashAPIToken(plain))
			if err != nil {
				return apiUnauthorized(c, "Invalid token")
			}
			if token.RevokedAt != nil {
				return apiUnauthorized(c, "Token has been revoked")
			}

			// Stamp the request as a token-authenticated admin caller so
			// currentUser / isAdmin / audit logs all line up.
			c.Set(ctxKeyTokenCaller, fmt.Sprintf("api-token:%s", token.Name))
			c.Set(ctxKeyTokenAdmin, true)

			// Touch last_used_at synchronously. SQLite serializes writes
			// anyway, so a goroutine here just races with the request's own
			// writes (SaveClient etc.) for the same write lock. Errors are
			// logged but not surfaced — a stale last_used_at must never
			// break the real request path.
			if err := db.TouchAPITokenLastUsed(token.ID, time.Now().UTC()); err != nil {
				log.Warnf("APITokenAuth: failed to update last_used_at for %s: %v", token.ID, err)
			}

			return next(c)
		}
	}
}

// --- Admin-only CRUD over tokens ---

// apiTokenCreateRequest is the JSON body for POST /api/v1/api-tokens.
type apiTokenCreateRequest struct {
	Name string `json:"name"`
}

// apiTokenCreateResponse echoes the token metadata and includes the plaintext
// — surfaced exactly once, never persisted, never returned again.
type apiTokenCreateResponse struct {
	model.APIToken
	Token string `json:"token"`
}

// APIListAPITokens returns every token (revoked + active) so admins can audit.
func APIListAPITokens(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		tokens, err := db.ListAPITokens()
		if err != nil {
			return apiInternalError(c, "Cannot list API tokens")
		}
		// Always return a non-nil array — avoids `null` in the frontend.
		if tokens == nil {
			tokens = []model.APIToken{}
		}
		return c.JSON(http.StatusOK, tokens)
	}
}

// APICreateAPIToken mints a new token. The plaintext appears in the response
// once; the caller is responsible for storing it.
func APICreateAPIToken(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req apiTokenCreateRequest
		if err := c.Bind(&req); err != nil {
			return apiBadRequest(c, "Invalid request body")
		}
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			return apiBadRequest(c, "Token name is required")
		}
		if len(req.Name) > 128 {
			return apiBadRequest(c, "Token name must be 128 characters or fewer")
		}

		plain, err := util.GenerateAPIToken()
		if err != nil {
			return apiInternalError(c, "Cannot generate token")
		}
		tok := model.APIToken{
			ID:        xid.New().String(),
			Name:      req.Name,
			CreatedBy: currentUser(c),
			CreatedAt: time.Now().UTC(),
		}
		if err := db.CreateAPIToken(tok, util.HashAPIToken(plain)); err != nil {
			return apiInternalError(c, fmt.Sprintf("Cannot save token: %v", err))
		}

		auditLogEvent(c, "api_token.create", "api_token", tok.ID, map[string]string{"name": tok.Name})
		return c.JSON(http.StatusCreated, apiTokenCreateResponse{APIToken: tok, Token: plain})
	}
}

// APIRevokeAPIToken stamps revoked_at on the token. Idempotent at the store
// level; the HTTP response is 204 either way.
func APIRevokeAPIToken(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		if _, err := xid.FromString(id); err != nil {
			return apiBadRequest(c, "Invalid token ID")
		}
		if err := db.RevokeAPIToken(id); err != nil {
			if err == store.ErrAPITokenNotFound {
				return apiNotFound(c, "Token not found")
			}
			return apiInternalError(c, fmt.Sprintf("Cannot revoke token: %v", err))
		}
		auditLogEvent(c, "api_token.revoke", "api_token", id, nil)
		return c.NoContent(http.StatusNoContent)
	}
}

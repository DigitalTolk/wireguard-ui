package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DigitalTolk/wireguard-ui/model"
	"github.com/DigitalTolk/wireguard-ui/util"
)

// mintToken creates a token directly in the store and returns the plaintext
// so tests can send it as a Bearer header.
func mintToken(t *testing.T, env *testEnv, name string) (model.APIToken, string) {
	t.Helper()
	plain, err := util.GenerateAPIToken()
	require.NoError(t, err)
	tok := model.APIToken{
		ID:        xid.New().String(),
		Name:      name,
		CreatedBy: "admin",
		CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, env.db.CreateAPIToken(tok, util.HashAPIToken(plain)))
	return tok, plain
}

func TestAPITokenAuth_AcceptsValidBearer(t *testing.T) {
	env := setupTestEnv(t)
	_, plain := mintToken(t, env, "ok-token")

	called := false
	handler := APITokenAuth(env.db)(func(c echo.Context) error {
		called = true
		// inside the protected handler, the token caller is admin
		assert.True(t, isAdmin(c))
		assert.Equal(t, "api-token:ok-token", currentUser(c))
		return c.NoContent(http.StatusOK)
	})

	req, rec := jsonRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+plain)
	c := env.echo.NewContext(req, rec)
	require.NoError(t, handler(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called, "next() must be invoked when token is valid")
}

func TestAPITokenAuth_RejectsMissingHeader(t *testing.T) {
	env := setupTestEnv(t)
	handler := APITokenAuth(env.db)(func(c echo.Context) error {
		t.Fatal("next() must not be invoked when token is missing")
		return nil
	})
	req, rec := jsonRequest(http.MethodGet, "/x", nil)
	c := env.echo.NewContext(req, rec)
	require.NoError(t, handler(c))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAPITokenAuth_RejectsBadShape(t *testing.T) {
	env := setupTestEnv(t)
	handler := APITokenAuth(env.db)(func(c echo.Context) error {
		t.Fatal("next() must not be invoked on shape failure")
		return nil
	})
	req, rec := jsonRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-token")
	c := env.echo.NewContext(req, rec)
	require.NoError(t, handler(c))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAPITokenAuth_RejectsUnknownToken(t *testing.T) {
	env := setupTestEnv(t)
	plain, err := util.GenerateAPIToken()
	require.NoError(t, err)

	handler := APITokenAuth(env.db)(func(c echo.Context) error {
		t.Fatal("next() must not be invoked for an unknown token")
		return nil
	})
	req, rec := jsonRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+plain)
	c := env.echo.NewContext(req, rec)
	require.NoError(t, handler(c))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAPITokenAuth_RejectsRevokedToken(t *testing.T) {
	env := setupTestEnv(t)
	tok, plain := mintToken(t, env, "revoked")
	require.NoError(t, env.db.RevokeAPIToken(tok.ID))

	handler := APITokenAuth(env.db)(func(c echo.Context) error {
		t.Fatal("next() must not be invoked for a revoked token")
		return nil
	})
	req, rec := jsonRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+plain)
	c := env.echo.NewContext(req, rec)
	require.NoError(t, handler(c))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAPITokenAuth_TouchesLastUsed(t *testing.T) {
	env := setupTestEnv(t)
	tok, plain := mintToken(t, env, "touchable")

	handler := APITokenAuth(env.db)(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	req, rec := jsonRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+plain)
	c := env.echo.NewContext(req, rec)
	require.NoError(t, handler(c))

	got, err := env.db.GetAPITokenByHash(util.HashAPIToken(plain))
	require.NoError(t, err)
	require.NotNil(t, got.LastUsedAt, "last_used_at must be set by the middleware")
	assert.Equal(t, tok.ID, got.ID)
}

// --- CRUD handlers ---

func TestAPICreateAPIToken_ReturnsPlaintextOnce(t *testing.T) {
	env := setupTestEnv(t)
	body := apiTokenCreateRequest{Name: "deploy-bot"}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/api-tokens", body)
	c := env.echo.NewContext(req, rec)
	require.NoError(t, APICreateAPIToken(env.db)(c))
	require.Equal(t, http.StatusCreated, rec.Code)

	var resp apiTokenCreateResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "deploy-bot", resp.Name)
	assert.True(t, util.LooksLikeAPIToken(resp.Token))
	assert.NotEmpty(t, resp.ID)

	// The token actually authenticates after creation
	stored, err := env.db.GetAPITokenByHash(util.HashAPIToken(resp.Token))
	require.NoError(t, err)
	assert.Equal(t, resp.ID, stored.ID)
}

func TestAPICreateAPIToken_RejectsEmptyName(t *testing.T) {
	env := setupTestEnv(t)
	body := apiTokenCreateRequest{Name: "   "}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/api-tokens", body)
	c := env.echo.NewContext(req, rec)
	require.NoError(t, APICreateAPIToken(env.db)(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIListAPITokens_DoesNotLeakHashes(t *testing.T) {
	env := setupTestEnv(t)
	mintToken(t, env, "alpha")
	mintToken(t, env, "beta")

	req, rec := jsonRequest(http.MethodGet, "/api/v1/api-tokens", nil)
	c := env.echo.NewContext(req, rec)
	require.NoError(t, APIListAPITokens(env.db)(c))
	require.Equal(t, http.StatusOK, rec.Code)

	// Hash must never appear in the listing response body — defense in depth
	// in case future schema changes accidentally widen the SELECT.
	assert.NotContains(t, rec.Body.String(), "token_hash")
	assert.NotContains(t, rec.Body.String(), "wgui_")

	var tokens []model.APIToken
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&tokens))
	assert.Len(t, tokens, 2)
}

func TestAPIRevokeAPIToken_BlocksFurtherAuth(t *testing.T) {
	env := setupTestEnv(t)
	tok, plain := mintToken(t, env, "burn-me")

	req, rec := jsonRequest(http.MethodDelete, fmt.Sprintf("/api/v1/api-tokens/%s", tok.ID), nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(tok.ID)
	require.NoError(t, APIRevokeAPIToken(env.db)(c))
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Authenticating with the now-revoked token must fail.
	auth := APITokenAuth(env.db)(func(c echo.Context) error {
		t.Fatal("next() must not run after revoke")
		return nil
	})
	r2, rec2 := jsonRequest(http.MethodGet, "/x", nil)
	r2.Header.Set("Authorization", "Bearer "+plain)
	c2 := env.echo.NewContext(r2, rec2)
	require.NoError(t, auth(c2))
	assert.Equal(t, http.StatusUnauthorized, rec2.Code)
}

func TestAPIRevokeAPIToken_RejectsBadID(t *testing.T) {
	env := setupTestEnv(t)
	req, rec := jsonRequest(http.MethodDelete, "/api/v1/api-tokens/not-an-xid", nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("not-an-xid")
	require.NoError(t, APIRevokeAPIToken(env.db)(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIRevokeAPIToken_UnknownIDReturns404(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	req, rec := jsonRequest(http.MethodDelete, "/api/v1/api-tokens/"+id, nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	require.NoError(t, APIRevokeAPIToken(env.db)(c))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

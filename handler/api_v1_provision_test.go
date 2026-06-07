package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Provision: happy paths ---

func TestProvision_ConfigDelivery_ReturnsConfBytes(t *testing.T) {
	env := setupTestEnv(t)
	mailer := &mockEmailer{}

	req, rec := jsonRequest(http.MethodPost, "/api/v1/provision-client", map[string]string{
		"email": "alice.smith@example.com", "delivery": "config",
	})
	c := env.echo.NewContext(req, rec)
	require.NoError(t, APIProvisionClient(env.db, env.cw, mailer, "subj", "body")(c))
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "text/conf", rec.Header().Get(echo.HeaderContentType))
	assert.Contains(t, rec.Header().Get(echo.HeaderContentDisposition), ".conf")
	body := rec.Body.String()
	assert.Contains(t, body, "[Interface]", "config body must contain a WG interface section")
	assert.Contains(t, body, "[Peer]", "config body must contain a WG peer section")
}

func TestProvision_QRCodeDelivery_ReturnsPNG(t *testing.T) {
	env := setupTestEnv(t)
	mailer := &mockEmailer{}

	req, rec := jsonRequest(http.MethodPost, "/api/v1/provision-client", map[string]string{
		"email": "qr.user@example.com", "delivery": "qrcode",
	})
	c := env.echo.NewContext(req, rec)
	require.NoError(t, APIProvisionClient(env.db, env.cw, mailer, "subj", "body")(c))
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "image/png", rec.Header().Get(echo.HeaderContentType))
	require.Greater(t, rec.Body.Len(), 100, "PNG body should be non-trivially sized")
	assert.Equal(t, []byte{0x89, 0x50, 0x4E, 0x47}, rec.Body.Bytes()[:4], "PNG magic bytes")
}

func TestProvision_EmailDelivery_FiresMailerAndReturnsJSON(t *testing.T) {
	env := setupTestEnv(t)
	mailer := &mockEmailer{}

	req, rec := jsonRequest(http.MethodPost, "/api/v1/provision-client", map[string]string{
		"email": "send.me@example.com", "delivery": "email",
	})
	c := env.echo.NewContext(req, rec)
	require.NoError(t, APIProvisionClient(env.db, env.cw, mailer, "subj", "body")(c))
	require.Equal(t, http.StatusCreated, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "send.me@example.com", resp["email"])
	assert.Equal(t, true, resp["sent"])
	assert.NotEmpty(t, resp["id"])

	assert.True(t, mailer.sendCalled, "mailer must be invoked")
	assert.Equal(t, "send.me@example.com", mailer.lastTo)
	require.GreaterOrEqual(t, len(mailer.lastAttachments), 1)
	assert.Contains(t, mailer.lastAttachments[0].Name, ".conf")
}

// --- Provision: naming via Client Name pattern ---

func TestProvision_UsesClientNamePattern(t *testing.T) {
	env := setupTestEnv(t)
	gs, err := env.db.GetGlobalSettings()
	require.NoError(t, err)
	gs.ClientNamePattern = `^([A-Za-z0-9]+)\.([A-Za-z0-9]+)@.+$`
	gs.ClientNameReplacement = "$1-$2"
	require.NoError(t, env.db.SaveGlobalSettings(gs))

	mailer := &mockEmailer{}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/provision-client", map[string]string{
		"email": "first.last@example.com", "delivery": "email",
	})
	c := env.echo.NewContext(req, rec)
	require.NoError(t, APIProvisionClient(env.db, env.cw, mailer, "subj", "body")(c))
	require.Equal(t, http.StatusCreated, rec.Code)

	clients, _ := env.db.GetClients(false)
	require.Len(t, clients, 1)
	assert.Equal(t, "first-last", clients[0].Client.Name)
}

func TestProvision_FallsBackToLocalPart_WhenNoPattern(t *testing.T) {
	env := setupTestEnv(t)
	mailer := &mockEmailer{}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/provision-client", map[string]string{
		"email": "weird+tag.user@example.com", "delivery": "email",
	})
	c := env.echo.NewContext(req, rec)
	require.NoError(t, APIProvisionClient(env.db, env.cw, mailer, "subj", "body")(c))
	require.Equal(t, http.StatusCreated, rec.Code)
	clients, _ := env.db.GetClients(false)
	require.Len(t, clients, 1)
	assert.Equal(t, "weirdtaguser", clients[0].Client.Name, "fallback strips non-alnum from local part")
}

// --- Provision: failure modes ---

func TestProvision_RejectsInvalidEmail(t *testing.T) {
	env := setupTestEnv(t)
	req, rec := jsonRequest(http.MethodPost, "/api/v1/provision-client", map[string]string{
		"email": "not an email", "delivery": "config",
	})
	c := env.echo.NewContext(req, rec)
	require.NoError(t, APIProvisionClient(env.db, env.cw, &mockEmailer{}, "s", "b")(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestProvision_RejectsUnknownDelivery(t *testing.T) {
	env := setupTestEnv(t)
	req, rec := jsonRequest(http.MethodPost, "/api/v1/provision-client", map[string]string{
		"email": "ok@example.com", "delivery": "magic",
	})
	c := env.echo.NewContext(req, rec)
	require.NoError(t, APIProvisionClient(env.db, env.cw, &mockEmailer{}, "s", "b")(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestProvision_RejectsDuplicateName_409(t *testing.T) {
	env := setupTestEnv(t)
	mailer := &mockEmailer{}

	// First call succeeds.
	req1, rec1 := jsonRequest(http.MethodPost, "/api/v1/provision-client", map[string]string{
		"email": "carl@example.com", "delivery": "email",
	})
	c1 := env.echo.NewContext(req1, rec1)
	require.NoError(t, APIProvisionClient(env.db, env.cw, mailer, "s", "b")(c1))
	require.Equal(t, http.StatusCreated, rec1.Code)

	// Same derived name → 409.
	req2, rec2 := jsonRequest(http.MethodPost, "/api/v1/provision-client", map[string]string{
		"email": "carl@example.com", "delivery": "email",
	})
	c2 := env.echo.NewContext(req2, rec2)
	require.NoError(t, APIProvisionClient(env.db, env.cw, mailer, "s", "b")(c2))
	assert.Equal(t, http.StatusConflict, rec2.Code)
}

// --- Provision: integration with the token-auth middleware ---

func TestProvision_BehindTokenAuth_RequiresValidToken(t *testing.T) {
	env := setupTestEnv(t)
	env.echo.POST("/api/v1/provision-client",
		APIProvisionClient(env.db, env.cw, &mockEmailer{}, "s", "b"),
		APITokenAuth(env.db), ContentTypeJson)

	req, rec := jsonRequest(http.MethodPost, "/api/v1/provision-client", map[string]string{
		"email": "x@y.com", "delivery": "config",
	})
	// No Authorization header — must be rejected before the handler runs.
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "no token must produce 401")
}

func TestProvision_BehindTokenAuth_HappyPathWithToken(t *testing.T) {
	env := setupTestEnv(t)
	mailer := &mockEmailer{}
	_, plain := mintToken(t, env, "provision-bot")

	env.echo.POST("/api/v1/provision-client",
		APIProvisionClient(env.db, env.cw, mailer, "s", "b"),
		APITokenAuth(env.db), ContentTypeJson)

	req, rec := jsonRequest(http.MethodPost, "/api/v1/provision-client", map[string]string{
		"email": "token.user@example.com", "delivery": "email",
	})
	req.Header.Set("Authorization", "Bearer "+plain)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.True(t, mailer.sendCalled, "valid token request must reach the handler and trigger mailer")
}

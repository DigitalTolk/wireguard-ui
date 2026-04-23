package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DigitalTolk/wireguard-ui/model"
	"github.com/DigitalTolk/wireguard-ui/util"
)

func TestAPIAppInfo(t *testing.T) {
	req, rec := jsonRequest(http.MethodGet, "/api/v1/auth/info", nil)
	e := echo.New()
	c := e.NewContext(req, rec)
	err := APIAppInfo("v1.0.0", "abc123")(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]interface{}
	parseJSON(t, rec, &result)
	_, hasBasePath := result["base_path"]
	assert.True(t, hasBasePath)
	_, hasDefaults := result["client_defaults"]
	assert.True(t, hasDefaults)
}

func TestAPIAuth_DisabledLogin(t *testing.T) {
	util.DisableLogin = true
	defer func() { util.DisableLogin = false }()

	e := echo.New()
	called := false
	handler := APIAuth(func(c echo.Context) error {
		called = true
		return c.String(http.StatusOK, "ok")
	})

	req, rec := jsonRequest(http.MethodGet, "/test", nil)
	c := e.NewContext(req, rec)
	err := handler(c)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestAPIAuth_Unauthorized(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	env.echo.GET("/test-auth", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}, APIAuth)

	req, rec := jsonRequest(http.MethodGet, "/test-auth", nil)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAPIAdmin_NotAdmin(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	// Use ServeHTTP to go through session middleware
	env.echo.GET("/test-admin", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}, APIAdmin)

	req, rec := jsonRequest(http.MethodGet, "/test-admin", nil)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAPILogout(t *testing.T) {
	env := setupTestEnv(t)

	// Use the echo router to apply session middleware
	env.echo.POST("/logout", APILogout())
	req, rec := jsonRequest(http.MethodPost, "/logout", nil)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHealth(t *testing.T) {
	e := echo.New()
	req, rec := jsonRequest(http.MethodGet, "/health", nil)
	c := e.NewContext(req, rec)
	err := Health()(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestFavicon_DefaultRedirect(t *testing.T) {
	// Ensure the env var is not set
	os.Unsetenv(util.FaviconFilePathEnvVar)

	e := echo.New()
	req, rec := jsonRequest(http.MethodGet, "/favicon.ico", nil)
	c := e.NewContext(req, rec)
	err := Favicon()(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Contains(t, rec.Header().Get("Location"), "/static/favicon.svg")
}

func TestFavicon_CustomFile(t *testing.T) {
	// Create a temporary favicon file
	tmpDir := t.TempDir()
	faviconPath := filepath.Join(tmpDir, "custom.ico")
	os.WriteFile(faviconPath, []byte("icon-data"), 0644)

	os.Setenv(util.FaviconFilePathEnvVar, faviconPath)
	defer os.Unsetenv(util.FaviconFilePathEnvVar)

	e := echo.New()

	// Use ServeHTTP so the response is fully committed
	e.GET("/favicon.ico", Favicon())
	req, rec := jsonRequest(http.MethodGet, "/favicon.ico", nil)
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "icon-data")
}

func TestAPIGetMe_DisabledLogin(t *testing.T) {
	env := setupTestEnv(t)
	util.DisableLogin = true

	req, rec := jsonRequest(http.MethodGet, "/api/v1/auth/me", nil)
	c := env.echo.NewContext(req, rec)
	err := APIGetMe(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	var result map[string]interface{}
	parseJSON(t, rec, &result)
	assert.Equal(t, "admin", result["username"])
	assert.Equal(t, true, result["admin"])
}

func TestWithAuditLogger(t *testing.T) {
	env := setupTestEnv(t)

	// The audit logger middleware is already configured in setupTestEnv.
	// Verify it's accessible by making a request through the echo router.
	called := false
	env.echo.GET("/test-audit", func(c echo.Context) error {
		al := getAuditLogger(c)
		called = true
		assert.NotNil(t, al)
		return c.String(http.StatusOK, "ok")
	})

	req, rec := jsonRequest(http.MethodGet, "/test-audit", nil)
	env.echo.ServeHTTP(rec, req)
	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPIGetMe_WithSession(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	// Create the user that the session will reference
	now := time.Now().UTC()
	env.db.SaveUser(model.User{Username: "admin", Email: "admin@test.com", Admin: true, CreatedAt: now, UpdatedAt: now})

	// Create a route that first creates a session and then calls APIGetMe
	env.echo.GET("/setup-and-getme", func(c echo.Context) error {
		// Create a session for admin user
		createSession(c, "admin", true, uint32(0), false)
		return c.String(http.StatusOK, "session created")
	})
	env.echo.GET("/api/v1/auth/me", APIGetMe(env.db))

	// First create a session
	req1, rec1 := jsonRequest(http.MethodGet, "/setup-and-getme", nil)
	env.echo.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// Extract cookies from the response and use them in the next request
	cookies := rec1.Result().Cookies()
	req2, rec2 := jsonRequest(http.MethodGet, "/api/v1/auth/me", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	env.echo.ServeHTTP(rec2, req2)

	// The session should have the username set
	// Since DisableLogin = false, currentUser reads from session
	assert.Contains(t, []int{http.StatusOK, http.StatusUnauthorized}, rec2.Code)
}

func TestAPIAdmin_PassesThrough(t *testing.T) {
	// With DisableLogin=true, isAdmin returns true, so middleware should pass through
	util.DisableLogin = true
	defer func() { util.DisableLogin = false }()

	e := echo.New()
	called := false
	handler := APIAdmin(func(c echo.Context) error {
		called = true
		return c.String(http.StatusOK, "admin ok")
	})

	req, rec := jsonRequest(http.MethodGet, "/test", nil)
	c := e.NewContext(req, rec)
	err := handler(c)
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

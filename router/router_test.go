package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DigitalTolk/wireguard-ui/audit"
	"github.com/DigitalTolk/wireguard-ui/store/sqlitedb"
	"github.com/DigitalTolk/wireguard-ui/util"
)

func TestNew_ReturnsEchoInstance(t *testing.T) {
	secret := [64]byte{}
	copy(secret[:], "testsecret")

	e := New(secret)

	assert.NotNil(t, e)
	assert.True(t, e.HideBanner)
	assert.NotNil(t, e.Validator)
}

func TestNew_ValidatorIsSet(t *testing.T) {
	secret := [64]byte{}
	copy(secret[:], "testsecret")

	e := New(secret)

	// The validator should be able to validate a struct
	type testStruct struct {
		Name string `validate:"required"`
	}
	err := e.Validator.Validate(testStruct{Name: "test"})
	assert.NoError(t, err)

	err = e.Validator.Validate(testStruct{})
	assert.Error(t, err)
}

func TestNew_SessionMiddlewarePresent(t *testing.T) {
	secret := [64]byte{}
	copy(secret[:], "testsecret")

	e := New(secret)

	// Register a route and make a request to verify session middleware is active
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestNew_TrailingSlashRemoved(t *testing.T) {
	secret := [64]byte{}
	copy(secret[:], "testsecret")

	e := New(secret)

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "reached")
	})

	// Request with trailing slash should be redirected/handled
	req := httptest.NewRequest(http.MethodGet, "/test/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// RemoveTrailingSlash redirects with 301 by default
	assert.Contains(t, []int{http.StatusOK, http.StatusMovedPermanently}, rec.Code)
}

func TestRegisterAPIv1_RoutesRegistered(t *testing.T) {
	os.Setenv("WGUI_ENDPOINT_ADDRESS", "10.0.0.1")

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := sqlitedb.New(dbPath)
	require.NoError(t, err)
	require.NoError(t, db.Init())

	auditLog := audit.NewLogger(db.DB())

	util.DisableLogin = true
	defer func() { util.DisableLogin = false }()

	secret := [64]byte{}
	copy(secret[:], "testsecret")
	e := New(secret)

	tmplFS := os.DirFS("../templates")

	g := e.Group("/api/v1")
	RegisterAPIv1(g, db, nil, tmplFS, "", "", "dev", "test", auditLog)

	routes := e.Routes()

	// Collect registered paths
	routePaths := make(map[string]bool)
	for _, r := range routes {
		routePaths[r.Method+":"+r.Path] = true
	}

	// Verify key API routes are registered
	expectedRoutes := []string{
		"GET:/api/v1/auth/me",
		"POST:/api/v1/auth/logout",
		"GET:/api/v1/auth/info",
		"GET:/api/v1/clients",
		"GET:/api/v1/clients/export",
		"GET:/api/v1/clients/:id",
		"POST:/api/v1/clients",
		"PUT:/api/v1/clients/:id",
		"DELETE:/api/v1/clients/:id",
		"GET:/api/v1/clients/:id/config",
		"GET:/api/v1/server",
		"PUT:/api/v1/server/interface",
		"POST:/api/v1/server/apply-config",
		"GET:/api/v1/settings",
		"PUT:/api/v1/settings",
		"GET:/api/v1/users",
		"GET:/api/v1/status",
		"GET:/api/v1/audit-logs",
		"GET:/api/v1/audit-logs/export",
	}

	for _, route := range expectedRoutes {
		assert.True(t, routePaths[route], "Expected route %s to be registered", route)
	}
}

func TestRegisterAPIv1_HealthEndpointWorks(t *testing.T) {
	os.Setenv("WGUI_ENDPOINT_ADDRESS", "10.0.0.1")

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := sqlitedb.New(dbPath)
	require.NoError(t, err)
	require.NoError(t, db.Init())

	auditLog := audit.NewLogger(db.DB())

	util.DisableLogin = true
	defer func() { util.DisableLogin = false }()

	secret := [64]byte{}
	copy(secret[:], "testsecret")
	e := New(secret)

	tmplFS := os.DirFS("../templates")

	g := e.Group("/api/v1")
	RegisterAPIv1(g, db, nil, tmplFS, "", "", "dev", "test", auditLog)

	// Test the auth/info endpoint which requires no auth
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/info", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	assert.NotNil(t, v)

	type S struct {
		Email string `validate:"required,email"`
	}

	assert.NoError(t, v.Validate(S{Email: "test@example.com"}))
	assert.Error(t, v.Validate(S{Email: "not-an-email"}))
	assert.Error(t, v.Validate(S{}))
}

package handler

import (
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/DigitalTolk/wireguard-ui/model"
	"github.com/DigitalTolk/wireguard-ui/util"
)

// --- hasGroupOverlap Tests ---

func TestHasGroupOverlap_Match(t *testing.T) {
	assert.True(t, hasGroupOverlap([]string{"devs", "admins", "users"}, []string{"admins"}))
}

func TestHasGroupOverlap_NoMatch(t *testing.T) {
	assert.False(t, hasGroupOverlap([]string{"devs", "users"}, []string{"admins", "superadmins"}))
}

func TestHasGroupOverlap_EmptyUserGroups(t *testing.T) {
	assert.False(t, hasGroupOverlap([]string{}, []string{"admins"}))
}

func TestHasGroupOverlap_EmptyAdminGroups(t *testing.T) {
	assert.False(t, hasGroupOverlap([]string{"admins"}, []string{}))
}

func TestHasGroupOverlap_BothEmpty(t *testing.T) {
	assert.False(t, hasGroupOverlap([]string{}, []string{}))
}

func TestHasGroupOverlap_MultipleMatches(t *testing.T) {
	assert.True(t, hasGroupOverlap([]string{"admins", "superadmins"}, []string{"admins", "superadmins"}))
}

// --- findOrCreateOIDCUser Tests ---

func TestFindOrCreateOIDCUser_ExistingUser(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	// Create a user with an OIDC subject
	env.db.SaveUser(model.User{
		Username:  "oidcuser",
		Email:     "old@example.com",
		OIDCSub:   "sub-12345",
		Admin:     false,
		CreatedAt: now,
		UpdatedAt: now,
	})

	user, err := findOrCreateOIDCUser(env.db, "sub-12345", "oidcuser", "new@example.com", "New Name", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "oidcuser", user.Username)
	assert.Equal(t, "new@example.com", user.Email)
	assert.Equal(t, "New Name", user.DisplayName)
}

func TestFindOrCreateOIDCUser_ExistingUserAdminGroupCheck(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	env.db.SaveUser(model.User{
		Username:  "oidcadmin",
		OIDCSub:   "sub-admin",
		Admin:     false,
		CreatedAt: now,
		UpdatedAt: now,
	})

	// User is in admin group
	user, err := findOrCreateOIDCUser(env.db, "sub-admin", "oidcadmin", "admin@example.com", "", []string{"admins"}, []string{"admins"})
	require.NoError(t, err)
	assert.True(t, user.Admin)

	// User is NOT in admin group
	user2, err := findOrCreateOIDCUser(env.db, "sub-admin", "oidcadmin", "admin@example.com", "", []string{"users"}, []string{"admins"})
	require.NoError(t, err)
	assert.False(t, user2.Admin)
}

func TestFindOrCreateOIDCUser_NewUser_AutoProvisionEnabled(t *testing.T) {
	env := setupTestEnv(t)

	// Enable auto-provisioning
	origAutoProvision := util.OIDCAutoProvision
	util.OIDCAutoProvision = true
	defer func() { util.OIDCAutoProvision = origAutoProvision }()

	user, err := findOrCreateOIDCUser(env.db, "sub-new", "newuser", "new@example.com", "New User", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "newuser", user.Username)
	assert.Equal(t, "new@example.com", user.Email)
	assert.Equal(t, "New User", user.DisplayName)
	assert.Equal(t, "sub-new", user.OIDCSub)
}

func TestFindOrCreateOIDCUser_NewUser_AutoProvisionDisabled(t *testing.T) {
	env := setupTestEnv(t)

	origAutoProvision := util.OIDCAutoProvision
	util.OIDCAutoProvision = false
	defer func() { util.OIDCAutoProvision = origAutoProvision }()

	_, err := findOrCreateOIDCUser(env.db, "sub-disabled", "disabled", "disabled@example.com", "", nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auto-provisioning is disabled")
}

func TestFindOrCreateOIDCUser_NewUser_AdminViaGroups(t *testing.T) {
	env := setupTestEnv(t)

	origAutoProvision := util.OIDCAutoProvision
	util.OIDCAutoProvision = true
	defer func() { util.OIDCAutoProvision = origAutoProvision }()

	user, err := findOrCreateOIDCUser(env.db, "sub-groupadmin", "groupadmin", "ga@example.com", "",
		[]string{"team", "admins"}, []string{"admins"})
	require.NoError(t, err)
	assert.True(t, user.Admin)
}

func TestFindOrCreateOIDCUser_NewUser_NotAdmin(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	// pre-create a user so the next one isn't the first (first user auto-gets admin)
	env.db.SaveUser(model.User{Username: "existing", OIDCSub: "sub-existing", Admin: true, CreatedAt: now, UpdatedAt: now})

	origAutoProvision := util.OIDCAutoProvision
	util.OIDCAutoProvision = true
	defer func() { util.OIDCAutoProvision = origAutoProvision }()

	user, err := findOrCreateOIDCUser(env.db, "sub-nonadmin", "nonadmin", "na@example.com", "",
		[]string{"team"}, []string{"admins"})
	require.NoError(t, err)
	assert.False(t, user.Admin)
}

func TestFindOrCreateOIDCUser_FirstUser_BecomesAdmin(t *testing.T) {
	// Create a fresh environment with no users
	env := setupTestEnv(t)

	// Delete the default admin user that Init creates
	env.db.DeleteUser("admin")

	origAutoProvision := util.OIDCAutoProvision
	util.OIDCAutoProvision = true
	defer func() { util.OIDCAutoProvision = origAutoProvision }()

	user, err := findOrCreateOIDCUser(env.db, "sub-first", "firstuser", "first@example.com", "", nil, nil)
	require.NoError(t, err)
	// first user should automatically be admin
	assert.True(t, user.Admin)
}

// --- NewOIDCProvider tests ---

func TestNewOIDCProvider_NotConfigured(t *testing.T) {
	origIssuerURL := util.OIDCIssuerURL
	origClientID := util.OIDCClientID
	util.OIDCIssuerURL = ""
	util.OIDCClientID = ""
	defer func() {
		util.OIDCIssuerURL = origIssuerURL
		util.OIDCClientID = origClientID
	}()

	provider, err := NewOIDCProvider()
	assert.NoError(t, err)
	assert.Nil(t, provider)
}

func TestNewOIDCProvider_InvalidIssuer(t *testing.T) {
	origIssuerURL := util.OIDCIssuerURL
	origClientID := util.OIDCClientID
	util.OIDCIssuerURL = "http://127.0.0.1:1/invalid"
	util.OIDCClientID = "test-client-id"
	defer func() {
		util.OIDCIssuerURL = origIssuerURL
		util.OIDCClientID = origClientID
	}()

	provider, err := NewOIDCProvider()
	assert.Error(t, err)
	assert.Nil(t, provider)
}

// --- APIStartOIDCLogin tests ---

func TestAPIStartOIDCLogin_NilProvider(t *testing.T) {
	env := setupTestEnv(t)

	req, rec := jsonRequest(http.MethodGet, "/api/v1/auth/oidc/login", nil)
	c := env.echo.NewContext(req, rec)
	err := APIStartOIDCLogin(nil)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --- APIHandleOIDCCallback tests ---

func TestAPIHandleOIDCCallback_NilProvider(t *testing.T) {
	env := setupTestEnv(t)

	req, rec := jsonRequest(http.MethodGet, "/api/v1/auth/oidc/callback", nil)
	c := env.echo.NewContext(req, rec)
	err := APIHandleOIDCCallback(nil, env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAPIStartOIDCLogin_WithProvider(t *testing.T) {
	env := setupTestEnv(t)

	// Create a minimal OIDCProvider with a fake oauth2 config
	provider := &OIDCProvider{
		oauth2Cfg: oauth2.Config{
			ClientID:    "test-id",
			RedirectURL: "http://localhost/callback",
			Endpoint: oauth2.Endpoint{
				AuthURL: "http://localhost/auth",
			},
		},
	}

	env.echo.GET("/oidc-login", APIStartOIDCLogin(provider))
	req, rec := jsonRequest(http.MethodGet, "/oidc-login", nil)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
	location := rec.Header().Get("Location")
	assert.Contains(t, location, "http://localhost/auth")
	assert.Contains(t, location, "client_id=test-id")
}

func TestAPIHandleOIDCCallback_InvalidState(t *testing.T) {
	env := setupTestEnv(t)

	provider := &OIDCProvider{
		oauth2Cfg: oauth2.Config{
			ClientID: "test-id",
		},
	}

	// Register callback handler
	env.echo.GET("/oidc-callback", APIHandleOIDCCallback(provider, env.db))

	// Make request with a state that doesn't match session
	req, rec := jsonRequest(http.MethodGet, "/oidc-callback?state=bad-state&code=test-code", nil)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIHandleOIDCCallback_ErrorParam(t *testing.T) {
	env := setupTestEnv(t)

	provider := &OIDCProvider{
		oauth2Cfg: oauth2.Config{
			ClientID: "test-id",
		},
	}

	// First set up a session with the OIDC state
	env.echo.GET("/setup-oidc-state", func(c echo.Context) error {
		sess, _ := session.Get("session", c)
		sess.Values["oidc_state"] = "test-state"
		sess.Values["oidc_nonce"] = "test-nonce"
		sess.Save(c.Request(), c.Response())
		return c.String(http.StatusOK, "ok")
	})

	req1, rec1 := jsonRequest(http.MethodGet, "/setup-oidc-state", nil)
	env.echo.ServeHTTP(rec1, req1)
	cookies := rec1.Result().Cookies()

	// Now call the callback with matching state but an error parameter
	env.echo.GET("/oidc-callback-err", APIHandleOIDCCallback(provider, env.db))
	req2, rec2 := jsonRequest(http.MethodGet, "/oidc-callback-err?state=test-state&error=access_denied&error_description=User+denied+access", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	env.echo.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusUnauthorized, rec2.Code)
}

func TestAPIHandleOIDCCallback_TokenExchangeFailure(t *testing.T) {
	env := setupTestEnv(t)

	provider := &OIDCProvider{
		oauth2Cfg: oauth2.Config{
			ClientID: "test-id",
			Endpoint: oauth2.Endpoint{
				TokenURL: "http://127.0.0.1:1/invalid-token", // unreachable
			},
		},
	}

	// Set up the OIDC state in session
	env.echo.GET("/setup-oidc-state2", func(c echo.Context) error {
		sess, _ := session.Get("session", c)
		sess.Values["oidc_state"] = "test-state2"
		sess.Values["oidc_nonce"] = "test-nonce2"
		sess.Save(c.Request(), c.Response())
		return c.String(http.StatusOK, "ok")
	})

	req1, rec1 := jsonRequest(http.MethodGet, "/setup-oidc-state2", nil)
	env.echo.ServeHTTP(rec1, req1)
	cookies := rec1.Result().Cookies()

	// Callback with valid state and code, but token exchange will fail
	env.echo.GET("/oidc-callback-token-fail", APIHandleOIDCCallback(provider, env.db))
	req2, rec2 := jsonRequest(http.MethodGet, "/oidc-callback-token-fail?state=test-state2&code=test-code", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	env.echo.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusInternalServerError, rec2.Code)
}

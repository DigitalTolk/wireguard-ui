package handler

import (
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DigitalTolk/wireguard-ui/util"
)

// suppress unused import warnings
var (
	_ = time.Now
	_ = session.Get
)

// --- getMaxAge tests ---

func TestGetMaxAge_WithIntValue(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	sess := sessions.NewSession(nil, "session")
	sess.Values["max_age"] = 3600
	assert.Equal(t, 3600, getMaxAge(sess))
}

func TestGetMaxAge_WithNonIntValue(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	sess := sessions.NewSession(nil, "session")
	sess.Values["max_age"] = "not-an-int"
	assert.Equal(t, 0, getMaxAge(sess))
}

func TestGetMaxAge_WithNilValue(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	sess := sessions.NewSession(nil, "session")
	assert.Equal(t, 0, getMaxAge(sess))
}

func TestGetMaxAge_DisabledLogin(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = true
	defer func() { util.DisableLogin = origDisable }()

	sess := sessions.NewSession(nil, "session")
	sess.Values["max_age"] = 3600
	assert.Equal(t, 0, getMaxAge(sess))
}

// --- getCreatedAt tests ---

func TestGetCreatedAt_WithInt64Value(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	sess := sessions.NewSession(nil, "session")
	sess.Values["created_at"] = int64(1700000000)
	assert.Equal(t, int64(1700000000), getCreatedAt(sess))
}

func TestGetCreatedAt_WithNonInt64Value(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	sess := sessions.NewSession(nil, "session")
	sess.Values["created_at"] = "not-a-number"
	assert.Equal(t, int64(0), getCreatedAt(sess))
}

func TestGetCreatedAt_WithNilValue(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	sess := sessions.NewSession(nil, "session")
	assert.Equal(t, int64(0), getCreatedAt(sess))
}

func TestGetCreatedAt_DisabledLogin(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = true
	defer func() { util.DisableLogin = origDisable }()

	sess := sessions.NewSession(nil, "session")
	sess.Values["created_at"] = int64(1700000000)
	assert.Equal(t, int64(0), getCreatedAt(sess))
}

// --- getUpdatedAt tests ---

func TestGetUpdatedAt_WithInt64Value(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	sess := sessions.NewSession(nil, "session")
	sess.Values["updated_at"] = int64(1700000000)
	assert.Equal(t, int64(1700000000), getUpdatedAt(sess))
}

func TestGetUpdatedAt_WithNonInt64Value(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	sess := sessions.NewSession(nil, "session")
	sess.Values["updated_at"] = 42
	assert.Equal(t, int64(0), getUpdatedAt(sess))
}

func TestGetUpdatedAt_WithNilValue(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	sess := sessions.NewSession(nil, "session")
	assert.Equal(t, int64(0), getUpdatedAt(sess))
}

func TestGetUpdatedAt_DisabledLogin(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = true
	defer func() { util.DisableLogin = origDisable }()

	sess := sessions.NewSession(nil, "session")
	sess.Values["updated_at"] = int64(1700000000)
	assert.Equal(t, int64(0), getUpdatedAt(sess))
}

// --- getUserHash tests ---

func TestGetUserHash_WithUint32Value(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	sess := sessions.NewSession(nil, "session")
	sess.Values["user_hash"] = uint32(12345)
	assert.Equal(t, uint32(12345), getUserHash(sess))
}

func TestGetUserHash_WithNonUint32Value(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	sess := sessions.NewSession(nil, "session")
	sess.Values["user_hash"] = "not-a-hash"
	assert.Equal(t, uint32(0), getUserHash(sess))
}

func TestGetUserHash_DisabledLogin(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = true
	defer func() { util.DisableLogin = origDisable }()

	sess := sessions.NewSession(nil, "session")
	sess.Values["user_hash"] = uint32(12345)
	assert.Equal(t, uint32(0), getUserHash(sess))
}

// --- currentUser tests ---

func TestCurrentUser_DisabledLogin(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = true
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = true

	req, rec := jsonRequest(http.MethodGet, "/test", nil)
	c := env.echo.NewContext(req, rec)

	assert.Equal(t, "", currentUser(c))
}

func TestCurrentUser_WithSession(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	called := false
	env.echo.GET("/test-user", func(c echo.Context) error {
		called = true
		// No session established, so username is the default fmt sprint of nil
		username := currentUser(c)
		assert.NotEmpty(t, username) // will be "<nil>" since no value set
		return c.String(http.StatusOK, username)
	})

	req, rec := jsonRequest(http.MethodGet, "/test-user", nil)
	env.echo.ServeHTTP(rec, req)
	assert.True(t, called)
}

// --- isAdmin tests ---

func TestIsAdmin_DisabledLogin(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = true
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = true

	req, rec := jsonRequest(http.MethodGet, "/test", nil)
	c := env.echo.NewContext(req, rec)

	// DisableLogin means isAdmin returns true
	assert.True(t, isAdmin(c))
}

// --- isValidSession tests ---

func TestIsValidSession_DisabledLogin(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = true
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = true

	req, rec := jsonRequest(http.MethodGet, "/test", nil)
	c := env.echo.NewContext(req, rec)

	assert.True(t, isValidSession(c))
}

func TestIsValidSession_NoSession(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	result := false
	env.echo.GET("/test-valid", func(c echo.Context) error {
		result = isValidSession(c)
		return c.String(http.StatusOK, "ok")
	})

	req, rec := jsonRequest(http.MethodGet, "/test-valid", nil)
	env.echo.ServeHTTP(rec, req)
	assert.False(t, result)
}

// --- createSession tests ---

func TestCreateSession(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	var sessionCreated bool
	env.echo.GET("/create-session", func(c echo.Context) error {
		createSession(c, "testuser", true, uint32(12345), false)
		sessionCreated = true
		return c.String(http.StatusOK, "ok")
	})

	req, rec := jsonRequest(http.MethodGet, "/create-session", nil)
	env.echo.ServeHTTP(rec, req)
	assert.True(t, sessionCreated)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify session_token cookie was set
	cookies := rec.Result().Cookies()
	found := false
	for _, cookie := range cookies {
		if cookie.Name == "session_token" {
			found = true
			assert.True(t, cookie.HttpOnly)
			break
		}
	}
	assert.True(t, found, "session_token cookie should be set")
}

func TestCreateSession_WithRememberMe(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	env.echo.GET("/create-session-remember", func(c echo.Context) error {
		createSession(c, "testuser", false, uint32(999), true)
		return c.String(http.StatusOK, "ok")
	})

	req, rec := jsonRequest(http.MethodGet, "/create-session-remember", nil)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify session_token cookie has MaxAge > 0
	cookies := rec.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "session_token" {
			assert.Greater(t, cookie.MaxAge, 0, "remember-me cookie should have positive MaxAge")
			break
		}
	}
}

// --- ValidSession tests ---

func TestValidSession_DisabledLogin(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = true
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = true

	called := false
	env.echo.GET("/test-valid-session", ValidSession(func(c echo.Context) error {
		called = true
		return c.String(http.StatusOK, "ok")
	}))

	req, rec := jsonRequest(http.MethodGet, "/test-valid-session", nil)
	env.echo.ServeHTTP(rec, req)
	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestValidSession_NoSession_Redirects(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	env.echo.GET("/protected", ValidSession(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}))

	req, rec := jsonRequest(http.MethodGet, "/protected", nil)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
	assert.Contains(t, rec.Header().Get("Location"), "/login")
}

// --- NeedsAdmin tests ---

func TestNeedsAdmin_DisabledLogin(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = true
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = true

	called := false
	env.echo.GET("/admin-only", NeedsAdmin(func(c echo.Context) error {
		called = true
		return c.String(http.StatusOK, "admin ok")
	}))

	req, rec := jsonRequest(http.MethodGet, "/admin-only", nil)
	env.echo.ServeHTTP(rec, req)
	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestNeedsAdmin_NotAdmin_Redirects(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	env.echo.GET("/admin-only", NeedsAdmin(func(c echo.Context) error {
		return c.String(http.StatusOK, "admin ok")
	}))

	req, rec := jsonRequest(http.MethodGet, "/admin-only", nil)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
}

// --- RefreshSession tests ---

func TestRefreshSession_DisabledLogin(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = true
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = true

	called := false
	env.echo.GET("/refresh-test", RefreshSession(func(c echo.Context) error {
		called = true
		return c.String(http.StatusOK, "ok")
	}))

	req, rec := jsonRequest(http.MethodGet, "/refresh-test", nil)
	env.echo.ServeHTTP(rec, req)
	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// --- setUser tests ---

func TestSetUser(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	env.echo.GET("/set-user", func(c echo.Context) error {
		setUser(c, "testuser", true, uint32(12345))
		return c.String(http.StatusOK, "ok")
	})

	req, rec := jsonRequest(http.MethodGet, "/set-user", nil)
	env.echo.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

// --- clearSession tests ---

func TestClearSession(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	env.echo.GET("/clear-session", func(c echo.Context) error {
		clearSession(c)
		return c.String(http.StatusOK, "ok")
	})

	req, rec := jsonRequest(http.MethodGet, "/clear-session", nil)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// --- doRefreshSession tests ---

func TestDoRefreshSession_DisabledLogin(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = true
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = true

	env.echo.GET("/do-refresh", func(c echo.Context) error {
		doRefreshSession(c) // should be a no-op
		return c.String(http.StatusOK, "ok")
	})

	req, rec := jsonRequest(http.MethodGet, "/do-refresh", nil)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDoRefreshSession_NoRememberMe(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	env.echo.GET("/do-refresh-no-remember", func(c echo.Context) error {
		doRefreshSession(c) // should be a no-op since no remember-me
		return c.String(http.StatusOK, "ok")
	})

	req, rec := jsonRequest(http.MethodGet, "/do-refresh-no-remember", nil)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDoRefreshSession_EligibleForRefresh(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	origMaxDuration := util.SessionMaxDuration
	util.SessionMaxDuration = 86400 * 90 // 90 days in seconds
	defer func() {
		util.DisableLogin = origDisable
		util.SessionMaxDuration = origMaxDuration
	}()

	env := setupTestEnv(t)
	util.DisableLogin = false

	// Step 1: Create a remember-me session with manipulated timestamps
	env.echo.GET("/create-old-session", func(c echo.Context) error {
		// Create a session with remember-me
		createSession(c, "admin", true, uint32(12345), true)

		// Now manipulate the session to be >24h old
		sess, _ := session.Get("session", c)
		now := time.Now().UTC().Unix()
		sess.Values["created_at"] = now - 172800 // created 2 days ago
		sess.Values["updated_at"] = now - 86401  // updated >24h ago
		sess.Save(c.Request(), c.Response())

		return c.String(http.StatusOK, "ok")
	})

	req1, rec1 := jsonRequest(http.MethodGet, "/create-old-session", nil)
	env.echo.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)

	// Step 2: Call doRefreshSession with the session cookies
	env.echo.GET("/trigger-refresh", func(c echo.Context) error {
		doRefreshSession(c)
		return c.String(http.StatusOK, "ok")
	})

	cookies := rec1.Result().Cookies()
	req2, rec2 := jsonRequest(http.MethodGet, "/trigger-refresh", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	env.echo.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)
}

// --- Integration tests: createSession + isValidSession ---

func TestCreateAndValidateSession(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	// Ensure the user CRC32 map is populated
	util.DBUsersToCRC32Mutex.Lock()
	util.DBUsersToCRC32["admin"] = uint32(12345)
	util.DBUsersToCRC32Mutex.Unlock()
	defer func() {
		util.DBUsersToCRC32Mutex.Lock()
		delete(util.DBUsersToCRC32, "admin")
		util.DBUsersToCRC32Mutex.Unlock()
	}()

	// Create a session
	env.echo.GET("/create", func(c echo.Context) error {
		createSession(c, "admin", true, uint32(12345), true)
		return c.String(http.StatusOK, "ok")
	})

	req1, rec1 := jsonRequest(http.MethodGet, "/create", nil)
	env.echo.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// Now validate the session with the cookies
	var valid bool
	env.echo.GET("/validate", func(c echo.Context) error {
		valid = isValidSession(c)
		return c.String(http.StatusOK, "ok")
	})

	cookies := rec1.Result().Cookies()
	req2, rec2 := jsonRequest(http.MethodGet, "/validate", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	env.echo.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)
	assert.True(t, valid, "Session should be valid after creation")
}

func TestIsValidSession_MismatchedCRC32(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	// Create session with one CRC32
	util.DBUsersToCRC32Mutex.Lock()
	util.DBUsersToCRC32["admin"] = uint32(12345)
	util.DBUsersToCRC32Mutex.Unlock()

	env.echo.GET("/create-mismatch", func(c echo.Context) error {
		createSession(c, "admin", true, uint32(12345), true)
		return c.String(http.StatusOK, "ok")
	})

	req1, rec1 := jsonRequest(http.MethodGet, "/create-mismatch", nil)
	env.echo.ServeHTTP(rec1, req1)

	// Now change the CRC32 in the map to simulate a user change
	util.DBUsersToCRC32Mutex.Lock()
	util.DBUsersToCRC32["admin"] = uint32(99999)
	util.DBUsersToCRC32Mutex.Unlock()
	defer func() {
		util.DBUsersToCRC32Mutex.Lock()
		delete(util.DBUsersToCRC32, "admin")
		util.DBUsersToCRC32Mutex.Unlock()
	}()

	var valid bool
	env.echo.GET("/validate-mismatch", func(c echo.Context) error {
		valid = isValidSession(c)
		return c.String(http.StatusOK, "ok")
	})

	cookies := rec1.Result().Cookies()
	req2, rec2 := jsonRequest(http.MethodGet, "/validate-mismatch", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	env.echo.ServeHTTP(rec2, req2)
	assert.False(t, valid, "Session should be invalid when CRC32 mismatches")
}

func TestValidSession_WithValidSession_PassesThrough(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	// Populate CRC32 map
	util.DBUsersToCRC32Mutex.Lock()
	util.DBUsersToCRC32["admin"] = uint32(12345)
	util.DBUsersToCRC32Mutex.Unlock()
	defer func() {
		util.DBUsersToCRC32Mutex.Lock()
		delete(util.DBUsersToCRC32, "admin")
		util.DBUsersToCRC32Mutex.Unlock()
	}()

	// Create a session
	env.echo.GET("/make-session", func(c echo.Context) error {
		createSession(c, "admin", true, uint32(12345), true)
		return c.String(http.StatusOK, "ok")
	})

	req1, rec1 := jsonRequest(http.MethodGet, "/make-session", nil)
	env.echo.ServeHTTP(rec1, req1)

	// Now access a ValidSession-protected route
	called := false
	env.echo.GET("/protected-route", ValidSession(func(c echo.Context) error {
		called = true
		return c.String(http.StatusOK, "protected")
	}))

	cookies := rec1.Result().Cookies()
	req2, rec2 := jsonRequest(http.MethodGet, "/protected-route", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	env.echo.ServeHTTP(rec2, req2)
	assert.True(t, called, "ValidSession should pass through for valid session")
	assert.Equal(t, http.StatusOK, rec2.Code)
}

func TestDoRefreshSession_WithSession_NoRefreshNeeded(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	// Create a remember-me session
	env.echo.GET("/create-for-refresh", func(c echo.Context) error {
		createSession(c, "admin", true, uint32(12345), true)
		return c.String(http.StatusOK, "ok")
	})

	req1, rec1 := jsonRequest(http.MethodGet, "/create-for-refresh", nil)
	env.echo.ServeHTTP(rec1, req1)

	// Try to refresh immediately (should not refresh since <24h since creation)
	env.echo.GET("/do-refresh-with-session", func(c echo.Context) error {
		doRefreshSession(c)
		return c.String(http.StatusOK, "ok")
	})

	cookies := rec1.Result().Cookies()
	req2, rec2 := jsonRequest(http.MethodGet, "/do-refresh-with-session", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	env.echo.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)
}

func TestValidSession_POST_NoNextURL(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	env.echo.POST("/protected-post", ValidSession(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}))

	req, rec := jsonRequest(http.MethodPost, "/protected-post", nil)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
	// POST redirect should go to /login without ?next= parameter
	location := rec.Header().Get("Location")
	assert.Contains(t, location, "/login")
}

func TestNeedsAdmin_WithAdminSession(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	// Create an admin session
	env.echo.GET("/setup-admin", func(c echo.Context) error {
		createSession(c, "admin", true, uint32(12345), false)
		return c.String(http.StatusOK, "ok")
	})

	req1, rec1 := jsonRequest(http.MethodGet, "/setup-admin", nil)
	env.echo.ServeHTTP(rec1, req1)

	// Access admin-only route
	called := false
	env.echo.GET("/admin-page", NeedsAdmin(func(c echo.Context) error {
		called = true
		return c.String(http.StatusOK, "admin page")
	}))

	cookies := rec1.Result().Cookies()
	req2, rec2 := jsonRequest(http.MethodGet, "/admin-page", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	env.echo.ServeHTTP(rec2, req2)
	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec2.Code)
}

func TestCurrentUser_WithRealSession(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	// Create a session first
	env.echo.GET("/setup-user", func(c echo.Context) error {
		createSession(c, "testuser", false, uint32(111), false)
		return c.String(http.StatusOK, "ok")
	})

	req1, rec1 := jsonRequest(http.MethodGet, "/setup-user", nil)
	env.echo.ServeHTTP(rec1, req1)

	// Read the current user
	var username string
	env.echo.GET("/read-user", func(c echo.Context) error {
		username = currentUser(c)
		return c.String(http.StatusOK, username)
	})

	cookies := rec1.Result().Cookies()
	req2, rec2 := jsonRequest(http.MethodGet, "/read-user", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	env.echo.ServeHTTP(rec2, req2)
	assert.Equal(t, "testuser", username)
}

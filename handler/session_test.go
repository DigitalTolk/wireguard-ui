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
		createSession(c, "testuser", true, uint32(12345))
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

func TestCreateSession_MaxAgeFromSessionMaxDuration(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	origMaxDuration := util.SessionMaxDuration
	util.SessionMaxDuration = 86400 * 7 // 7 days
	defer func() {
		util.DisableLogin = origDisable
		util.SessionMaxDuration = origMaxDuration
	}()

	env := setupTestEnv(t)
	util.DisableLogin = false

	env.echo.GET("/create-session-maxage", func(c echo.Context) error {
		createSession(c, "testuser", false, uint32(999))
		return c.String(http.StatusOK, "ok")
	})

	req, rec := jsonRequest(http.MethodGet, "/create-session-maxage", nil)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify session_token cookie has MaxAge from SessionMaxDuration
	cookies := rec.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "session_token" {
			assert.Equal(t, int(util.SessionMaxDuration), cookie.MaxAge,
				"session cookie should have MaxAge equal to SessionMaxDuration")
			break
		}
	}
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
		createSession(c, "admin", true, uint32(12345))
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
		createSession(c, "admin", true, uint32(12345))
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

// --- createSession with custom SessionMaxDuration ---

func TestCreateSession_WithSessionMaxDuration(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	origMaxDuration := util.SessionMaxDuration
	util.SessionMaxDuration = 86400 * 30 // 30 days
	defer func() {
		util.DisableLogin = origDisable
		util.SessionMaxDuration = origMaxDuration
	}()

	env := setupTestEnv(t)
	util.DisableLogin = false

	env.echo.GET("/create-with-duration", func(c echo.Context) error {
		createSession(c, "duruser", true, uint32(44444))
		return c.String(http.StatusOK, "ok")
	})

	req, rec := jsonRequest(http.MethodGet, "/create-with-duration", nil)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify cookie MaxAge is set to SessionMaxDuration
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "session_token" {
			assert.Equal(t, int(util.SessionMaxDuration), cookie.MaxAge,
				"Cookie MaxAge should equal SessionMaxDuration")
			break
		}
	}
}

// --- isAdmin with non-admin session ---

func TestIsAdmin_WithNonAdminSession(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	env.echo.GET("/setup-nonadmin", func(c echo.Context) error {
		createSession(c, "regular", false, uint32(55555))
		return c.String(http.StatusOK, "ok")
	})

	req1, rec1 := jsonRequest(http.MethodGet, "/setup-nonadmin", nil)
	env.echo.ServeHTTP(rec1, req1)

	var adminResult bool
	env.echo.GET("/check-admin", func(c echo.Context) error {
		adminResult = isAdmin(c)
		return c.String(http.StatusOK, "ok")
	})

	cookies := rec1.Result().Cookies()
	req2, rec2 := jsonRequest(http.MethodGet, "/check-admin", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	env.echo.ServeHTTP(rec2, req2)
	assert.False(t, adminResult, "Non-admin session should return false for isAdmin")
}

// --- isAdmin with admin session ---

func TestIsAdmin_WithAdminSession(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	env.echo.GET("/setup-admin-check", func(c echo.Context) error {
		createSession(c, "adminuser", true, uint32(66666))
		return c.String(http.StatusOK, "ok")
	})

	req1, rec1 := jsonRequest(http.MethodGet, "/setup-admin-check", nil)
	env.echo.ServeHTTP(rec1, req1)

	var adminResult bool
	env.echo.GET("/check-admin2", func(c echo.Context) error {
		adminResult = isAdmin(c)
		return c.String(http.StatusOK, "ok")
	})

	cookies := rec1.Result().Cookies()
	req2, rec2 := jsonRequest(http.MethodGet, "/check-admin2", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	env.echo.ServeHTTP(rec2, req2)
	assert.True(t, adminResult, "Admin session should return true for isAdmin")
}

// --- isValidSession: expired time bounds ---

func TestIsValidSession_ExpiredTimeBounds(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	origMaxDuration := util.SessionMaxDuration
	util.SessionMaxDuration = 100 // 100 seconds (short, so createdAt + 100 < now easily)
	defer func() {
		util.DisableLogin = origDisable
		util.SessionMaxDuration = origMaxDuration
	}()

	env := setupTestEnv(t)
	util.DisableLogin = false

	// Create a session with timestamps that will fail time bounds
	env.echo.GET("/create-expired-session", func(c echo.Context) error {
		createSession(c, "timeuser", true, uint32(77777))

		// Manipulate: created 200s ago (past max duration of 100s)
		sess, _ := session.Get("session", c)
		now := time.Now().UTC().Unix()
		sess.Values["created_at"] = now - 200 // past max duration
		sess.Values["updated_at"] = now - 50  // recent enough that expiration > now
		sess.Save(c.Request(), c.Response())

		return c.String(http.StatusOK, "ok")
	})

	req1, rec1 := jsonRequest(http.MethodGet, "/create-expired-session", nil)
	env.echo.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)

	// Deduplicate cookies (keep last for each name)
	allCookies := rec1.Result().Cookies()
	lastCookie := make(map[string]*http.Cookie)
	for _, c := range allCookies {
		lastCookie[c.Name] = c
	}

	var valid bool
	env.echo.GET("/validate-expired-bounds", func(c echo.Context) error {
		valid = isValidSession(c)
		return c.String(http.StatusOK, "ok")
	})

	req2, rec2 := jsonRequest(http.MethodGet, "/validate-expired-bounds", nil)
	for _, cookie := range lastCookie {
		req2.AddCookie(cookie)
	}
	env.echo.ServeHTTP(rec2, req2)
	assert.False(t, valid, "Session should be invalid when time bounds are exceeded")
}

// --- isValidSession: user not in CRC32 map ---

func TestIsValidSession_UserRemovedFromDB(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	util.DBUsersToCRC32Mutex.Lock()
	util.DBUsersToCRC32["tempuser"] = uint32(54321)
	util.DBUsersToCRC32Mutex.Unlock()

	env.echo.GET("/create-temp-session", func(c echo.Context) error {
		createSession(c, "tempuser", false, uint32(54321))
		return c.String(http.StatusOK, "ok")
	})

	req1, rec1 := jsonRequest(http.MethodGet, "/create-temp-session", nil)
	env.echo.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)

	// Remove user from CRC32 map (simulates user deletion)
	util.DBUsersToCRC32Mutex.Lock()
	delete(util.DBUsersToCRC32, "tempuser")
	util.DBUsersToCRC32Mutex.Unlock()

	var valid bool
	env.echo.GET("/validate-removed-user", func(c echo.Context) error {
		valid = isValidSession(c)
		return c.String(http.StatusOK, "ok")
	})

	cookies := rec1.Result().Cookies()
	req2, rec2 := jsonRequest(http.MethodGet, "/validate-removed-user", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	env.echo.ServeHTTP(rec2, req2)
	assert.False(t, valid, "Session should be invalid when user is removed from DB")
}

// --- isValidSession: temporary session (maxAge=0) within 24h ---

func TestIsValidSession_TemporarySession(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	util.DBUsersToCRC32Mutex.Lock()
	util.DBUsersToCRC32["tempsess"] = uint32(11111)
	util.DBUsersToCRC32Mutex.Unlock()
	defer func() {
		util.DBUsersToCRC32Mutex.Lock()
		delete(util.DBUsersToCRC32, "tempsess")
		util.DBUsersToCRC32Mutex.Unlock()
	}()

	// Create session without remember-me (maxAge=0)
	env.echo.GET("/create-temp-sess", func(c echo.Context) error {
		createSession(c, "tempsess", false, uint32(11111))
		return c.String(http.StatusOK, "ok")
	})

	req1, rec1 := jsonRequest(http.MethodGet, "/create-temp-sess", nil)
	env.echo.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)

	var valid bool
	env.echo.GET("/validate-temp-sess", func(c echo.Context) error {
		valid = isValidSession(c)
		return c.String(http.StatusOK, "ok")
	})

	cookies := rec1.Result().Cookies()
	req2, rec2 := jsonRequest(http.MethodGet, "/validate-temp-sess", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	env.echo.ServeHTTP(rec2, req2)
	assert.True(t, valid, "Temporary session should be valid within 24h virtual expiration")
}

// --- clearSession clears a valid session ---

func TestClearSession_ThenInvalid(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	util.DBUsersToCRC32Mutex.Lock()
	util.DBUsersToCRC32["clearme"] = uint32(22222)
	util.DBUsersToCRC32Mutex.Unlock()
	defer func() {
		util.DBUsersToCRC32Mutex.Lock()
		delete(util.DBUsersToCRC32, "clearme")
		util.DBUsersToCRC32Mutex.Unlock()
	}()

	env.echo.GET("/create-clear-session", func(c echo.Context) error {
		createSession(c, "clearme", true, uint32(22222))
		return c.String(http.StatusOK, "ok")
	})

	req1, rec1 := jsonRequest(http.MethodGet, "/create-clear-session", nil)
	env.echo.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)
	cookies := rec1.Result().Cookies()

	// Now clear it
	env.echo.GET("/do-clear-session", func(c echo.Context) error {
		clearSession(c)
		return c.String(http.StatusOK, "ok")
	})

	req2, rec2 := jsonRequest(http.MethodGet, "/do-clear-session", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	env.echo.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)

	// After clearing, the session_token cookie should have MaxAge=-1
	for _, cookie := range rec2.Result().Cookies() {
		if cookie.Name == "session_token" {
			assert.Equal(t, -1, cookie.MaxAge, "session_token should be expired after clear")
		}
	}
}

func TestCurrentUser_WithRealSession(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	// Create a session first
	env.echo.GET("/setup-user", func(c echo.Context) error {
		createSession(c, "testuser", false, uint32(111))
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

package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DigitalTolk/wireguard-ui/model"
	"github.com/DigitalTolk/wireguard-ui/util"
)

func TestAPIListUsers(t *testing.T) {
	env := setupTestEnv(t)

	req, rec := jsonRequest(http.MethodGet, "/api/v1/users", nil)
	c := env.echo.NewContext(req, rec)
	err := APIListUsers(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPICreateUser(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]interface{}{
		"username": "newuser",
		"admin":    false,
		"email":    "new@test.com",
	}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/users", body)
	c := env.echo.NewContext(req, rec)
	err := APICreateUser(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var user model.User
	parseJSON(t, rec, &user)
	assert.Equal(t, "newuser", user.Username)
	assert.Equal(t, "new@test.com", user.Email)
}

func TestAPICreateUser_DuplicateUsername(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	env.db.SaveUser(model.User{Username: "existing", CreatedAt: now, UpdatedAt: now})

	body := map[string]interface{}{"username": "existing", "admin": false}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/users", body)
	c := env.echo.NewContext(req, rec)
	err := APICreateUser(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPICreateUser_InvalidUsername(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]interface{}{"username": "", "admin": false}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/users", body)
	c := env.echo.NewContext(req, rec)
	err := APICreateUser(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIGetUser(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	env.db.SaveUser(model.User{Username: "getme", Email: "get@test.com", CreatedAt: now, UpdatedAt: now})

	req, rec := jsonRequest(http.MethodGet, "/api/v1/users/getme", nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("username")
	c.SetParamValues("getme")
	err := APIGetUser(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPIGetUser_NotFound(t *testing.T) {
	env := setupTestEnv(t)

	req, rec := jsonRequest(http.MethodGet, "/api/v1/users/nonexistent", nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("username")
	c.SetParamValues("nonexistent")
	err := APIGetUser(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAPIDeleteUser(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	env.db.SaveUser(model.User{Username: "delme", CreatedAt: now, UpdatedAt: now})

	req, rec := jsonRequest(http.MethodDelete, "/api/v1/users/delme", nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("username")
	c.SetParamValues("delme")
	err := APIDeleteUser(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestAPIDeleteUser_InvalidUsername(t *testing.T) {
	env := setupTestEnv(t)

	req, rec := jsonRequest(http.MethodDelete, "/api/v1/users/bad-user", nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("username")
	c.SetParamValues("bad user!")
	err := APIDeleteUser(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- APIUpdateUser Tests ---

func TestAPIUpdateUser_Success(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	env.db.SaveUser(model.User{Username: "updateme", Email: "old@test.com", CreatedAt: now, UpdatedAt: now})

	body := map[string]interface{}{
		"email":        "new@test.com",
		"display_name": "Updated Name",
	}
	req, rec := jsonRequest(http.MethodPut, "/api/v1/users/updateme", body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("username")
	c.SetParamValues("updateme")
	err := APIUpdateUser(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var user model.User
	parseJSON(t, rec, &user)
	assert.Equal(t, "new@test.com", user.Email)
	assert.Equal(t, "Updated Name", user.DisplayName)
}

func TestAPIUpdateUser_InvalidUsername(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]interface{}{"email": "test@test.com"}
	req, rec := jsonRequest(http.MethodPut, "/api/v1/users/baduser", body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("username")
	c.SetParamValues("bad user!")
	err := APIUpdateUser(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIUpdateUser_NotFound(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]interface{}{"email": "test@test.com"}
	req, rec := jsonRequest(http.MethodPut, "/api/v1/users/nonexistent", body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("username")
	c.SetParamValues("nonexistent")
	err := APIUpdateUser(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAPIGetUser_InvalidUsername(t *testing.T) {
	env := setupTestEnv(t)

	req, rec := jsonRequest(http.MethodGet, "/api/v1/users/baduser", nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("username")
	c.SetParamValues("bad user!")
	err := APIGetUser(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIGetUser_NonAdminAccessingOther(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false
	now := time.Now().UTC()
	env.db.SaveUser(model.User{Username: "otheruser", Email: "other@test.com", CreatedAt: now, UpdatedAt: now})

	// Register all routes at once
	env.echo.GET("/setup-nonadmin", func(c echo.Context) error {
		createSession(c, "nonadmin", false, uint32(111), false)
		return c.String(http.StatusOK, "ok")
	})
	env.echo.GET("/users/:username", APIGetUser(env.db))

	req1, rec1 := jsonRequest(http.MethodGet, "/setup-nonadmin", nil)
	env.echo.ServeHTTP(rec1, req1)

	cookies := rec1.Result().Cookies()
	req2, rec2 := jsonRequest(http.MethodGet, "/users/otheruser", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	env.echo.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusForbidden, rec2.Code)
}

func TestAPIUpdateUser_NonAdminUpdatingOther(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false
	now := time.Now().UTC()
	env.db.SaveUser(model.User{Username: "target", Email: "target@test.com", CreatedAt: now, UpdatedAt: now})

	env.echo.GET("/setup-nonadmin2", func(c echo.Context) error {
		createSession(c, "nonadmin", false, uint32(222), false)
		return c.String(http.StatusOK, "ok")
	})
	env.echo.PUT("/users/:username", APIUpdateUser(env.db))

	req1, rec1 := jsonRequest(http.MethodGet, "/setup-nonadmin2", nil)
	env.echo.ServeHTTP(rec1, req1)

	cookies := rec1.Result().Cookies()
	req2, rec2 := jsonRequest(http.MethodPut, "/users/target", map[string]interface{}{"email": "hacked@test.com"})
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	env.echo.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusForbidden, rec2.Code)
}

func TestAPIUpdateUser_AdminCanChangeAdminStatus(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	env.db.SaveUser(model.User{Username: "target", Admin: false, CreatedAt: now, UpdatedAt: now})

	admin := true
	body := map[string]interface{}{
		"admin": admin,
	}
	req, rec := jsonRequest(http.MethodPut, "/api/v1/users/target", body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("username")
	c.SetParamValues("target")
	err := APIUpdateUser(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var user model.User
	parseJSON(t, rec, &user)
	assert.True(t, user.Admin)
}

func TestAPIDeleteUser_Self(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false
	now := time.Now().UTC()
	env.db.SaveUser(model.User{Username: "selfdelete", CreatedAt: now, UpdatedAt: now})

	env.echo.GET("/setup-self", func(c echo.Context) error {
		createSession(c, "selfdelete", false, uint32(333), false)
		return c.String(http.StatusOK, "ok")
	})
	env.echo.DELETE("/users/:username", APIDeleteUser(env.db))

	req1, rec1 := jsonRequest(http.MethodGet, "/setup-self", nil)
	env.echo.ServeHTTP(rec1, req1)

	cookies := rec1.Result().Cookies()
	req2, rec2 := jsonRequest(http.MethodDelete, "/users/selfdelete", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	env.echo.ServeHTTP(rec2, req2)
	// Should be blocked from deleting self
	assert.Equal(t, http.StatusBadRequest, rec2.Code)
}

func TestAPIUpdateUser_SelfUpdate(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false
	now := time.Now().UTC()
	env.db.SaveUser(model.User{Username: "selfupdate", Email: "old@test.com", CreatedAt: now, UpdatedAt: now})

	env.echo.GET("/setup-self-update", func(c echo.Context) error {
		createSession(c, "selfupdate", false, uint32(444), false)
		return c.String(http.StatusOK, "ok")
	})
	env.echo.PUT("/users/:username", APIUpdateUser(env.db))

	req1, rec1 := jsonRequest(http.MethodGet, "/setup-self-update", nil)
	env.echo.ServeHTTP(rec1, req1)

	cookies := rec1.Result().Cookies()
	req2, rec2 := jsonRequest(http.MethodPut, "/users/selfupdate", map[string]interface{}{"email": "new@test.com"})
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	env.echo.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)
}

func TestAPICreateUser_InvalidBody(t *testing.T) {
	env := setupTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader("{invalid"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := env.echo.NewContext(req, rec)
	err := APICreateUser(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIUpdateUser_InvalidBody(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()
	env.db.SaveUser(model.User{Username: "bodytest", CreatedAt: now, UpdatedAt: now})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/bodytest", strings.NewReader("{invalid"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("username")
	c.SetParamValues("bodytest")
	err := APIUpdateUser(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

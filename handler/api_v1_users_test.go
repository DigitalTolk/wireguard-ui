package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo-contrib/session"
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

func TestAPIListUsers_WithPopulatedData(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	env.db.SaveUser(model.User{Username: "user1", Email: "user1@test.com", Admin: true, OIDCSub: "sub-1", CreatedAt: now, UpdatedAt: now})
	env.db.SaveUser(model.User{Username: "user2", Email: "user2@test.com", Admin: false, OIDCSub: "sub-2", CreatedAt: now, UpdatedAt: now})
	env.db.SaveUser(model.User{Username: "user3", Email: "user3@test.com", Admin: false, OIDCSub: "sub-3", CreatedAt: now, UpdatedAt: now})

	req, rec := jsonRequest(http.MethodGet, "/api/v1/users", nil)
	c := env.echo.NewContext(req, rec)
	err := APIListUsers(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var users []model.User
	parseJSON(t, rec, &users)
	assert.Len(t, users, 3)
}

func TestAPIGetUser_WithFullData(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	env.db.SaveUser(model.User{
		Username:    "fulluser",
		Email:       "full@test.com",
		DisplayName: "Full User",
		OIDCSub:     "sub-full",
		Admin:       true,
		CreatedAt:   now,
		UpdatedAt:   now,
	})

	req, rec := jsonRequest(http.MethodGet, "/api/v1/users/fulluser", nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("username")
	c.SetParamValues("fulluser")
	err := APIGetUser(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var user model.User
	parseJSON(t, rec, &user)
	assert.Equal(t, "fulluser", user.Username)
	assert.Equal(t, "full@test.com", user.Email)
	assert.Equal(t, "Full User", user.DisplayName)
	assert.True(t, user.Admin)
}

func TestAPIListUsers_DBError(t *testing.T) {
	db := &errStore{}
	e := echo.New()

	req, rec := jsonRequest(http.MethodGet, "/api/v1/users", nil)
	c := e.NewContext(req, rec)
	err := APIListUsers(db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
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

// --- APIPatchUserAdmin tests ---

func TestAPIPatchUserAdmin_PromoteSuccess(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	env.db.SaveUser(model.User{Username: "patchuser", Email: "patch@test.com", Admin: false, OIDCSub: "sub-patch", CreatedAt: now, UpdatedAt: now})

	body := map[string]interface{}{"admin": true}
	req, rec := jsonRequest(http.MethodPatch, "/api/v1/users/patchuser/admin", body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("username")
	c.SetParamValues("patchuser")

	err := APIPatchUserAdmin(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result model.User
	parseJSON(t, rec, &result)
	assert.True(t, result.Admin)
	assert.Equal(t, "patchuser", result.Username)

	// Verify CRC32 cache was updated
	util.DBUsersToCRC32Mutex.RLock()
	_, ok := util.DBUsersToCRC32["patchuser"]
	util.DBUsersToCRC32Mutex.RUnlock()
	assert.True(t, ok, "CRC32 cache should be updated after admin change")
}

func TestAPIPatchUserAdmin_DemoteSuccess(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	// need at least 2 admins so demoting one is allowed
	env.db.SaveUser(model.User{Username: "demoteuser", Email: "demote@test.com", Admin: true, OIDCSub: "sub-demote", CreatedAt: now, UpdatedAt: now})
	env.db.SaveUser(model.User{Username: "otheradmin", Email: "other@test.com", Admin: true, OIDCSub: "sub-other", CreatedAt: now, UpdatedAt: now})

	body := map[string]interface{}{"admin": false}
	req, rec := jsonRequest(http.MethodPatch, "/api/v1/users/demoteuser/admin", body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("username")
	c.SetParamValues("demoteuser")

	err := APIPatchUserAdmin(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result model.User
	parseJSON(t, rec, &result)
	assert.False(t, result.Admin)
}

func TestAPIPatchUserAdmin_InvalidUsername(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]interface{}{"admin": true}
	req, rec := jsonRequest(http.MethodPatch, "/api/v1/users/baduser/admin", body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("username")
	c.SetParamValues("bad user!")

	err := APIPatchUserAdmin(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIPatchUserAdmin_UserNotFound(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]interface{}{"admin": true}
	req, rec := jsonRequest(http.MethodPatch, "/api/v1/users/nonexistent/admin", body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("username")
	c.SetParamValues("nonexistent")

	err := APIPatchUserAdmin(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAPIPatchUserAdmin_InvalidBody(t *testing.T) {
	env := setupTestEnv(t)

	// Send a request with invalid JSON body
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/someuser/admin", strings.NewReader("{invalid"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("username")
	c.SetParamValues("someuser")

	err := APIPatchUserAdmin(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIPatchUserAdmin_DBSaveError(t *testing.T) {
	db := &errStore{}
	e := echo.New()

	body := map[string]interface{}{"admin": true}
	req, rec := jsonRequest(http.MethodPatch, "/api/v1/users/anyuser/admin", body)
	c := e.NewContext(req, rec)
	c.SetParamNames("username")
	c.SetParamValues("anyuser")

	err := APIPatchUserAdmin(db)(c)
	require.NoError(t, err)
	// errStore.GetUserByName returns error, so we get 404
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAPIPatchUserAdmin_SaveUserFails(t *testing.T) {
	// Use saveFailStore: GetUserByName succeeds, SaveUser fails
	now := time.Now().UTC()
	db := &saveFailStore{
		user: model.User{Username: "saveuser", Email: "save@test.com", Admin: false, CreatedAt: now, UpdatedAt: now},
	}
	e := echo.New()

	body := map[string]interface{}{"admin": true}
	req, rec := jsonRequest(http.MethodPatch, "/api/v1/users/saveuser/admin", body)
	c := e.NewContext(req, rec)
	c.SetParamNames("username")
	c.SetParamValues("saveuser")

	err := APIPatchUserAdmin(db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// Regression: an admin must not be able to demote themselves
func TestAPIPatchUserAdmin_CannotDemoteSelf(t *testing.T) {
	env := setupTestEnv(t)

	// disable the DisableLogin bypass so currentUser reads from session
	origDisableLogin := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisableLogin }()

	now := time.Now().UTC()
	env.db.SaveUser(model.User{
		Username: "selfadmin", Email: "self@test.com", Admin: true,
		OIDCSub: "sub-self", CreatedAt: now, UpdatedAt: now,
	})
	env.db.SaveUser(model.User{
		Username: "otheradmin", Email: "other@test.com", Admin: true,
		OIDCSub: "sub-other", CreatedAt: now, UpdatedAt: now,
	})

	body := map[string]interface{}{"admin": false}
	req, rec := jsonRequest(http.MethodPatch, "/api/v1/users/selfadmin/admin", body)

	// set up a real session via the Echo router so the session middleware runs
	env.echo.PATCH("/api/v1/users/:username/admin", func(c echo.Context) error {
		// write session first, then call the handler
		sess, _ := session.Get("session", c)
		sess.Values["username"] = "selfadmin"
		sess.Values["admin"] = true
		sess.Values["session_token"] = "tok"
		sess.Save(c.Request(), c.Response())
		req.AddCookie(&http.Cookie{Name: "session_token", Value: "tok"})
		return APIPatchUserAdmin(env.db)(c)
	})
	env.echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Cannot remove your own admin role")

	// verify user is still admin in DB
	user, _ := env.db.GetUserByName("selfadmin")
	assert.True(t, user.Admin)
}

// Regression: cannot demote the last remaining admin
func TestAPIPatchUserAdmin_CannotDemoteLastAdmin(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	// only one admin exists
	env.db.SaveUser(model.User{
		Username: "onlyadmin", Email: "only@test.com", Admin: true,
		OIDCSub: "sub-only", CreatedAt: now, UpdatedAt: now,
	})

	body := map[string]interface{}{"admin": false}
	req, rec := jsonRequest(http.MethodPatch, "/api/v1/users/onlyadmin/admin", body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("username")
	c.SetParamValues("onlyadmin")

	err := APIPatchUserAdmin(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Cannot remove the last admin")

	// verify still admin
	user, _ := env.db.GetUserByName("onlyadmin")
	assert.True(t, user.Admin)
}

// Promoting a non-admin when you're the only admin should work fine
func TestAPIPatchUserAdmin_PromoteWhenOnlyAdmin(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	env.db.SaveUser(model.User{
		Username: "admin1", Email: "a1@test.com", Admin: true,
		OIDCSub: "sub-a1", CreatedAt: now, UpdatedAt: now,
	})
	env.db.SaveUser(model.User{
		Username: "regular", Email: "reg@test.com", Admin: false,
		OIDCSub: "sub-reg", CreatedAt: now, UpdatedAt: now,
	})

	body := map[string]interface{}{"admin": true}
	req, rec := jsonRequest(http.MethodPatch, "/api/v1/users/regular/admin", body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("username")
	c.SetParamValues("regular")

	err := APIPatchUserAdmin(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	user, _ := env.db.GetUserByName("regular")
	assert.True(t, user.Admin)
}

// Demoting when 2 admins exist should succeed
func TestAPIPatchUserAdmin_DemoteWithMultipleAdmins(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	env.db.SaveUser(model.User{
		Username: "admin1", Email: "a1@test.com", Admin: true,
		OIDCSub: "sub-a1", CreatedAt: now, UpdatedAt: now,
	})
	env.db.SaveUser(model.User{
		Username: "admin2", Email: "a2@test.com", Admin: true,
		OIDCSub: "sub-a2", CreatedAt: now, UpdatedAt: now,
	})

	body := map[string]interface{}{"admin": false}
	req, rec := jsonRequest(http.MethodPatch, "/api/v1/users/admin2/admin", body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("username")
	c.SetParamValues("admin2")

	err := APIPatchUserAdmin(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	user, _ := env.db.GetUserByName("admin2")
	assert.False(t, user.Admin)
}

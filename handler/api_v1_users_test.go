package handler

import (
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DigitalTolk/wireguard-ui/model"
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

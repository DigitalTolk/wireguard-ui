package handler

import (
	"net/http"
	"testing"
	"time"

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

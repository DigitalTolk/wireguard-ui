package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContentTypeJson_ValidContentType(t *testing.T) {
	e := echo.New()
	called := false
	handler := ContentTypeJson(func(c echo.Context) error {
		called = true
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestContentTypeJson_ValidContentTypeWithCharset(t *testing.T) {
	e := echo.New()
	called := false
	handler := ContentTypeJson(func(c echo.Context) error {
		called = true
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestContentTypeJson_InvalidContentType(t *testing.T) {
	e := echo.New()
	called := false
	handler := ContentTypeJson(func(c echo.Context) error {
		called = true
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("data"))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	require.NoError(t, err)
	assert.False(t, called)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var result map[string]interface{}
	parseJSON(t, rec, &result)
	errObj := result["error"].(map[string]interface{})
	assert.Equal(t, "INVALID_CONTENT_TYPE", errObj["code"])
}

func TestContentTypeJson_EmptyContentType(t *testing.T) {
	e := echo.New()
	called := false
	handler := ContentTypeJson(func(c echo.Context) error {
		called = true
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("data"))
	// No Content-Type header set
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	require.NoError(t, err)
	assert.False(t, called)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestContentTypeJson_FormUrlEncoded(t *testing.T) {
	e := echo.New()
	called := false
	handler := ContentTypeJson(func(c echo.Context) error {
		called = true
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("key=value"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	require.NoError(t, err)
	assert.False(t, called)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

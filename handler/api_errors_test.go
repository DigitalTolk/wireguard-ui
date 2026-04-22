package handler

import (
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApiErrorResponses(t *testing.T) {
	tests := []struct {
		name   string
		fn     func(echo.Context, string) error
		status int
		code   string
	}{
		{"bad request", apiBadRequest, http.StatusBadRequest, "BAD_REQUEST"},
		{"not found", apiNotFound, http.StatusNotFound, "NOT_FOUND"},
		{"internal error", apiInternalError, http.StatusInternalServerError, "INTERNAL_ERROR"},
		{"forbidden", apiForbidden, http.StatusForbidden, "FORBIDDEN"},
		{"unauthorized", apiUnauthorized, http.StatusUnauthorized, "UNAUTHORIZED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, rec := jsonRequest(http.MethodGet, "/test", nil)
			e := echo.New()
			c := e.NewContext(req, rec)
			err := tt.fn(c, "test message")
			require.NoError(t, err)
			assert.Equal(t, tt.status, rec.Code)

			var result APIError
			parseJSON(t, rec, &result)
			assert.Equal(t, tt.code, result.Error.Code)
			assert.Equal(t, "test message", result.Error.Message)
		})
	}
}

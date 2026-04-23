package handler

import (
	"net/http"
	"regexp"

	"github.com/labstack/echo/v4"
)

var usernameRegexp = regexp.MustCompile(`^\w[\w\-.@]*$`)

// APIError is the standard error response for API v1 endpoints
type APIError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func apiError(c echo.Context, status int, code, message string) error {
	resp := APIError{}
	resp.Error.Code = code
	resp.Error.Message = message
	return c.JSON(status, resp)
}

func apiBadRequest(c echo.Context, message string) error {
	return apiError(c, http.StatusBadRequest, "BAD_REQUEST", message)
}

func apiNotFound(c echo.Context, message string) error {
	return apiError(c, http.StatusNotFound, "NOT_FOUND", message)
}

func apiInternalError(c echo.Context, message string) error {
	return apiError(c, http.StatusInternalServerError, "INTERNAL_ERROR", message)
}

func apiForbidden(c echo.Context, message string) error {
	return apiError(c, http.StatusForbidden, "FORBIDDEN", message)
}

func apiUnauthorized(c echo.Context, message string) error {
	return apiError(c, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

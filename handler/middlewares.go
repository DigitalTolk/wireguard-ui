package handler

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// ContentTypeJson middleware checks that the request Content-Type is application/json.
// This mitigates CSRF attacks since browsers don't allow setting Content-Type on cross-origin requests.
func ContentTypeJson(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		contentType := c.Request().Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "application/json") {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": map[string]string{
					"code":    "INVALID_CONTENT_TYPE",
					"message": "Content-Type must be application/json",
				},
			})
		}
		return next(c)
	}
}

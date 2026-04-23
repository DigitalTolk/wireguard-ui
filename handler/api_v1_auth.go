package handler

import (
	"net/http"
	"os"

	"github.com/labstack/echo/v4"

	"github.com/DigitalTolk/wireguard-ui/store"
	"github.com/DigitalTolk/wireguard-ui/util"
)

// APIAuth middleware validates session for API endpoints (returns JSON 401 instead of redirect)
func APIAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if util.DisableLogin {
			return next(c)
		}
		if !isValidSession(c) {
			return apiUnauthorized(c, "Not authenticated")
		}
		return next(c)
	}
}

// APIAdmin middleware checks admin status for API endpoints
func APIAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !isAdmin(c) {
			return apiForbidden(c, "Admin access required")
		}
		return next(c)
	}
}

// APIGetMe returns the current authenticated user's info
func APIGetMe(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		if util.DisableLogin {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"username":     "admin",
				"email":        "",
				"display_name": "Admin",
				"admin":        true,
			})
		}

		username := currentUser(c)
		if username == "" {
			return apiUnauthorized(c, "Not authenticated")
		}

		user, err := db.GetUserByName(username)
		if err != nil {
			return apiInternalError(c, "Cannot find user")
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"username":     user.Username,
			"email":        user.Email,
			"display_name": user.DisplayName,
			"admin":        user.Admin,
		})
	}
}

// APILogout destroys the current session
func APILogout() echo.HandlerFunc {
	return func(c echo.Context) error {
		auditLogEvent(c, "user.logout", "user", "", nil)
		clearSession(c)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "Logged out successfully",
		})
	}
}

// Health returns a simple health check
func Health() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}
}

// Favicon serves the favicon
func Favicon() echo.HandlerFunc {
	return func(c echo.Context) error {
		if favicon, ok := os.LookupEnv(util.FaviconFilePathEnvVar); ok {
			return c.File(favicon)
		}
		return c.Redirect(http.StatusFound, util.BasePath+"/static/favicon.svg")
	}
}

// APIAppInfo returns app metadata for the frontend
func APIAppInfo(appVersion, gitCommit string) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"base_path":       util.BasePath,
			"app_version":     appVersion,
			"git_commit":      gitCommit,
			"client_defaults": util.ClientDefaultsFromEnv(),
		})
	}
}

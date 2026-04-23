package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/DigitalTolk/wireguard-ui/store"
)

// APIListUsers returns all users (read-only, managed via SSO)
func APIListUsers(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		users, err := db.GetUsers()
		if err != nil {
			return apiInternalError(c, "Cannot get user list")
		}
		return c.JSON(http.StatusOK, users)
	}
}

// APIGetUser returns a single user by username
func APIGetUser(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		username := c.Param("username")
		if !usernameRegexp.MatchString(username) {
			return apiBadRequest(c, "Invalid username")
		}

		user, err := db.GetUserByName(username)
		if err != nil {
			return apiNotFound(c, "User not found")
		}
		return c.JSON(http.StatusOK, user)
	}
}

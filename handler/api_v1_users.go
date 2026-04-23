package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"

	"github.com/DigitalTolk/wireguard-ui/store"
	"github.com/DigitalTolk/wireguard-ui/util"
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

// APIPatchUserAdmin toggles admin status for a user
func APIPatchUserAdmin(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		username := c.Param("username")
		if !usernameRegexp.MatchString(username) {
			return apiBadRequest(c, "Invalid username")
		}

		var body struct {
			Admin bool `json:"admin"`
		}
		if err := c.Bind(&body); err != nil {
			return apiBadRequest(c, "Invalid request body")
		}

		user, err := db.GetUserByName(username)
		if err != nil {
			return apiNotFound(c, "User not found")
		}

		user.Admin = body.Admin
		user.UpdatedAt = time.Now().UTC()
		if err := db.SaveUser(user); err != nil {
			return apiInternalError(c, "Cannot update user")
		}

		// update CRC32 cache so existing sessions reflect the change
		util.DBUsersToCRC32Mutex.Lock()
		util.DBUsersToCRC32[user.Username] = util.GetDBUserCRC32(user)
		util.DBUsersToCRC32Mutex.Unlock()

		action := "user.demote"
		if body.Admin {
			action = "user.promote"
		}
		log.Infof("Changed admin status for %s to %v", username, body.Admin)
		auditLogEvent(c, action, "user", username, nil)
		return c.JSON(http.StatusOK, user)
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

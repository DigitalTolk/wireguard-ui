package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"

	"github.com/DigitalTolk/wireguard-ui/model"
	"github.com/DigitalTolk/wireguard-ui/store"
	"github.com/DigitalTolk/wireguard-ui/util"
)

// APIListUsers returns all users
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

		// non-admins can only access their own data
		if !isAdmin(c) && username != currentUser(c) {
			return apiForbidden(c, "Cannot access other user data")
		}

		user, err := db.GetUserByName(username)
		if err != nil {
			return apiNotFound(c, "User not found")
		}
		return c.JSON(http.StatusOK, user)
	}
}

// APICreateUser creates a new user
func APICreateUser(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var body struct {
			Username string `json:"username"`
			Admin    bool   `json:"admin"`
			Email    string `json:"email"`
		}
		if err := c.Bind(&body); err != nil {
			return apiBadRequest(c, "Invalid request body")
		}

		if body.Username == "" || !usernameRegexp.MatchString(body.Username) {
			return apiBadRequest(c, "Invalid username")
		}

		// check if user exists
		if _, err := db.GetUserByName(body.Username); err == nil {
			return apiBadRequest(c, "Username already taken")
		}

		now := time.Now().UTC()
		user := model.User{
			Username:  body.Username,
			Email:     body.Email,
			Admin:     body.Admin,
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := db.SaveUser(user); err != nil {
			return apiInternalError(c, err.Error())
		}

		log.Infof("Created user: %s", user.Username)
		auditLogEvent(c, "user.create", "user", user.Username, map[string]interface{}{"admin": user.Admin})
		return c.JSON(http.StatusCreated, user)
	}
}

// APIUpdateUser updates an existing user
func APIUpdateUser(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		username := c.Param("username")
		if !usernameRegexp.MatchString(username) {
			return apiBadRequest(c, "Invalid username")
		}

		// non-admins can only update their own data
		if !isAdmin(c) && username != currentUser(c) {
			return apiForbidden(c, "Cannot update other user data")
		}

		user, err := db.GetUserByName(username)
		if err != nil {
			return apiNotFound(c, "User not found")
		}

		var body struct {
			Email       string `json:"email"`
			DisplayName string `json:"display_name"`
			Admin       *bool  `json:"admin"`
		}
		if err := c.Bind(&body); err != nil {
			return apiBadRequest(c, "Invalid request body")
		}

		if body.Email != "" {
			user.Email = body.Email
		}
		if body.DisplayName != "" {
			user.DisplayName = body.DisplayName
		}
		// only admins can change admin status, and not their own
		if body.Admin != nil && isAdmin(c) && username != currentUser(c) {
			user.Admin = *body.Admin
		}
		user.UpdatedAt = time.Now().UTC()

		if err := db.SaveUser(user); err != nil {
			return apiInternalError(c, err.Error())
		}

		// update session if the current user updated themselves
		if username == currentUser(c) {
			setUser(c, user.Username, user.Admin, util.GetDBUserCRC32(user))
		}

		log.Infof("Updated user: %s", user.Username)
		auditLogEvent(c, "user.update", "user", user.Username, nil)
		return c.JSON(http.StatusOK, user)
	}
}

// APIDeleteUser deletes a user
func APIDeleteUser(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		username := c.Param("username")
		if !usernameRegexp.MatchString(username) {
			return apiBadRequest(c, "Invalid username")
		}

		if username == currentUser(c) {
			return apiBadRequest(c, "Cannot delete yourself")
		}

		if err := db.DeleteUser(username); err != nil {
			return apiInternalError(c, "Cannot delete user")
		}

		log.Infof("Deleted user: %s", username)
		auditLogEvent(c, "user.delete", "user", username, nil)
		return c.NoContent(http.StatusNoContent)
	}
}

package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/rs/xid"
	"golang.org/x/oauth2"

	"github.com/DigitalTolk/wireguard-ui/model"
	"github.com/DigitalTolk/wireguard-ui/store"
	"github.com/DigitalTolk/wireguard-ui/util"
)

// OIDCProvider holds the OIDC provider and OAuth2 config
type OIDCProvider struct {
	provider    *oidc.Provider
	oauth2Cfg   oauth2.Config
	verifier    *oidc.IDTokenVerifier
	adminGroups []string
}

// NewOIDCProvider creates a new OIDC provider from configuration
func NewOIDCProvider() (*OIDCProvider, error) {
	if util.OIDCIssuerURL == "" || util.OIDCClientID == "" {
		return nil, nil // OIDC not configured
	}

	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, util.OIDCIssuerURL)
	if err != nil {
		return nil, fmt.Errorf("cannot create OIDC provider: %w", err)
	}

	scopes := util.OIDCScopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}

	oauth2Cfg := oauth2.Config{
		ClientID:     util.OIDCClientID,
		ClientSecret: util.OIDCClientSecret,
		RedirectURL:  util.OIDCRedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       scopes,
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: util.OIDCClientID})

	return &OIDCProvider{
		provider:    provider,
		oauth2Cfg:   oauth2Cfg,
		verifier:    verifier,
		adminGroups: util.OIDCAdminGroups,
	}, nil
}

// APIStartOIDCLogin initiates the OIDC authorization code flow
func APIStartOIDCLogin(oidcProvider *OIDCProvider) echo.HandlerFunc {
	return func(c echo.Context) error {
		if oidcProvider == nil {
			return apiInternalError(c, "OIDC is not configured")
		}

		state := xid.New().String()
		nonce := xid.New().String()

		// store state and nonce in session for validation in callback
		sess, _ := session.Get("session", c)
		sess.Options = &sessions.Options{
			Path:     util.GetCookiePath(),
			MaxAge:   300, // 5 minutes for login flow
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		}
		sess.Values["oidc_state"] = state
		sess.Values["oidc_nonce"] = nonce
		sess.Save(c.Request(), c.Response())

		authURL := oidcProvider.oauth2Cfg.AuthCodeURL(state, oidc.Nonce(nonce))
		return c.Redirect(http.StatusTemporaryRedirect, authURL)
	}
}

// APIHandleOIDCCallback handles the OIDC callback after user authenticates
func APIHandleOIDCCallback(oidcProvider *OIDCProvider, db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		if oidcProvider == nil {
			return apiInternalError(c, "OIDC is not configured")
		}

		ctx := c.Request().Context()

		// verify state
		sess, _ := session.Get("session", c)
		expectedState, _ := sess.Values["oidc_state"].(string)
		expectedNonce, _ := sess.Values["oidc_nonce"].(string)

		if c.QueryParam("state") != expectedState || expectedState == "" {
			return apiBadRequest(c, "Invalid state parameter")
		}

		// check for error from OIDC provider
		if errParam := c.QueryParam("error"); errParam != "" {
			errDesc := c.QueryParam("error_description")
			log.Errorf("OIDC error: %s - %s", errParam, errDesc)
			// return 403 (not 401) to avoid the SPA redirect loop — 401 triggers OIDC login again
			return apiError(c, http.StatusForbidden, "OIDC_ERROR", fmt.Sprintf("Authentication failed: %s", errDesc))
		}

		// exchange code for token
		code := c.QueryParam("code")
		token, err := oidcProvider.oauth2Cfg.Exchange(ctx, code)
		if err != nil {
			log.Errorf("OIDC token exchange failed: %v", err)
			return apiInternalError(c, "Token exchange failed")
		}

		// extract and verify ID token
		rawIDToken, ok := token.Extra("id_token").(string)
		if !ok {
			return apiInternalError(c, "No id_token in response")
		}

		idToken, err := oidcProvider.verifier.Verify(ctx, rawIDToken)
		if err != nil {
			log.Errorf("OIDC token verification failed: %v", err)
			return apiInternalError(c, "Token verification failed")
		}

		// verify nonce
		if idToken.Nonce != expectedNonce {
			return apiBadRequest(c, "Invalid nonce")
		}

		// extract claims
		var claims struct {
			Sub               string   `json:"sub"`
			Email             string   `json:"email"`
			Name              string   `json:"name"`
			PreferredUsername string   `json:"preferred_username"`
			Groups            []string `json:"groups"`
		}
		if err := idToken.Claims(&claims); err != nil {
			return apiInternalError(c, "Cannot read token claims")
		}

		// determine username (prefer preferred_username, fallback to email, then sub)
		username := claims.PreferredUsername
		if username == "" {
			username = claims.Email
		}
		if username == "" {
			username = claims.Sub
		}

		// look up or create user
		user, err := findOrCreateOIDCUser(db, claims.Sub, username, claims.Email, claims.Name, claims.Groups, oidcProvider.adminGroups)
		if err != nil {
			log.Errorf("OIDC user provisioning failed: %v", err)
			return apiInternalError(c, "User provisioning failed")
		}

		// create session using shared helper (respects SessionMaxDuration config)
		createSession(c, user.Username, user.Admin, util.GetDBUserCRC32(user))

		auditLogEvent(c, "user.login", "user", user.Username, map[string]string{"email": user.Email})
		log.Infof("OIDC login successful for user: %s", user.Username)

		// redirect to SPA root
		return c.Redirect(http.StatusTemporaryRedirect, util.BasePath+"/")
	}
}

// findOrCreateOIDCUser looks up a user by OIDC subject, or creates one if auto-provisioning is enabled
func findOrCreateOIDCUser(db store.IStore, sub, username, email, displayName string, userGroups, adminGroups []string) (model.User, error) {
	// indexed lookup by oidc_sub
	u, err := db.GetUserByOIDCSub(sub)
	if err == nil {
		// existing user - update claims
		u.Email = email
		if displayName != "" {
			u.DisplayName = displayName
		}
		if len(adminGroups) > 0 {
			u.Admin = hasGroupOverlap(userGroups, adminGroups)
		}
		u.UpdatedAt = time.Now().UTC()
		if err := db.SaveUser(u); err != nil {
			return model.User{}, err
		}
		return u, nil
	}

	// user not found - check if auto-provisioning is enabled
	if !util.OIDCAutoProvision {
		return model.User{}, fmt.Errorf("user %s not found and auto-provisioning is disabled", username)
	}

	// determine admin status
	isAdmin := false
	if len(adminGroups) > 0 {
		isAdmin = hasGroupOverlap(userGroups, adminGroups)
	}
	// first user gets admin
	users, _ := db.GetUsers()
	if len(users) == 0 {
		isAdmin = true
	}

	now := time.Now().UTC()
	newUser := model.User{
		Username:    username,
		Email:       email,
		DisplayName: displayName,
		OIDCSub:     sub,
		Admin:       isAdmin,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := db.SaveUser(newUser); err != nil {
		return model.User{}, fmt.Errorf("cannot create user: %w", err)
	}

	log.Infof("Auto-provisioned new OIDC user: %s (admin=%v)", username, isAdmin)
	return newUser, nil
}

// hasGroupOverlap checks if any user group matches any admin group
func hasGroupOverlap(userGroups, adminGroups []string) bool {
	groupSet := make(map[string]bool)
	for _, g := range adminGroups {
		groupSet[g] = true
	}
	for _, g := range userGroups {
		if groupSet[g] {
			return true
		}
	}
	return false
}

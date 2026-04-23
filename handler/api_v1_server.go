package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/DigitalTolk/wireguard-ui/model"
	"github.com/DigitalTolk/wireguard-ui/store"
	"github.com/DigitalTolk/wireguard-ui/util"
)

// APIGetServer returns server config (interface + keypair)
func APIGetServer(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		server, err := db.GetServer()
		if err != nil {
			return apiInternalError(c, "Cannot get server config")
		}
		return c.JSON(http.StatusOK, server)
	}
}

// APIUpdateServerInterface updates server interface settings
func APIUpdateServerInterface(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var serverInterface model.ServerInterface
		if err := c.Bind(&serverInterface); err != nil {
			return apiBadRequest(c, "Invalid request body")
		}

		if !util.ValidateServerAddresses(serverInterface.Addresses) {
			return apiBadRequest(c, "Interface IP address must be in CIDR format")
		}

		if serverInterface.ListenPort < 1 || serverInterface.ListenPort > 65535 {
			return apiBadRequest(c, "Listen port must be between 1 and 65535")
		}

		// capture before state
		oldServer, _ := db.GetServer()

		serverInterface.UpdatedAt = time.Now().UTC()

		if err := db.SaveServerInterface(serverInterface); err != nil {
			return apiInternalError(c, "Cannot save server interface")
		}

		log.Infof("Updated server interfaces: %v", serverInterface)
		auditLogEvent(c, "server.interface.update", "server", "interface", map[string]interface{}{
			"before": oldServer.Interface,
			"after":  serverInterface,
		})
		return c.JSON(http.StatusOK, serverInterface)
	}
}

// APIRegenerateServerKeypair generates a new server keypair
func APIRegenerateServerKeypair(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		key, err := wgtypes.GeneratePrivateKey()
		if err != nil {
			return apiInternalError(c, "Cannot generate WireGuard key pair")
		}

		kp := model.ServerKeypair{
			PrivateKey: key.String(),
			PublicKey:  key.PublicKey().String(),
			UpdatedAt:  time.Now().UTC(),
		}

		if err := db.SaveServerKeyPair(kp); err != nil {
			return apiInternalError(c, "Cannot save server keypair")
		}

		log.Infof("Regenerated server keypair")
		auditLogEvent(c, "server.keypair.regenerate", "server", "keypair", nil)
		return c.JSON(http.StatusOK, kp)
	}
}

// APIGetSettings returns global settings
func APIGetSettings(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		settings, err := db.GetGlobalSettings()
		if err != nil {
			return apiInternalError(c, "Cannot get global settings")
		}
		return c.JSON(http.StatusOK, settings)
	}
}

// APIUpdateSettings updates global settings
func APIUpdateSettings(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var settings model.GlobalSetting
		if err := c.Bind(&settings); err != nil {
			return apiBadRequest(c, "Invalid request body")
		}

		if !util.ValidateIPAddressList(settings.DNSServers) {
			return apiBadRequest(c, "Invalid DNS server address")
		}

		if settings.MTU != 0 && (settings.MTU < 1280 || settings.MTU > 9000) {
			return apiBadRequest(c, "MTU must be 0 (to omit) or between 1280 and 9000")
		}

		if settings.PersistentKeepalive < 0 || settings.PersistentKeepalive > 65535 {
			return apiBadRequest(c, "Persistent keepalive must be between 0 and 65535")
		}

		if settings.ConfigFilePath != "" && !strings.HasPrefix(settings.ConfigFilePath, "/") {
			return apiBadRequest(c, "Config file path must be an absolute path starting with /")
		}

		// capture before state
		oldSettings, _ := db.GetGlobalSettings()

		settings.UpdatedAt = time.Now().UTC()

		if err := db.SaveGlobalSettings(settings); err != nil {
			return apiInternalError(c, "Cannot save global settings")
		}

		log.Infof("Updated global settings")
		auditLogEvent(c, "settings.update", "settings", "global", map[string]interface{}{
			"before": oldSettings,
			"after":  settings,
		})
		return c.JSON(http.StatusOK, settings)
	}
}

package handler

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/sabhiram/go-wol/wol"

	"github.com/DigitalTolk/wireguard-ui/model"
	"github.com/DigitalTolk/wireguard-ui/store"
)

// APIListWolHosts returns all Wake-on-LAN hosts
func APIListWolHosts(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		hosts, err := db.GetWakeOnLanHosts()
		if err != nil {
			return apiInternalError(c, "Cannot get WoL hosts")
		}
		if hosts == nil {
			hosts = []model.WakeOnLanHost{}
		}
		return c.JSON(http.StatusOK, hosts)
	}
}

// APISaveWolHost creates or updates a Wake-on-LAN host
func APISaveWolHost(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var body struct {
			Name          string `json:"name"`
			MacAddress    string `json:"mac_address"`
			OldMacAddress string `json:"old_mac_address"`
		}
		if err := c.Bind(&body); err != nil {
			return apiBadRequest(c, "Invalid request body")
		}

		if strings.TrimSpace(body.Name) == "" {
			return apiBadRequest(c, "Name is required")
		}

		host := model.WakeOnLanHost{
			MacAddress: body.MacAddress,
			Name:       body.Name,
		}

		// validate MAC
		if _, err := host.ResolveResourceName(); err != nil {
			return apiBadRequest(c, "Invalid MAC address")
		}

		// if updating with a new MAC address, delete the old one
		if body.OldMacAddress != "" && body.OldMacAddress != body.MacAddress {
			oldHost, err := db.GetWakeOnLanHost(body.OldMacAddress)
			if err == nil && oldHost != nil {
				host.LatestUsed = oldHost.LatestUsed
			}
			db.DeleteWakeOnHostLanHost(body.OldMacAddress)
		} else {
			// preserve latest_used
			existing, err := db.GetWakeOnLanHost(body.MacAddress)
			if err == nil && existing != nil {
				host.LatestUsed = existing.LatestUsed
			}
		}

		if err := db.SaveWakeOnLanHost(host); err != nil {
			return apiInternalError(c, "Cannot save WoL host")
		}

		auditLogEvent(c, "wol.host.save", "wol", host.MacAddress, map[string]string{"name": host.Name})
		return c.JSON(http.StatusOK, host)
	}
}

// APIDeleteWolHost deletes a Wake-on-LAN host
func APIDeleteWolHost(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		macAddress := c.Param("mac")
		if err := db.DeleteWakeOnHostLanHost(macAddress); err != nil {
			return apiInternalError(c, "Cannot delete WoL host")
		}
		auditLogEvent(c, "wol.host.delete", "wol", macAddress, nil)
		return c.NoContent(http.StatusNoContent)
	}
}

// APIWakeHost sends a Wake-on-LAN magic packet
func APIWakeHost(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		macAddress := c.Param("mac")

		host, err := db.GetWakeOnLanHost(macAddress)
		if err != nil || host == nil {
			return apiNotFound(c, "WoL host not found")
		}

		mp, err := wol.New(host.MacAddress)
		if err != nil {
			return apiInternalError(c, fmt.Sprintf("Cannot create magic packet: %v", err))
		}

		bs, err := mp.Marshal()
		if err != nil {
			return apiInternalError(c, fmt.Sprintf("Cannot marshal magic packet: %v", err))
		}

		udpAddr, err := net.ResolveUDPAddr("udp", "255.255.255.255:0")
		if err != nil {
			return apiInternalError(c, fmt.Sprintf("Cannot resolve UDP address: %v", err))
		}

		conn, err := net.DialUDP("udp", nil, udpAddr)
		if err != nil {
			return apiInternalError(c, fmt.Sprintf("Cannot create UDP connection: %v", err))
		}
		defer conn.Close()

		_, err = conn.Write(bs)
		if err != nil {
			return apiInternalError(c, fmt.Sprintf("Cannot send magic packet: %v", err))
		}

		now := time.Now().UTC()
		host.LatestUsed = &now
		if err := db.SaveWakeOnLanHost(*host); err != nil {
			log.Warnf("Cannot update latest_used for WoL host %s: %v", macAddress, err)
		}

		auditLogEvent(c, "wol.host.wake", "wol", host.MacAddress, map[string]string{"name": host.Name})
		return c.JSON(http.StatusOK, host)
	}
}

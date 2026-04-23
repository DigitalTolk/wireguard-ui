package handler

import (
	"encoding/base64"
	"fmt"
	"io/fs"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/rs/xid"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/xuri/excelize/v2"

	"github.com/DigitalTolk/wireguard-ui/emailer"
	"github.com/DigitalTolk/wireguard-ui/model"
	"github.com/DigitalTolk/wireguard-ui/store"
	"github.com/DigitalTolk/wireguard-ui/util"
)

// connectedThreshold defines how recently a peer must have handshaked to be considered connected
var connectedThreshold = 3 * time.Minute

func connectedPeerKeys() map[string]bool {
	keys := make(map[string]bool)
	wgClient, err := wgctrl.New()
	if err != nil {
		log.Warnf("Cannot create wgctrl client: %v", err)
		return keys
	}
	defer wgClient.Close()

	devices, err := wgClient.Devices()
	if err != nil {
		log.Warnf("Cannot list WireGuard devices: %v", err)
		return keys
	}

	for _, dev := range devices {
		for _, peer := range dev.Peers {
			if time.Since(peer.LastHandshakeTime) < connectedThreshold {
				keys[peer.PublicKey.String()] = true
			}
		}
	}
	return keys
}

// APIListClients returns all WireGuard clients
func APIListClients(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		clientDataList, err := db.GetClients(false)
		if err != nil {
			return apiInternalError(c, fmt.Sprintf("Cannot get client list: %v", err))
		}

		search := strings.ToLower(c.QueryParam("search"))
		status := c.QueryParam("status")

		// Only query WireGuard when connected/disconnected filter is active
		var connKeys map[string]bool
		if status == "connected" || status == "disconnected" {
			connKeys = connectedPeerKeys()
		}

		filtered := make([]model.ClientData, 0, len(clientDataList))
		for _, clientData := range clientDataList {
			clientData = util.FillClientSubnetRange(clientData)
			cl := clientData.Client

			// filter by status
			if status == "enabled" && !cl.Enabled {
				continue
			}
			if status == "disabled" && cl.Enabled {
				continue
			}
			if status == "connected" && !connKeys[cl.PublicKey] {
				continue
			}
			if status == "disconnected" && connKeys[cl.PublicKey] {
				continue
			}

			// filter by search
			if search != "" {
				nameLower := strings.ToLower(cl.Name)
				emailLower := strings.ToLower(cl.Email)
				ipsLower := strings.ToLower(strings.Join(cl.AllocatedIPs, " "))
				if !strings.Contains(nameLower, search) && !strings.Contains(emailLower, search) && !strings.Contains(ipsLower, search) {
					continue
				}
			}

			filtered = append(filtered, clientData)
		}
		return c.JSON(http.StatusOK, filtered)
	}
}

// APIGetClient returns a single client by ID
func APIGetClient(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		clientID := c.Param("id")
		if _, err := xid.FromString(clientID); err != nil {
			return apiBadRequest(c, "Invalid client ID")
		}

		clientData, err := db.GetClientByID(clientID, util.DefaultQRCodeSettings)
		if err != nil {
			return apiNotFound(c, "Client not found")
		}
		return c.JSON(http.StatusOK, util.FillClientSubnetRange(clientData))
	}
}

// APICreateClient creates a new WireGuard client
func APICreateClient(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var client model.Client
		if err := c.Bind(&client); err != nil {
			return apiBadRequest(c, "Invalid request body")
		}

		// validate email is required
		if strings.TrimSpace(client.Email) == "" {
			return apiBadRequest(c, "Email is required")
		}

		server, err := db.GetServer()
		if err != nil {
			return apiInternalError(c, "Cannot fetch server config")
		}

		// validate allocated IPs
		allocatedIPs, err := db.GetAllocatedIPs("")
		if err != nil {
			return apiInternalError(c, "Cannot get allocated IPs")
		}
		check, err := util.ValidateIPAllocation(server.Interface.Addresses, allocatedIPs, client.AllocatedIPs)
		if !check {
			return apiBadRequest(c, err.Error())
		}

		if !util.ValidateAllowedIPs(client.AllowedIPs) {
			return apiBadRequest(c, "Allowed IPs must be in CIDR format")
		}

		if !util.ValidateExtraAllowedIPs(client.ExtraAllowedIPs) {
			return apiBadRequest(c, "Extra AllowedIPs must be in CIDR format")
		}

		// generate ID
		client.ID = xid.New().String()

		// generate keypair
		if client.PublicKey == "" {
			key, err := wgtypes.GeneratePrivateKey()
			if err != nil {
				return apiInternalError(c, "Cannot generate WireGuard key pair")
			}
			client.PrivateKey = key.String()
			client.PublicKey = key.PublicKey().String()
		} else {
			if _, err := wgtypes.ParseKey(client.PublicKey); err != nil {
				return apiBadRequest(c, "Cannot verify WireGuard public key")
			}
			// check duplicates
			clients, err := db.GetClients(false)
			if err != nil {
				return apiInternalError(c, "Cannot check for duplicate keys")
			}
			for _, other := range clients {
				if other.Client.PublicKey == client.PublicKey {
					return apiBadRequest(c, "Duplicate public key")
				}
			}
		}

		// generate preshared key
		switch client.PresharedKey {
		case "":
			psk, err := wgtypes.GenerateKey()
			if err != nil {
				return apiInternalError(c, "Cannot generate preshared key")
			}
			client.PresharedKey = psk.String()
		case "-":
			client.PresharedKey = ""
		default:
			if _, err := wgtypes.ParseKey(client.PresharedKey); err != nil {
				return apiBadRequest(c, "Cannot verify preshared key")
			}
		}

		client.CreatedAt = time.Now().UTC()
		client.UpdatedAt = client.CreatedAt

		if err := db.SaveClient(client); err != nil {
			return apiInternalError(c, err.Error())
		}

		log.Infof("Created wireguard client: %v", client.Name)
		auditLogEvent(c, "client.create", "client", client.ID, map[string]string{"name": client.Name, "email": client.Email})
		return c.JSON(http.StatusCreated, client)
	}
}

// APIUpdateClient updates an existing client
func APIUpdateClient(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		clientID := c.Param("id")
		if _, err := xid.FromString(clientID); err != nil {
			return apiBadRequest(c, "Invalid client ID")
		}

		var _client model.Client
		if err := c.Bind(&_client); err != nil {
			return apiBadRequest(c, "Invalid request body")
		}

		clientData, err := db.GetClientByID(clientID, model.QRCodeSettings{Enabled: false})
		if err != nil {
			return apiNotFound(c, "Client not found")
		}

		server, err := db.GetServer()
		if err != nil {
			return apiInternalError(c, "Cannot fetch server config")
		}

		client := *clientData.Client

		allocatedIPs, err := db.GetAllocatedIPs(client.ID)
		if err != nil {
			return apiInternalError(c, "Cannot get allocated IPs")
		}
		check, err := util.ValidateIPAllocation(server.Interface.Addresses, allocatedIPs, _client.AllocatedIPs)
		if !check {
			return apiBadRequest(c, err.Error())
		}

		if !util.ValidateAllowedIPs(_client.AllowedIPs) {
			return apiBadRequest(c, "Allowed IPs must be in CIDR format")
		}
		if !util.ValidateExtraAllowedIPs(_client.ExtraAllowedIPs) {
			return apiBadRequest(c, "Extra Allowed IPs must be in CIDR format")
		}

		// handle public key change
		if client.PublicKey != _client.PublicKey && _client.PublicKey != "" {
			if _, err := wgtypes.ParseKey(_client.PublicKey); err != nil {
				return apiBadRequest(c, "Cannot verify WireGuard public key")
			}
			clients, err := db.GetClients(false)
			if err != nil {
				return apiInternalError(c, "Cannot check for duplicate keys")
			}
			for _, other := range clients {
				if other.Client.PublicKey == _client.PublicKey {
					return apiBadRequest(c, "Duplicate public key")
				}
			}
			if client.PrivateKey != "" {
				client.PrivateKey = ""
			}
		}

		// handle preshared key change
		if client.PresharedKey != _client.PresharedKey && _client.PresharedKey != "" {
			if _, err := wgtypes.ParseKey(_client.PresharedKey); err != nil {
				return apiBadRequest(c, "Cannot verify preshared key")
			}
		}

		client.Name = _client.Name
		// email is immutable after creation — preserve original
		client.Enabled = _client.Enabled
		client.UseServerDNS = _client.UseServerDNS
		client.AllocatedIPs = _client.AllocatedIPs
		client.AllowedIPs = _client.AllowedIPs
		client.ExtraAllowedIPs = _client.ExtraAllowedIPs
		client.Endpoint = _client.Endpoint
		client.PublicKey = _client.PublicKey
		client.PresharedKey = _client.PresharedKey
		client.UpdatedAt = time.Now().UTC()
		client.AdditionalNotes = strings.ReplaceAll(strings.Trim(_client.AdditionalNotes, "\r\n"), "\r\n", "\n")

		if err := db.SaveClient(client); err != nil {
			return apiInternalError(c, err.Error())
		}

		log.Infof("Updated client: %v", client.Name)
		auditLogEvent(c, "client.update", "client", client.ID, map[string]string{"name": client.Name, "email": client.Email})
		return c.JSON(http.StatusOK, client)
	}
}

// APIPatchClientStatus enables/disables a client
func APIPatchClientStatus(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		clientID := c.Param("id")
		if _, err := xid.FromString(clientID); err != nil {
			return apiBadRequest(c, "Invalid client ID")
		}

		var body struct {
			Enabled bool `json:"enabled"`
		}
		if err := c.Bind(&body); err != nil {
			return apiBadRequest(c, "Invalid request body")
		}

		clientData, err := db.GetClientByID(clientID, model.QRCodeSettings{Enabled: false})
		if err != nil {
			return apiNotFound(c, "Client not found")
		}

		client := *clientData.Client
		client.Enabled = body.Enabled
		if err := db.SaveClient(client); err != nil {
			return apiInternalError(c, err.Error())
		}

		action := "client.disable"
		if body.Enabled {
			action = "client.enable"
		}
		log.Infof("Changed client %s enabled status to %v", client.ID, body.Enabled)
		auditLogEvent(c, action, "client", client.ID, map[string]string{"name": client.Name, "email": client.Email})
		return c.JSON(http.StatusOK, client)
	}
}

// APIDeleteClient deletes a client
func APIDeleteClient(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		clientID := c.Param("id")
		if _, err := xid.FromString(clientID); err != nil {
			return apiBadRequest(c, "Invalid client ID")
		}

		if err := db.DeleteClient(clientID); err != nil {
			return apiInternalError(c, "Cannot delete client")
		}

		log.Infof("Deleted wireguard client: %s", clientID)
		auditLogEvent(c, "client.delete", "client", clientID, nil)
		return c.NoContent(http.StatusNoContent)
	}
}

// APIDownloadClientConfig returns the .conf file for a client
func APIDownloadClientConfig(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		clientID := c.Param("id")
		if _, err := xid.FromString(clientID); err != nil {
			return apiBadRequest(c, "Invalid client ID")
		}

		clientData, err := db.GetClientByID(clientID, model.QRCodeSettings{Enabled: false})
		if err != nil {
			return apiNotFound(c, "Client not found")
		}

		server, err := db.GetServer()
		if err != nil {
			return apiInternalError(c, "Cannot get server config")
		}
		globalSettings, err := db.GetGlobalSettings()
		if err != nil {
			return apiInternalError(c, "Cannot get global settings")
		}

		config := util.BuildClientConfig(*clientData.Client, server, globalSettings)
		c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%s.conf", clientData.Client.Name))
		auditLogEvent(c, "client.config.download", "client", clientID, map[string]string{"name": clientData.Client.Name, "email": clientData.Client.Email})
		return c.Stream(http.StatusOK, "text/conf", strings.NewReader(config))
	}
}

// APIGetClientQRCode returns the QR code for a client
func APIGetClientQRCode(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		clientID := c.Param("id")
		if _, err := xid.FromString(clientID); err != nil {
			return apiBadRequest(c, "Invalid client ID")
		}

		clientData, err := db.GetClientByID(clientID, util.DefaultQRCodeSettings)
		if err != nil {
			return apiNotFound(c, "Client not found")
		}

		return c.JSON(http.StatusOK, map[string]string{
			"qr_code": clientData.QRCode,
		})
	}
}

// APIEmailClient sends the client config via email
func APIEmailClient(db store.IStore, mailer emailer.Emailer, emailSubject, emailContent string) echo.HandlerFunc {
	return func(c echo.Context) error {
		clientID := c.Param("id")
		if _, err := xid.FromString(clientID); err != nil {
			return apiBadRequest(c, "Invalid client ID")
		}

		var body struct {
			Email string `json:"email"`
		}
		if err := c.Bind(&body); err != nil {
			return apiBadRequest(c, "Invalid request body")
		}

		qrCodeSettings := model.QRCodeSettings{Enabled: true, IncludeDNS: true, IncludeMTU: true}
		clientData, err := db.GetClientByID(clientID, qrCodeSettings)
		if err != nil {
			return apiNotFound(c, "Client not found")
		}

		server, _ := db.GetServer()
		globalSettings, _ := db.GetGlobalSettings()
		config := util.BuildClientConfig(*clientData.Client, server, globalSettings)

		cfgAtt := emailer.Attachment{Name: "wg0.conf", Data: []byte(config)}
		var attachments []emailer.Attachment
		if clientData.Client.PrivateKey != "" {
			qrdata, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(clientData.QRCode, "data:image/png;base64,"))
			if err != nil {
				return apiInternalError(c, "Cannot decode QR code")
			}
			attachments = []emailer.Attachment{cfgAtt, {Name: "wg.png", Data: qrdata}}
		} else {
			attachments = []emailer.Attachment{cfgAtt}
		}

		err = mailer.Send(clientData.Client.Name, body.Email, emailSubject, emailContent, attachments)
		if err != nil {
			return apiInternalError(c, err.Error())
		}

		auditLogEvent(c, "client.config.email", "client", clientID, map[string]string{"name": clientData.Client.Name, "email": clientData.Client.Email, "sent_to": body.Email})
		return c.JSON(http.StatusOK, map[string]string{"message": "Email sent successfully"})
	}
}

// APISuggestClientIPs suggests available IP addresses
func APISuggestClientIPs(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		server, err := db.GetServer()
		if err != nil {
			return apiInternalError(c, "Cannot fetch server config")
		}

		allocatedIPs, err := db.GetAllocatedIPs("")
		if err != nil {
			return apiInternalError(c, "Cannot get allocated IPs")
		}

		sr := c.QueryParam("sr")
		searchCIDRList := make([]string, 0)

		if util.SubnetRanges[sr] != nil {
			for _, cidr := range util.SubnetRanges[sr] {
				searchCIDRList = append(searchCIDRList, cidr.String())
			}
		} else {
			searchCIDRList = append(searchCIDRList, server.Interface.Addresses...)
		}

		ipSet := make(map[string]struct{})
		found := false

		for _, cidr := range searchCIDRList {
			ip, err := util.GetAvailableIP(cidr, allocatedIPs, server.Interface.Addresses)
			if err != nil {
				continue
			}
			found = true
			if strings.Contains(ip, ":") {
				ipSet[fmt.Sprintf("%s/128", ip)] = struct{}{}
			} else {
				ipSet[fmt.Sprintf("%s/32", ip)] = struct{}{}
			}
		}

		if !found {
			return apiInternalError(c, "No available IPs. Try a different subnet or deallocate some IPs.")
		}

		suggestedIPs := make([]string, 0, len(ipSet))
		for ip := range ipSet {
			suggestedIPs = append(suggestedIPs, ip)
		}
		return c.JSON(http.StatusOK, suggestedIPs)
	}
}

// APIMachineIPs returns local machine IP addresses
func APIMachineIPs() echo.HandlerFunc {
	return func(c echo.Context) error {
		interfaceList, err := util.GetInterfaceIPs()
		if err != nil {
			return apiInternalError(c, "Cannot get machine IP addresses")
		}

		publicInterface, err := util.GetPublicIP()
		if err == nil {
			interfaceList = append([]model.Interface{publicInterface}, interfaceList...)
		}

		return c.JSON(http.StatusOK, interfaceList)
	}
}

// APISubnetRanges returns the ordered list of subnet ranges
func APISubnetRanges() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, util.SubnetRangesOrder)
	}
}

// APIServerStatus returns WireGuard status with connected peers
func APIServerStatus(db store.IStore) echo.HandlerFunc {
	type PeerStatus struct {
		Name              string        `json:"name"`
		Email             string        `json:"email"`
		PublicKey         string        `json:"public_key"`
		ReceivedBytes     int64         `json:"received_bytes"`
		TransmitBytes     int64         `json:"transmit_bytes"`
		LastHandshakeTime time.Time     `json:"last_handshake_time"`
		LastHandshakeRel  time.Duration `json:"last_handshake_rel"`
		Connected         bool          `json:"connected"`
		AllocatedIP       string        `json:"allocated_ip"`
		Endpoint          string        `json:"endpoint,omitempty"`
	}

	type DeviceStatus struct {
		Name  string       `json:"name"`
		Peers []PeerStatus `json:"peers"`
	}

	return func(c echo.Context) error {
		wgClient, err := wgctrl.New()
		if err != nil {
			return apiInternalError(c, err.Error())
		}
		defer wgClient.Close()

		devices, err := wgClient.Devices()
		if err != nil {
			return apiInternalError(c, err.Error())
		}

		devicesStatus := make([]DeviceStatus, 0, len(devices))
		if len(devices) > 0 {
			m := make(map[string]*model.Client)
			clients, _ := db.GetClients(false)
			for i := range clients {
				if clients[i].Client != nil {
					m[clients[i].Client.PublicKey] = clients[i].Client
				}
			}

			conv := map[bool]int{true: 1, false: 0}
			for i := range devices {
				dev := DeviceStatus{Name: devices[i].Name}
				for j := range devices[i].Peers {
					var allocatedIPs string
					for _, ip := range devices[i].Peers[j].AllowedIPs {
						if len(allocatedIPs) > 0 {
							allocatedIPs += ", "
						}
						allocatedIPs += ip.String()
					}
					p := PeerStatus{
						PublicKey:         devices[i].Peers[j].PublicKey.String(),
						ReceivedBytes:     devices[i].Peers[j].ReceiveBytes,
						TransmitBytes:     devices[i].Peers[j].TransmitBytes,
						LastHandshakeTime: devices[i].Peers[j].LastHandshakeTime,
						LastHandshakeRel:  time.Since(devices[i].Peers[j].LastHandshakeTime),
						AllocatedIP:       allocatedIPs,
					}
					p.Connected = p.LastHandshakeRel < connectedThreshold

					if isAdmin(c) && devices[i].Peers[j].Endpoint != nil {
						p.Endpoint = devices[i].Peers[j].Endpoint.String()
					}

					if cl, ok := m[p.PublicKey]; ok {
						p.Name = cl.Name
						p.Email = cl.Email
					}
					dev.Peers = append(dev.Peers, p)
				}
				sort.SliceStable(dev.Peers, func(a, b int) bool { return dev.Peers[a].Name < dev.Peers[b].Name })
				sort.SliceStable(dev.Peers, func(a, b int) bool { return conv[dev.Peers[a].Connected] > conv[dev.Peers[b].Connected] })
				devicesStatus = append(devicesStatus, dev)
			}
		}

		return c.JSON(http.StatusOK, devicesStatus)
	}
}

// APIApplyServerConfig writes the wg0.conf and updates hashes
func APIApplyServerConfig(db store.IStore, tmplDir fs.FS) echo.HandlerFunc {
	return func(c echo.Context) error {
		server, err := db.GetServer()
		if err != nil {
			return apiInternalError(c, "Cannot get server config")
		}
		clients, err := db.GetClients(false)
		if err != nil {
			return apiInternalError(c, "Cannot get client config")
		}
		users, err := db.GetUsers()
		if err != nil {
			return apiInternalError(c, "Cannot get users config")
		}
		settings, err := db.GetGlobalSettings()
		if err != nil {
			return apiInternalError(c, "Cannot get global settings")
		}

		if err := util.WriteWireGuardServerConfig(tmplDir, server, clients, users, settings); err != nil {
			return apiInternalError(c, fmt.Sprintf("Cannot apply config: %v", err))
		}

		if err := util.UpdateHashes(db); err != nil {
			return apiInternalError(c, fmt.Sprintf("Cannot update hashes: %v", err))
		}

		auditLogEvent(c, "server.config.apply", "server", "config", nil)
		return c.JSON(http.StatusOK, map[string]string{"message": "Config applied successfully"})
	}
}

// APIExportClients exports all clients as an Excel file
func APIExportClients(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		clientDataList, err := db.GetClients(false)
		if err != nil {
			return apiInternalError(c, "Cannot get client list")
		}

		f := excelize.NewFile()
		sheet := "Clients"
		f.SetSheetName("Sheet1", sheet)

		headers := []string{"Name", "Email", "Allocated IPs", "Allowed IPs", "Extra Allowed IPs", "Enabled", "Created", "Updated"}
		for i, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			f.SetCellValue(sheet, cell, h)
		}

		style, _ := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true},
		})
		f.SetCellStyle(sheet, "A1", fmt.Sprintf("%s1", string(rune('A'+len(headers)-1))), style)

		for row, cd := range clientDataList {
			cl := cd.Client
			r := row + 2
			f.SetCellValue(sheet, fmt.Sprintf("A%d", r), cl.Name)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", r), cl.Email)
			f.SetCellValue(sheet, fmt.Sprintf("C%d", r), strings.Join(cl.AllocatedIPs, ", "))
			f.SetCellValue(sheet, fmt.Sprintf("D%d", r), strings.Join(cl.AllowedIPs, ", "))
			f.SetCellValue(sheet, fmt.Sprintf("E%d", r), strings.Join(cl.ExtraAllowedIPs, ", "))
			enabled := "No"
			if cl.Enabled {
				enabled = "Yes"
			}
			f.SetCellValue(sheet, fmt.Sprintf("F%d", r), enabled)
			f.SetCellValue(sheet, fmt.Sprintf("G%d", r), cl.CreatedAt.Format("2006-01-02 15:04:05"))
			f.SetCellValue(sheet, fmt.Sprintf("H%d", r), cl.UpdatedAt.Format("2006-01-02 15:04:05"))
		}

		for i := range headers {
			col, _ := excelize.ColumnNumberToName(i + 1)
			f.SetColWidth(sheet, col, col, 25)
		}

		c.Response().Header().Set("Content-Disposition", "attachment; filename=clients.xlsx")
		c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		return f.Write(c.Response())
	}
}

// APIConfigStatus checks if the config has changed
func APIConfigStatus(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		changed := util.HashesChanged(db)
		return c.JSON(http.StatusOK, map[string]bool{"changed": changed})
	}
}

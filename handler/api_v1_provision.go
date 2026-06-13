package handler

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/rs/xid"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/DigitalTolk/wireguard-ui/emailer"
	"github.com/DigitalTolk/wireguard-ui/model"
	"github.com/DigitalTolk/wireguard-ui/store"
	"github.com/DigitalTolk/wireguard-ui/util"
)

// Delivery modes for POST /api/v1/provision-client.
const (
	deliveryConfig  = "config"
	deliveryQRCode  = "qrcode"
	deliveryEmail   = "email"
)

type provisionRequest struct {
	Email    string `json:"email"`
	Delivery string `json:"delivery"`
}

// nameFallbackPattern matches anything that isn't [A-Za-z0-9]; used to strip
// the email local-part down to a name when no Client Name pattern is set.
// Mirrors the same fallback the bulk-create UI uses.
var nameFallbackPattern = regexp.MustCompile(`[^A-Za-z0-9]+`)

// emailFilename mirrors APIEmailClient's filename rules so the .conf
// attachment for provisioned clients is named consistently with what the
// existing email flow produces.
func emailFilename(client model.Client, gs model.GlobalSetting) string {
	var filename string
	switch {
	case strings.TrimSpace(gs.EmailFilenamePattern) == "" && strings.TrimSpace(gs.EmailFilenameReplacement) != "":
		// Static mode: replacement is the literal filename.
		filename = strings.TrimSpace(gs.EmailFilenameReplacement)
	default:
		filename = util.ApplyNamePattern(client.Email, gs.EmailFilenamePattern, gs.EmailFilenameReplacement)
	}
	if strings.TrimSpace(filename) == "" {
		filename = client.Name
	}
	return filename
}

// APIProvisionClient is the one-shot client-create endpoint for external
// callers. Body: {email, delivery}. Delivery picks how the config reaches the
// caller:
//
//   - "config"  → .conf bytes (text/conf), filename derived from the Email
//     Filename pattern, falling back to the client name
//   - "qrcode"  → PNG bytes (image/png)
//   - "email"   → fire off the email to the same address; response is JSON ack
//
// Auth is enforced by APITokenAuth at the route. The endpoint deliberately
// does NOT accept a name in the request — names are derived from the Client
// Name pattern (or the email local-part) so the caller can't bypass the
// naming policy.
func APIProvisionClient(db store.IStore, cw *ConfigWriter, mailer emailer.Emailer, emailSubject, emailContent string) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req provisionRequest
		if err := c.Bind(&req); err != nil {
			return apiBadRequest(c, "Invalid request body")
		}
		req.Email = strings.TrimSpace(req.Email)
		if req.Email == "" {
			return apiBadRequest(c, "Email is required")
		}
		if _, err := mail.ParseAddress(req.Email); err != nil {
			return apiBadRequest(c, "Valid email is required")
		}
		switch req.Delivery {
		case deliveryConfig, deliveryQRCode, deliveryEmail:
			// ok
		default:
			return apiBadRequest(c, fmt.Sprintf("delivery must be one of %q, %q, %q", deliveryConfig, deliveryQRCode, deliveryEmail))
		}

		gs, err := db.GetGlobalSettings()
		if err != nil {
			return apiInternalError(c, "Cannot fetch global settings")
		}
		server, err := db.GetServer()
		if err != nil {
			return apiInternalError(c, "Cannot fetch server config")
		}

		// Derive the client name from the email per the Client Name pattern,
		// falling back to the email's local-part with non-alnum stripped.
		name := strings.TrimSpace(util.ApplyNamePattern(req.Email, gs.ClientNamePattern, gs.ClientNameReplacement))
		if name == "" {
			localPart := strings.SplitN(req.Email, "@", 2)[0]
			name = nameFallbackPattern.ReplaceAllString(localPart, "")
		}
		if name == "" {
			return apiBadRequest(c, "Could not derive a valid client name from the email")
		}

		// Reject duplicate name early — schema also enforces UNIQUE(name) but
		// surfacing it here gives a friendlier 409 instead of a 500.
		existing, err := db.GetClients(false)
		if err != nil {
			return apiInternalError(c, "Cannot check for duplicates")
		}
		for _, ec := range existing {
			if strings.EqualFold(ec.Client.Name, name) {
				return apiConflict(c, fmt.Sprintf("A client named %q already exists", name))
			}
		}

		// Allocate the next available IP from the first interface address.
		allocatedIPs, err := db.GetAllocatedIPs("")
		if err != nil {
			return apiInternalError(c, "Cannot list allocated IPs")
		}
		var chosenIP string
		for _, cidr := range server.Interface.Addresses {
			ip, err := util.GetAvailableIP(cidr, allocatedIPs, server.Interface.Addresses)
			if err == nil {
				if strings.Contains(ip, ":") {
					chosenIP = fmt.Sprintf("%s/128", ip)
				} else {
					chosenIP = fmt.Sprintf("%s/32", ip)
				}
				break
			}
		}
		if chosenIP == "" {
			return apiInternalError(c, "No available IPs to allocate")
		}

		// Build the client. Keys + preshared key are generated server-side —
		// callers don't pass them. Defaults come from the WGUI_DEFAULT_CLIENT_*
		// envs so the org's policy is consistent with the web UI flow.
		defaults := util.ClientDefaultsFromEnv()
		key, err := wgtypes.GeneratePrivateKey()
		if err != nil {
			return apiInternalError(c, "Cannot generate WireGuard key pair")
		}
		psk, err := wgtypes.GenerateKey()
		if err != nil {
			return apiInternalError(c, "Cannot generate preshared key")
		}
		now := time.Now().UTC()
		client := model.Client{
			ID:              xid.New().String(),
			Name:            name,
			Email:           req.Email,
			PrivateKey:      key.String(),
			PublicKey:       key.PublicKey().String(),
			PresharedKey:    psk.String(),
			AllocatedIPs:    []string{chosenIP},
			AllowedIPs:      defaults.AllowedIps,
			ExtraAllowedIPs: defaults.ExtraAllowedIps,
			UseServerDNS:    defaults.UseServerDNS,
			Enabled:         true,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if err := db.SaveClient(client); err != nil {
			return apiInternalError(c, fmt.Sprintf("Cannot save client: %v", err))
		}
		cw.Trigger()
		log.Infof("Provisioned wireguard client: %v (delivery=%s)", client.Name, req.Delivery)
		auditLogEvent(c, "client.provision", "client", client.ID, map[string]string{
			"name": client.Name, "email": client.Email, "delivery": req.Delivery,
		})

		switch req.Delivery {
		case deliveryConfig:
			cfg := util.BuildClientConfig(client, server, gs)
			filename := emailFilename(client, gs)
			c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%q", filename+".conf"))
			return c.Blob(http.StatusCreated, "text/conf", []byte(cfg))

		case deliveryQRCode:
			// Pull the freshly-saved client back with the QR included; we don't
			// have a separate "render QR for this in-memory client" path.
			cd, err := db.GetClientByID(client.ID, util.DefaultQRCodeSettings)
			if err != nil || cd.QRCode == "" {
				return apiInternalError(c, "Cannot render QR code")
			}
			raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(cd.QRCode, "data:image/png;base64,"))
			if err != nil {
				return apiInternalError(c, "Cannot decode QR code")
			}
			c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%q", client.Name+".png"))
			return c.Blob(http.StatusCreated, "image/png", raw)

		case deliveryEmail:
			cd, err := db.GetClientByID(client.ID, util.DefaultQRCodeSettings)
			if err != nil {
				return apiInternalError(c, "Cannot fetch saved client")
			}
			cfg := util.BuildClientConfig(client, server, gs)
			filename := emailFilename(client, gs)
			attachments := []emailer.Attachment{{Name: filename + ".conf", Data: []byte(cfg)}}
			if cd.QRCode != "" {
				if raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(cd.QRCode, "data:image/png;base64,")); err == nil {
					attachments = append(attachments, emailer.Attachment{Name: "wg.png", Data: raw})
				}
			}
			if err := mailer.Send(client.Name, client.Email, emailSubject, emailContent, attachments); err != nil {
				return apiInternalError(c, fmt.Sprintf("Cannot send email: %v", err))
			}
			return c.JSON(http.StatusCreated, map[string]interface{}{
				"id":    client.ID,
				"name":  client.Name,
				"email": client.Email,
				"sent":  true,
			})
		}
		// Unreachable — the switch above is exhaustive over validated values.
		return apiInternalError(c, "Unhandled delivery mode")
	}
}

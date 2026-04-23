package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/DigitalTolk/wireguard-ui/model"
)

func TestAPIGetServer(t *testing.T) {
	env := setupTestEnv(t)

	req, rec := jsonRequest(http.MethodGet, "/api/v1/server", nil)
	c := env.echo.NewContext(req, rec)
	err := APIGetServer(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var server model.Server
	parseJSON(t, rec, &server)
	assert.NotEmpty(t, server.KeyPair.PublicKey)
}

func TestAPIRegenerateServerKeypair(t *testing.T) {
	env := setupTestEnv(t)

	// get original
	orig, _ := env.db.GetServer()

	req, rec := jsonRequest(http.MethodPost, "/api/v1/server/keypair", nil)
	c := env.echo.NewContext(req, rec)
	err := APIRegenerateServerKeypair(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var kp model.ServerKeypair
	parseJSON(t, rec, &kp)
	assert.NotEqual(t, orig.KeyPair.PublicKey, kp.PublicKey)
	assert.NotEmpty(t, kp.PrivateKey)
}

func TestAPIGetSettings(t *testing.T) {
	env := setupTestEnv(t)

	req, rec := jsonRequest(http.MethodGet, "/api/v1/settings", nil)
	c := env.echo.NewContext(req, rec)
	err := APIGetSettings(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var gs model.GlobalSetting
	parseJSON(t, rec, &gs)
	assert.NotEmpty(t, gs.EndpointAddress)
}

func TestAPIUpdateSettings(t *testing.T) {
	env := setupTestEnv(t)

	body := model.GlobalSetting{
		EndpointAddress:     "vpn.new.com",
		DNSServers:          []string{"8.8.8.8"},
		MTU:                 1400,
		PersistentKeepalive: 25,
		FirewallMark:        "0x1234",
		Table:               "auto",
		ConfigFilePath:      "/etc/wireguard/wg0.conf",
	}

	req, rec := jsonRequest(http.MethodPut, "/api/v1/settings", body)
	c := env.echo.NewContext(req, rec)
	err := APIUpdateSettings(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	gs, _ := env.db.GetGlobalSettings()
	assert.Equal(t, "vpn.new.com", gs.EndpointAddress)
}

func TestAPIUpdateSettings_InvalidDNS(t *testing.T) {
	env := setupTestEnv(t)

	body := model.GlobalSetting{
		DNSServers: []string{"not-an-ip"},
	}

	req, rec := jsonRequest(http.MethodPut, "/api/v1/settings", body)
	c := env.echo.NewContext(req, rec)
	err := APIUpdateSettings(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIUpdateServerInterface(t *testing.T) {
	env := setupTestEnv(t)

	body := model.ServerInterface{
		Addresses:  []string{"10.0.0.0/24"},
		ListenPort: 51821,
	}

	req, rec := jsonRequest(http.MethodPut, "/api/v1/server/interface", body)
	c := env.echo.NewContext(req, rec)
	err := APIUpdateServerInterface(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPIUpdateServerInterface_InvalidAddress(t *testing.T) {
	env := setupTestEnv(t)

	body := model.ServerInterface{
		Addresses: []string{"not-cidr"},
	}

	req, rec := jsonRequest(http.MethodPut, "/api/v1/server/interface", body)
	c := env.echo.NewContext(req, rec)
	err := APIUpdateServerInterface(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIUpdateServerInterface_InvalidBody(t *testing.T) {
	env := setupTestEnv(t)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/server/interface", strings.NewReader("{invalid"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := env.echo.NewContext(req, rec)
	err := APIUpdateServerInterface(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIUpdateSettings_InvalidBody(t *testing.T) {
	env := setupTestEnv(t)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", strings.NewReader("{invalid"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := env.echo.NewContext(req, rec)
	err := APIUpdateSettings(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// Regression: frontend sends JSON numbers for mtu/keepalive/listen_port, not strings
func TestAPIUpdateSettings_FrontendJSON(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]interface{}{
		"endpoint_address":     "vpn.test.com",
		"dns_servers":          []string{"1.1.1.1", "8.8.8.8"},
		"mtu":                  1420,
		"persistent_keepalive": 25,
		"firewall_mark":        "0xca6c",
		"table":                "auto",
		"config_file_path":     "/etc/wireguard/wg0.conf",
	}

	req, rec := jsonRequest(http.MethodPut, "/api/v1/settings", body)
	c := env.echo.NewContext(req, rec)
	err := APIUpdateSettings(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	gs, _ := env.db.GetGlobalSettings()
	assert.Equal(t, "vpn.test.com", gs.EndpointAddress)
	assert.Equal(t, 1420, gs.MTU)
	assert.Equal(t, 25, gs.PersistentKeepalive)
	assert.Equal(t, []string{"1.1.1.1", "8.8.8.8"}, gs.DNSServers)
}

func TestAPIUpdateSettings_InvalidMTU_TooLow(t *testing.T) {
	env := setupTestEnv(t)

	body := model.GlobalSetting{
		DNSServers: []string{"8.8.8.8"},
		MTU:        500, // below 1280
	}

	req, rec := jsonRequest(http.MethodPut, "/api/v1/settings", body)
	c := env.echo.NewContext(req, rec)
	err := APIUpdateSettings(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "MTU")
}

func TestAPIUpdateSettings_InvalidMTU_TooHigh(t *testing.T) {
	env := setupTestEnv(t)

	body := model.GlobalSetting{
		DNSServers: []string{"8.8.8.8"},
		MTU:        10000, // above 9000
	}

	req, rec := jsonRequest(http.MethodPut, "/api/v1/settings", body)
	c := env.echo.NewContext(req, rec)
	err := APIUpdateSettings(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIUpdateSettings_InvalidPersistentKeepalive(t *testing.T) {
	env := setupTestEnv(t)

	body := model.GlobalSetting{
		DNSServers:          []string{"8.8.8.8"},
		PersistentKeepalive: -1,
	}

	req, rec := jsonRequest(http.MethodPut, "/api/v1/settings", body)
	c := env.echo.NewContext(req, rec)
	err := APIUpdateSettings(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIUpdateSettings_InvalidConfigFilePath(t *testing.T) {
	env := setupTestEnv(t)

	body := model.GlobalSetting{
		DNSServers:     []string{"8.8.8.8"},
		ConfigFilePath: "relative/path.conf", // not absolute
	}

	req, rec := jsonRequest(http.MethodPut, "/api/v1/settings", body)
	c := env.echo.NewContext(req, rec)
	err := APIUpdateSettings(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "absolute path")
}

func TestAPIUpdateSettings_ZeroMTU(t *testing.T) {
	env := setupTestEnv(t)

	// MTU = 0 should be valid (means omit)
	body := model.GlobalSetting{
		EndpointAddress: "vpn.zero-mtu.com",
		DNSServers:      []string{"8.8.8.8"},
		MTU:             0,
		ConfigFilePath:  "/etc/wireguard/wg0.conf",
	}

	req, rec := jsonRequest(http.MethodPut, "/api/v1/settings", body)
	c := env.echo.NewContext(req, rec)
	err := APIUpdateSettings(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPIUpdateServerInterface_InvalidPort_TooHigh(t *testing.T) {
	env := setupTestEnv(t)

	body := model.ServerInterface{
		Addresses:  []string{"10.0.0.0/24"},
		ListenPort: 70000, // above 65535
	}

	req, rec := jsonRequest(http.MethodPut, "/api/v1/server/interface", body)
	c := env.echo.NewContext(req, rec)
	err := APIUpdateServerInterface(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Listen port")
}

func TestAPIUpdateServerInterface_InvalidPort_Zero(t *testing.T) {
	env := setupTestEnv(t)

	body := model.ServerInterface{
		Addresses:  []string{"10.0.0.0/24"},
		ListenPort: 0, // below 1
	}

	req, rec := jsonRequest(http.MethodPut, "/api/v1/server/interface", body)
	c := env.echo.NewContext(req, rec)
	err := APIUpdateServerInterface(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- APIGetServer returns populated data ---

func TestAPIGetServer_ReturnsFullData(t *testing.T) {
	env := setupTestEnv(t)

	// Update server with specific values so we know what to expect
	iface := model.ServerInterface{
		Addresses:  []string{"10.50.0.0/24", "fd50::1/64"},
		ListenPort: 55555,
		PostUp:     "echo up",
		PostDown:   "echo down",
		UpdatedAt:  time.Now().UTC(),
	}
	require.NoError(t, env.db.SaveServerInterface(iface))

	req, rec := jsonRequest(http.MethodGet, "/api/v1/server", nil)
	c := env.echo.NewContext(req, rec)
	err := APIGetServer(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var server model.Server
	parseJSON(t, rec, &server)
	assert.Equal(t, 55555, server.Interface.ListenPort)
	assert.Contains(t, server.Interface.Addresses, "10.50.0.0/24")
	assert.Contains(t, server.Interface.Addresses, "fd50::1/64")
	assert.NotEmpty(t, server.KeyPair.PublicKey)
	assert.NotEmpty(t, server.KeyPair.PrivateKey)
}

// --- APIGetSettings returns populated data ---

func TestAPIGetSettings_ReturnsFullData(t *testing.T) {
	env := setupTestEnv(t)

	// Save specific settings
	gs := model.GlobalSetting{
		EndpointAddress:     "settings.example.com",
		DNSServers:          []string{"1.1.1.1", "9.9.9.9"},
		MTU:                 1380,
		PersistentKeepalive: 20,
		FirewallMark:        "0xabc",
		Table:               "off",
		ConfigFilePath:      "/etc/wireguard/custom.conf",
		UpdatedAt:           time.Now().UTC(),
	}
	require.NoError(t, env.db.SaveGlobalSettings(gs))

	req, rec := jsonRequest(http.MethodGet, "/api/v1/settings", nil)
	c := env.echo.NewContext(req, rec)
	err := APIGetSettings(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var got model.GlobalSetting
	parseJSON(t, rec, &got)
	assert.Equal(t, "settings.example.com", got.EndpointAddress)
	assert.Equal(t, []string{"1.1.1.1", "9.9.9.9"}, got.DNSServers)
	assert.Equal(t, 1380, got.MTU)
	assert.Equal(t, 20, got.PersistentKeepalive)
	assert.Equal(t, "0xabc", got.FirewallMark)
	assert.Equal(t, "off", got.Table)
}

// --- APIUpdateServerInterface with PostUp/PreDown/PostDown ---

func TestAPIUpdateServerInterface_WithHooks(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]interface{}{
		"addresses":   []string{"10.0.0.0/24"},
		"listen_port": 51820,
		"post_up":     "iptables -A FORWARD -i wg0 -j ACCEPT",
		"pre_down":    "iptables -D FORWARD -i wg0 -j ACCEPT",
		"post_down":   "iptables -D FORWARD -i wg0 -j ACCEPT",
	}

	req, rec := jsonRequest(http.MethodPut, "/api/v1/server/interface", body)
	c := env.echo.NewContext(req, rec)
	err := APIUpdateServerInterface(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	server, _ := env.db.GetServer()
	assert.Equal(t, "iptables -D FORWARD -i wg0 -j ACCEPT", server.Interface.PreDown)
	assert.Equal(t, "iptables -A FORWARD -i wg0 -j ACCEPT", server.Interface.PostUp)
}

// --- APIRegenerateServerKeypair produces valid keys ---

func TestAPIRegenerateServerKeypair_ProducesValidKeys(t *testing.T) {
	env := setupTestEnv(t)

	req, rec := jsonRequest(http.MethodPost, "/api/v1/server/keypair", nil)
	c := env.echo.NewContext(req, rec)
	err := APIRegenerateServerKeypair(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var kp model.ServerKeypair
	parseJSON(t, rec, &kp)
	assert.NotEmpty(t, kp.PrivateKey)
	assert.NotEmpty(t, kp.PublicKey)

	// Verify the key is a valid WireGuard key by parsing it
	privKey, err := wgtypes.ParseKey(kp.PrivateKey)
	require.NoError(t, err)
	assert.Equal(t, kp.PublicKey, privKey.PublicKey().String(), "Public key should derive from private key")

	// Verify the keypair was saved to the database
	server, err := env.db.GetServer()
	require.NoError(t, err)
	assert.Equal(t, kp.PublicKey, server.KeyPair.PublicKey)
	assert.Equal(t, kp.PrivateKey, server.KeyPair.PrivateKey)
}

// --- Error path tests using errStore ---

func TestAPIGetServer_DBError(t *testing.T) {
	db := &errStore{}
	e := echo.New()

	req, rec := jsonRequest(http.MethodGet, "/api/v1/server", nil)
	c := e.NewContext(req, rec)
	err := APIGetServer(db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAPIGetSettings_DBError(t *testing.T) {
	db := &errStore{}
	e := echo.New()

	req, rec := jsonRequest(http.MethodGet, "/api/v1/settings", nil)
	c := e.NewContext(req, rec)
	err := APIGetSettings(db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAPIUpdateServerInterface_SaveError(t *testing.T) {
	db := &errStore{}
	env := setupTestEnv(t)

	body := model.ServerInterface{
		Addresses:  []string{"10.0.0.0/24"},
		ListenPort: 51820,
	}

	req, rec := jsonRequest(http.MethodPut, "/api/v1/server/interface", body)
	c := env.echo.NewContext(req, rec)
	err := APIUpdateServerInterface(db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAPIRegenerateServerKeypair_SaveError(t *testing.T) {
	db := &errStore{}
	env := setupTestEnv(t)

	req, rec := jsonRequest(http.MethodPost, "/api/v1/server/keypair", nil)
	c := env.echo.NewContext(req, rec)
	err := APIRegenerateServerKeypair(db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAPIUpdateSettings_SaveError(t *testing.T) {
	db := &errStore{}
	env := setupTestEnv(t)

	body := model.GlobalSetting{
		EndpointAddress: "vpn.test.com",
		DNSServers:      []string{"8.8.8.8"},
		ConfigFilePath:  "/etc/wireguard/wg0.conf",
	}

	req, rec := jsonRequest(http.MethodPut, "/api/v1/settings", body)
	c := env.echo.NewContext(req, rec)
	err := APIUpdateSettings(db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAPIUpdateServerInterface_FrontendJSON(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]interface{}{
		"addresses":   []string{"10.0.0.0/24"},
		"listen_port": 51821,
		"post_up":     "iptables -A FORWARD",
		"post_down":   "iptables -D FORWARD",
	}

	req, rec := jsonRequest(http.MethodPut, "/api/v1/server/interface", body)
	c := env.echo.NewContext(req, rec)
	err := APIUpdateServerInterface(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	server, _ := env.db.GetServer()
	assert.Equal(t, 51821, server.Interface.ListenPort)
	assert.Equal(t, []string{"10.0.0.0/24"}, server.Interface.Addresses)
}

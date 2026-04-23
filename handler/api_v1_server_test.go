package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	err := APIRegenerateServerKeypair(env.db)(c)
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
	err := APIUpdateSettings(env.db)(c)
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
	err := APIUpdateSettings(env.db)(c)
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
	err := APIUpdateServerInterface(env.db)(c)
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
	err := APIUpdateServerInterface(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIUpdateServerInterface_InvalidBody(t *testing.T) {
	env := setupTestEnv(t)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/server/interface", strings.NewReader("{invalid"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := env.echo.NewContext(req, rec)
	err := APIUpdateServerInterface(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIUpdateSettings_InvalidBody(t *testing.T) {
	env := setupTestEnv(t)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", strings.NewReader("{invalid"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := env.echo.NewContext(req, rec)
	err := APIUpdateSettings(env.db)(c)
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
	err := APIUpdateSettings(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	gs, _ := env.db.GetGlobalSettings()
	assert.Equal(t, "vpn.test.com", gs.EndpointAddress)
	assert.Equal(t, 1420, gs.MTU)
	assert.Equal(t, 25, gs.PersistentKeepalive)
	assert.Equal(t, []string{"1.1.1.1", "8.8.8.8"}, gs.DNSServers)
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
	err := APIUpdateServerInterface(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	server, _ := env.db.GetServer()
	assert.Equal(t, 51821, server.Interface.ListenPort)
	assert.Equal(t, []string{"10.0.0.0/24"}, server.Interface.Addresses)
}

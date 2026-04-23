package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DigitalTolk/wireguard-ui/emailer"
	"github.com/DigitalTolk/wireguard-ui/model"
)

// mockEmailer implements the emailer.Emailer interface for testing
type mockEmailer struct {
	sendCalled bool
	lastTo     string
	shouldFail bool
}

func (m *mockEmailer) Send(toName, to, subject, content string, attachments []emailer.Attachment) error {
	m.sendCalled = true
	m.lastTo = to
	if m.shouldFail {
		return fmt.Errorf("email send failed")
	}
	return nil
}

func makeClient(id string) model.Client {
	now := time.Now().UTC()
	return model.Client{
		ID: id, Name: "Client " + id, AllocatedIPs: []string{}, AllowedIPs: []string{},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	}
}

func TestAPIListClients(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: xid.New().String(), Name: "Client 1", AllocatedIPs: []string{"10.252.1.2/32"},
		AllowedIPs: []string{"0.0.0.0/0"}, ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	req, rec := jsonRequest(http.MethodGet, "/api/v1/clients", nil)
	c := env.echo.NewContext(req, rec)
	err := APIListClients(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var clients []model.ClientData
	parseJSON(t, rec, &clients)
	assert.Len(t, clients, 1)
	assert.Equal(t, "Client 1", clients[0].Client.Name)
}

func TestAPIGetClient(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	env.db.SaveClient(makeClient(id))

	req, rec := jsonRequest(http.MethodGet, "/api/v1/clients/"+id, nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIGetClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPIGetClient_InvalidID(t *testing.T) {
	env := setupTestEnv(t)

	req, rec := jsonRequest(http.MethodGet, "/api/v1/clients/bad", nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad!")
	err := APIGetClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIGetClient_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()

	req, rec := jsonRequest(http.MethodGet, "/api/v1/clients/"+id, nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIGetClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAPIDeleteClient(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	env.db.SaveClient(makeClient(id))

	req, rec := jsonRequest(http.MethodDelete, "/api/v1/clients/"+id, nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIDeleteClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	_, getErr := env.db.GetClientByID(id, model.QRCodeSettings{})
	assert.Error(t, getErr)
}

func TestAPIPatchClientStatus(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	env.db.SaveClient(makeClient(id))

	req, rec := jsonRequest(http.MethodPatch, "/api/v1/clients/"+id+"/status",
		map[string]bool{"enabled": false})
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIPatchClientStatus(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	got, _ := env.db.GetClientByID(id, model.QRCodeSettings{})
	assert.False(t, got.Client.Enabled)
}

func TestAPISuggestClientIPs(t *testing.T) {
	env := setupTestEnv(t)

	req, rec := jsonRequest(http.MethodGet, "/api/v1/suggest-client-ips", nil)
	c := env.echo.NewContext(req, rec)
	err := APISuggestClientIPs(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var ips []string
	parseJSON(t, rec, &ips)
	assert.NotEmpty(t, ips)
}

func TestAPIMachineIPs(t *testing.T) {
	req, rec := jsonRequest(http.MethodGet, "/api/v1/machine-ips", nil)
	e := echo.New()
	c := e.NewContext(req, rec)
	err := APIMachineIPs()(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPISubnetRanges(t *testing.T) {
	req, rec := jsonRequest(http.MethodGet, "/api/v1/subnet-ranges", nil)
	e := echo.New()
	c := e.NewContext(req, rec)
	err := APISubnetRanges()(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPIConfigStatus(t *testing.T) {
	env := setupTestEnv(t)

	req, rec := jsonRequest(http.MethodGet, "/api/v1/server/config-status", nil)
	c := env.echo.NewContext(req, rec)
	err := APIConfigStatus(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]bool
	parseJSON(t, rec, &result)
	_, hasChanged := result["changed"]
	assert.True(t, hasChanged)
}

// --- APICreateClient Tests ---

func TestAPICreateClient_Success(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]interface{}{
		"name":              "New Client",
		"email":             "client@test.com",
		"allocated_ips":     []string{"10.252.1.50/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"use_server_dns":    true,
		"enabled":           true,
	}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/clients", body)
	c := env.echo.NewContext(req, rec)
	err := APICreateClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var client model.Client
	parseJSON(t, rec, &client)
	assert.Equal(t, "New Client", client.Name)
	assert.NotEmpty(t, client.ID)
	assert.NotEmpty(t, client.PublicKey)
	assert.NotEmpty(t, client.PrivateKey)
	assert.NotEmpty(t, client.PresharedKey)
}

func TestAPICreateClient_InvalidAllocatedIPs(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]interface{}{
		"name":              "Bad IPs",
		"email":             "bad@test.com",
		"allocated_ips":     []string{"192.168.99.1/32"}, // outside server range
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
	}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/clients", body)
	c := env.echo.NewContext(req, rec)
	err := APICreateClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPICreateClient_InvalidAllowedIPs(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]interface{}{
		"name":              "Bad Allowed",
		"email":             "bad@test.com",
		"allocated_ips":     []string{"10.252.1.50/32"},
		"allowed_ips":       []string{"not-a-cidr"},
		"extra_allowed_ips": []string{},
	}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/clients", body)
	c := env.echo.NewContext(req, rec)
	err := APICreateClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPICreateClient_InvalidExtraAllowedIPs(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]interface{}{
		"name":              "Bad Extra",
		"email":             "bad@test.com",
		"allocated_ips":     []string{"10.252.1.50/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{"not-a-cidr"},
	}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/clients", body)
	c := env.echo.NewContext(req, rec)
	err := APICreateClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPICreateClient_InvalidTelegramUserid(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]interface{}{
		"name":              "TG Bad",
		"email":             "tg@test.com",
		"telegram_userid":   "notanumber",
		"allocated_ips":     []string{"10.252.1.50/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
	}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/clients", body)
	c := env.echo.NewContext(req, rec)
	err := APICreateClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPICreateClient_WithPresharedKeyDash(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]interface{}{
		"name":              "No PSK",
		"email":             "nopsk@test.com",
		"allocated_ips":     []string{"10.252.1.51/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"preshared_key":     "-",
		"enabled":           true,
	}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/clients", body)
	c := env.echo.NewContext(req, rec)
	err := APICreateClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var client model.Client
	parseJSON(t, rec, &client)
	assert.Empty(t, client.PresharedKey)
}

func TestAPICreateClient_DuplicateAllocatedIP(t *testing.T) {
	env := setupTestEnv(t)

	// First, create a client
	body1 := map[string]interface{}{
		"name":              "First",
		"email":             "first@test.com",
		"allocated_ips":     []string{"10.252.1.60/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"enabled":           true,
	}
	req1, rec1 := jsonRequest(http.MethodPost, "/api/v1/clients", body1)
	c1 := env.echo.NewContext(req1, rec1)
	err := APICreateClient(env.db)(c1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec1.Code)

	// Try to create another with same IP
	body2 := map[string]interface{}{
		"name":              "Second",
		"email":             "second@test.com",
		"allocated_ips":     []string{"10.252.1.60/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
	}
	req2, rec2 := jsonRequest(http.MethodPost, "/api/v1/clients", body2)
	c2 := env.echo.NewContext(req2, rec2)
	err = APICreateClient(env.db)(c2)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec2.Code)
}

// --- APIUpdateClient Tests ---

func TestAPIUpdateClient_Success(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: id, Name: "Original", Email: "original@test.com", PublicKey: "origpub", PrivateKey: "origpriv",
		AllocatedIPs: []string{"10.252.1.70/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	body := map[string]interface{}{
		"name":              "Updated",
		"email":             "try-change@test.com",
		"allocated_ips":     []string{"10.252.1.70/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"public_key":        "origpub",
		"preshared_key":     "",
		"enabled":           true,
		"use_server_dns":    false,
	}
	req, rec := jsonRequest(http.MethodPut, "/api/v1/clients/"+id, body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIUpdateClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var client model.Client
	parseJSON(t, rec, &client)
	assert.Equal(t, "Updated", client.Name)
	// email is immutable — original preserved, attempted change ignored
	assert.Equal(t, "original@test.com", client.Email)
}

func TestAPIUpdateClient_InvalidID(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]interface{}{"name": "test"}
	req, rec := jsonRequest(http.MethodPut, "/api/v1/clients/bad", body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad!")
	err := APIUpdateClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIUpdateClient_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()

	body := map[string]interface{}{
		"name":              "Update",
		"allocated_ips":     []string{"10.252.1.70/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
	}
	req, rec := jsonRequest(http.MethodPut, "/api/v1/clients/"+id, body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIUpdateClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAPIUpdateClient_InvalidAllowedIPs(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: id, Name: "Orig", PublicKey: "pub1",
		AllocatedIPs: []string{"10.252.1.71/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	body := map[string]interface{}{
		"name":              "Bad update",
		"allocated_ips":     []string{"10.252.1.71/32"},
		"allowed_ips":       []string{"not-cidr"},
		"extra_allowed_ips": []string{},
		"public_key":        "pub1",
	}
	req, rec := jsonRequest(http.MethodPut, "/api/v1/clients/"+id, body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIUpdateClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIUpdateClient_InvalidTelegramUserid(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: id, Name: "Orig", PublicKey: "pub1",
		AllocatedIPs: []string{"10.252.1.72/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	body := map[string]interface{}{
		"name":              "TG bad",
		"telegram_userid":   "notanumber",
		"allocated_ips":     []string{"10.252.1.72/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"public_key":        "pub1",
	}
	req, rec := jsonRequest(http.MethodPut, "/api/v1/clients/"+id, body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIUpdateClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- APIDownloadClientConfig Tests ---

func TestAPIDownloadClientConfig_Success(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: id, Name: "Download Client", PublicKey: "dlpub", PrivateKey: "dlpriv",
		AllocatedIPs: []string{"10.252.1.80/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, UseServerDNS: true, CreatedAt: now, UpdatedAt: now,
	})

	req, rec := jsonRequest(http.MethodGet, "/api/v1/clients/"+id+"/config", nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIDownloadClientConfig(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "# CONFIDENTIAL")
	assert.Contains(t, rec.Body.String(), "[Interface]")
	assert.Contains(t, rec.Body.String(), "PrivateKey = dlpriv")
	assert.Contains(t, rec.Header().Get(echo.HeaderContentDisposition), "Download Client.conf")
}

func TestAPIDownloadClientConfig_InvalidID(t *testing.T) {
	env := setupTestEnv(t)

	req, rec := jsonRequest(http.MethodGet, "/api/v1/clients/bad/config", nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad!")
	err := APIDownloadClientConfig(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIDownloadClientConfig_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()

	req, rec := jsonRequest(http.MethodGet, "/api/v1/clients/"+id+"/config", nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIDownloadClientConfig(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- APIGetClientQRCode Tests ---

func TestAPIGetClientQRCode_Success(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: id, Name: "QR Client", PublicKey: "qrpub", PrivateKey: "qrpriv",
		AllocatedIPs: []string{"10.252.1.81/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	req, rec := jsonRequest(http.MethodGet, "/api/v1/clients/"+id+"/qrcode", nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIGetClientQRCode(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]string
	parseJSON(t, rec, &result)
	assert.NotEmpty(t, result["qr_code"])
	assert.Contains(t, result["qr_code"], "data:image/png;base64,")
}

func TestAPIGetClientQRCode_InvalidID(t *testing.T) {
	env := setupTestEnv(t)

	req, rec := jsonRequest(http.MethodGet, "/api/v1/clients/bad/qrcode", nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad!")
	err := APIGetClientQRCode(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIGetClientQRCode_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()

	req, rec := jsonRequest(http.MethodGet, "/api/v1/clients/"+id+"/qrcode", nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIGetClientQRCode(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- APIApplyServerConfig Tests ---

func TestAPIApplyServerConfig_Success(t *testing.T) {
	env := setupTestEnv(t)

	// Set config file path to a temp location
	tmpDir := t.TempDir()
	gs, err := env.db.GetGlobalSettings()
	require.NoError(t, err)
	gs.ConfigFilePath = tmpDir + "/wg0.conf"
	require.NoError(t, env.db.SaveGlobalSettings(gs))

	tmplFS := os.DirFS("../templates")

	req, rec := jsonRequest(http.MethodPost, "/api/v1/server/apply-config", nil)
	c := env.echo.NewContext(req, rec)
	err = APIApplyServerConfig(env.db, tmplFS)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify the file was created
	_, statErr := os.Stat(tmpDir + "/wg0.conf")
	assert.NoError(t, statErr)
}

// --- APIPatchClientStatus edge cases ---

func TestAPIPatchClientStatus_InvalidID(t *testing.T) {
	env := setupTestEnv(t)

	req, rec := jsonRequest(http.MethodPatch, "/api/v1/clients/bad/status",
		map[string]bool{"enabled": true})
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad!")
	err := APIPatchClientStatus(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIPatchClientStatus_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()

	req, rec := jsonRequest(http.MethodPatch, "/api/v1/clients/"+id+"/status",
		map[string]bool{"enabled": true})
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIPatchClientStatus(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAPIPatchClientStatus_Enable(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: id, Name: "Disabled", PublicKey: "pub1",
		AllocatedIPs: []string{}, AllowedIPs: []string{},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: false, CreatedAt: now, UpdatedAt: now,
	})

	req, rec := jsonRequest(http.MethodPatch, "/api/v1/clients/"+id+"/status",
		map[string]bool{"enabled": true})
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIPatchClientStatus(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	got, _ := env.db.GetClientByID(id, model.QRCodeSettings{})
	assert.True(t, got.Client.Enabled)
}

// --- APIDeleteClient edge cases ---

func TestAPIDeleteClient_InvalidID(t *testing.T) {
	env := setupTestEnv(t)

	req, rec := jsonRequest(http.MethodDelete, "/api/v1/clients/bad", nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad!")
	err := APIDeleteClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- APIEmailClient Tests ---

func TestAPIEmailClient_Success(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: id, Name: "Email Client", PublicKey: "emailpub", PrivateKey: "emailpriv",
		AllocatedIPs: []string{"10.252.1.90/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	mailer := &mockEmailer{}

	body := map[string]string{"email": "recipient@test.com"}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/clients/"+id+"/email", body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIEmailClient(env.db, mailer, "WireGuard Config", "Your config is attached.")(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, mailer.sendCalled)
	assert.Equal(t, "recipient@test.com", mailer.lastTo)
}

func TestAPIEmailClient_InvalidID(t *testing.T) {
	env := setupTestEnv(t)
	mailer := &mockEmailer{}

	body := map[string]string{"email": "test@test.com"}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/clients/bad/email", body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad!")
	err := APIEmailClient(env.db, mailer, "", "")(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.False(t, mailer.sendCalled)
}

func TestAPIEmailClient_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	mailer := &mockEmailer{}

	body := map[string]string{"email": "test@test.com"}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/clients/"+id+"/email", body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIEmailClient(env.db, mailer, "", "")(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAPIEmailClient_SendFailure(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: id, Name: "Fail Client", PublicKey: "fpub", PrivateKey: "fpriv",
		AllocatedIPs: []string{"10.252.1.91/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	mailer := &mockEmailer{shouldFail: true}

	body := map[string]string{"email": "fail@test.com"}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/clients/"+id+"/email", body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIEmailClient(env.db, mailer, "Subject", "Body")(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAPIEmailClient_NoPrivateKey(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	now := time.Now().UTC()

	// Client with no private key (external public key provided)
	env.db.SaveClient(model.Client{
		ID: id, Name: "NoPK Client", PublicKey: "nopkpub", PrivateKey: "",
		AllocatedIPs: []string{"10.252.1.92/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	mailer := &mockEmailer{}

	body := map[string]string{"email": "nopk@test.com"}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/clients/"+id+"/email", body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIEmailClient(env.db, mailer, "Subject", "Body")(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, mailer.sendCalled)
}

// --- APIUpdateClient extra edge cases ---

func TestAPIUpdateClient_InvalidExtraAllowedIPs(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: id, Name: "Orig", PublicKey: "pub1",
		AllocatedIPs: []string{"10.252.1.73/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	body := map[string]interface{}{
		"name":              "Bad extra",
		"allocated_ips":     []string{"10.252.1.73/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{"not-cidr"},
		"public_key":        "pub1",
	}
	req, rec := jsonRequest(http.MethodPut, "/api/v1/clients/"+id, body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIUpdateClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- APICreateClient with provided public key ---

func TestAPICreateClient_WithProvidedPublicKey(t *testing.T) {
	env := setupTestEnv(t)

	// Generate a real key pair for a valid public key
	body := map[string]interface{}{
		"name":              "External Key Client",
		"email":             "ext@test.com",
		"allocated_ips":     []string{"10.252.1.52/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"public_key":        "YW52YWxpZGtleWJ1dGVub3VnaGJ5dGVzYWF6enp6eg==", // valid base64 but not a valid WG key
		"enabled":           true,
	}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/clients", body)
	c := env.echo.NewContext(req, rec)
	err := APICreateClient(env.db)(c)
	require.NoError(t, err)
	// Invalid WG key should return bad request
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPICreateClient_WithInvalidPresharedKey(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]interface{}{
		"name":              "Bad PSK",
		"email":             "psk@test.com",
		"allocated_ips":     []string{"10.252.1.53/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"preshared_key":     "not-a-valid-key",
		"enabled":           true,
	}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/clients", body)
	c := env.echo.NewContext(req, rec)
	err := APICreateClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- APIUpdateClient with changed public key ---

func TestAPIUpdateClient_ChangePublicKey_Invalid(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: id, Name: "Orig", PublicKey: "origpub", PrivateKey: "origpriv",
		AllocatedIPs: []string{"10.252.1.74/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	body := map[string]interface{}{
		"name":              "Change Key",
		"allocated_ips":     []string{"10.252.1.74/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"public_key":        "invalid-new-key",
		"enabled":           true,
	}
	req, rec := jsonRequest(http.MethodPut, "/api/v1/clients/"+id, body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIUpdateClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIUpdateClient_ChangePresharedKey_Invalid(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: id, Name: "Orig", PublicKey: "origpub", PresharedKey: "origpsk",
		AllocatedIPs: []string{"10.252.1.75/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	body := map[string]interface{}{
		"name":              "Change PSK",
		"allocated_ips":     []string{"10.252.1.75/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"public_key":        "origpub",
		"preshared_key":     "invalid-psk",
		"enabled":           true,
	}
	req, rec := jsonRequest(http.MethodPut, "/api/v1/clients/"+id, body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIUpdateClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPICreateClient_InvalidBody(t *testing.T) {
	env := setupTestEnv(t)

	// Send non-JSON body with JSON content-type to trigger bind error
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clients", strings.NewReader("{invalid json"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := env.echo.NewContext(req, rec)
	err := APICreateClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIUpdateClient_InvalidBody(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: id, Name: "Orig", PublicKey: "pub1",
		AllocatedIPs: []string{"10.252.1.77/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/clients/"+id, strings.NewReader("{invalid"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIUpdateClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIPatchClientStatus_InvalidBody(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: id, Name: "Patch", PublicKey: "pub1",
		AllocatedIPs: []string{}, AllowedIPs: []string{},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/clients/"+id+"/status", strings.NewReader("{invalid"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIPatchClientStatus(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIEmailClient_InvalidBody(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: id, Name: "Bad Email Body", PublicKey: "pub1",
		AllocatedIPs: []string{"10.252.1.94/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	mailer := &mockEmailer{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clients/"+id+"/email", strings.NewReader("{invalid"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIEmailClient(env.db, mailer, "", "")(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIUpdateClient_InvalidAllocatedIPs(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: id, Name: "Orig", PublicKey: "pub1",
		AllocatedIPs: []string{"10.252.1.76/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	body := map[string]interface{}{
		"name":              "Bad allocated",
		"allocated_ips":     []string{"192.168.99.1/32"}, // outside server range
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"public_key":        "pub1",
	}
	req, rec := jsonRequest(http.MethodPut, "/api/v1/clients/"+id, body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIUpdateClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- APICreateClient email validation ---

func TestAPICreateClient_MissingEmail(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]interface{}{
		"name":              "No Email",
		"allocated_ips":     []string{"10.252.1.54/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"enabled":           true,
	}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/clients", body)
	c := env.echo.NewContext(req, rec)
	err := APICreateClient(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Email is required")
}

// --- APIListClients search and status filtering ---

func TestAPIListClients_SearchFilter(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: xid.New().String(), Name: "Alice Laptop", Email: "alice@test.com",
		AllocatedIPs: []string{"10.252.1.30/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})
	env.db.SaveClient(model.Client{
		ID: xid.New().String(), Name: "Bob Phone", Email: "bob@test.com",
		AllocatedIPs: []string{"10.252.1.31/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: false, CreatedAt: now, UpdatedAt: now,
	})

	// search by name
	req, rec := jsonRequest(http.MethodGet, "/api/v1/clients?search=alice", nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames()
	c.QueryParams().Set("search", "alice")
	err := APIListClients(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	var clients []model.ClientData
	parseJSON(t, rec, &clients)
	assert.Len(t, clients, 1)
	assert.Equal(t, "Alice Laptop", clients[0].Client.Name)
}

func TestAPIListClients_StatusFilter(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: xid.New().String(), Name: "Enabled Client", Email: "en@test.com",
		AllocatedIPs: []string{"10.252.1.32/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})
	env.db.SaveClient(model.Client{
		ID: xid.New().String(), Name: "Disabled Client", Email: "dis@test.com",
		AllocatedIPs: []string{"10.252.1.33/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: false, CreatedAt: now, UpdatedAt: now,
	})

	// filter enabled only
	req, rec := jsonRequest(http.MethodGet, "/api/v1/clients?status=enabled", nil)
	c := env.echo.NewContext(req, rec)
	c.QueryParams().Set("status", "enabled")
	err := APIListClients(env.db)(c)
	require.NoError(t, err)
	var enabledClients []model.ClientData
	parseJSON(t, rec, &enabledClients)
	assert.Len(t, enabledClients, 1)
	assert.Equal(t, "Enabled Client", enabledClients[0].Client.Name)

	// filter disabled only
	req2, rec2 := jsonRequest(http.MethodGet, "/api/v1/clients?status=disabled", nil)
	c2 := env.echo.NewContext(req2, rec2)
	c2.QueryParams().Set("status", "disabled")
	err = APIListClients(env.db)(c2)
	require.NoError(t, err)
	var disabledClients []model.ClientData
	parseJSON(t, rec2, &disabledClients)
	assert.Len(t, disabledClients, 1)
	assert.Equal(t, "Disabled Client", disabledClients[0].Client.Name)
}

// --- APIExportClients ---

func TestAPIExportClients(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: xid.New().String(), Name: "Export Client", Email: "export@test.com",
		AllocatedIPs: []string{"10.252.1.40/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	req, rec := jsonRequest(http.MethodGet, "/api/v1/clients/export", nil)
	c := env.echo.NewContext(req, rec)
	err := APIExportClients(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "clients.xlsx")
	assert.Contains(t, rec.Header().Get("Content-Type"), "spreadsheetml.sheet")
	assert.True(t, rec.Body.Len() > 0)
}

// --- APIServerStatus ---

func TestAPIServerStatus(t *testing.T) {
	env := setupTestEnv(t)

	// Add some clients to exercise the peer matching logic
	now := time.Now().UTC()
	env.db.SaveClient(model.Client{
		ID: xid.New().String(), Name: "Status Client",
		PublicKey: "statuspub", PrivateKey: "statuspriv",
		AllocatedIPs:    []string{"10.252.1.95/32"},
		AllowedIPs:      []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	req, rec := jsonRequest(http.MethodGet, "/api/v1/status", nil)
	c := env.echo.NewContext(req, rec)
	err := APIServerStatus(env.db)(c)
	require.NoError(t, err)
	// On systems with WireGuard support, wgctrl.New() succeeds and returns 200
	// On systems without, it returns 500
	assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, rec.Code)
}

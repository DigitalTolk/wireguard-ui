package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/DigitalTolk/wireguard-ui/emailer"
	"github.com/DigitalTolk/wireguard-ui/model"
	"github.com/DigitalTolk/wireguard-ui/util"
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
	err := APIDeleteClient(env.db, env.cw)(c)
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
	err := APIPatchClientStatus(env.db, env.cw)(c)
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
	err := APICreateClient(env.db, env.cw)(c)
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
	err := APICreateClient(env.db, env.cw)(c)
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
	err := APICreateClient(env.db, env.cw)(c)
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
	err := APICreateClient(env.db, env.cw)(c)
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
	err := APICreateClient(env.db, env.cw)(c)
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
	err := APICreateClient(env.db, env.cw)(c1)
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
	err = APICreateClient(env.db, env.cw)(c2)
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
	err := APIUpdateClient(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var client model.Client
	parseJSON(t, rec, &client)
	assert.Equal(t, "Updated", client.Name)
	// email is immutable — original preserved, attempted change ignored
	assert.Equal(t, "original@test.com", client.Email)
}

// Regression: editing a client must not change its enabled status.
// The edit form does not send "enabled", so Go zero-value (false) was overwriting it.
func TestAPIUpdateClient_PreservesEnabledStatus(t *testing.T) {
	env := setupTestEnv(t)

	now := time.Now().UTC()
	id := xid.New().String()
	env.db.SaveClient(model.Client{
		ID: id, Name: "Stay Enabled", Email: "stay@test.com", PublicKey: "origpub",
		AllocatedIPs: []string{"10.252.1.80/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	// update without sending "enabled" in the body (simulates the edit form)
	body := map[string]interface{}{
		"name":              "Stay Enabled",
		"allocated_ips":     []string{"10.252.1.80/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"public_key":        "origpub",
		"use_server_dns":    true,
	}
	req, rec := jsonRequest(http.MethodPut, "/api/v1/clients/"+id, body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIUpdateClient(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var client model.Client
	parseJSON(t, rec, &client)
	assert.True(t, client.Enabled, "editing a client must not disable it")
}

func TestAPIUpdateClient_InvalidID(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]interface{}{"name": "test"}
	req, rec := jsonRequest(http.MethodPut, "/api/v1/clients/bad", body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad!")
	err := APIUpdateClient(env.db, env.cw)(c)
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
	err := APIUpdateClient(env.db, env.cw)(c)
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
	err := APIUpdateClient(env.db, env.cw)(c)
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

	req, rec := jsonRequest(http.MethodPost, "/api/v1/server/apply-config", nil)
	c := env.echo.NewContext(req, rec)
	err := APIApplyServerConfig(env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// --- APIPatchClientStatus edge cases ---

func TestAPIPatchClientStatus_InvalidID(t *testing.T) {
	env := setupTestEnv(t)

	req, rec := jsonRequest(http.MethodPatch, "/api/v1/clients/bad/status",
		map[string]bool{"enabled": true})
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad!")
	err := APIPatchClientStatus(env.db, env.cw)(c)
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
	err := APIPatchClientStatus(env.db, env.cw)(c)
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
	err := APIPatchClientStatus(env.db, env.cw)(c)
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
	err := APIDeleteClient(env.db, env.cw)(c)
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
	err := APIUpdateClient(env.db, env.cw)(c)
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
	err := APICreateClient(env.db, env.cw)(c)
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
	err := APICreateClient(env.db, env.cw)(c)
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
	err := APIUpdateClient(env.db, env.cw)(c)
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
	err := APIUpdateClient(env.db, env.cw)(c)
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
	err := APICreateClient(env.db, env.cw)(c)
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
	err := APIUpdateClient(env.db, env.cw)(c)
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
	err := APIPatchClientStatus(env.db, env.cw)(c)
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
	err := APIUpdateClient(env.db, env.cw)(c)
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
	err := APICreateClient(env.db, env.cw)(c)
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

// --- currentUserEmail Tests ---

func TestCurrentUserEmail_DisabledLogin(t *testing.T) {
	env := setupTestEnv(t)
	util.DisableLogin = true

	req, rec := jsonRequest(http.MethodGet, "/test", nil)
	c := env.echo.NewContext(req, rec)
	email := currentUserEmail(c, env.db)
	// DisableLogin -> currentUser returns "" -> currentUserEmail returns ""
	assert.Equal(t, "", email)
}

func TestCurrentUserEmail_UserNotFound(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	var email string
	env.echo.GET("/test-email-nf", func(c echo.Context) error {
		// session has a username that doesn't exist in DB
		createSession(c, "nonexistent", false, uint32(0), false)
		email = currentUserEmail(c, env.db)
		return c.String(http.StatusOK, "ok")
	})

	req, rec := jsonRequest(http.MethodGet, "/test-email-nf", nil)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, "", email)
}

func TestCurrentUserEmail_UserExists(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	now := time.Now().UTC()
	env.db.SaveUser(model.User{Username: "emailuser", Email: "emailuser@test.com", Admin: false, CreatedAt: now, UpdatedAt: now})

	var email string
	env.echo.GET("/test-email-found", func(c echo.Context) error {
		createSession(c, "emailuser", false, uint32(0), false)
		return c.String(http.StatusOK, "ok")
	})
	env.echo.GET("/read-email", func(c echo.Context) error {
		email = currentUserEmail(c, env.db)
		return c.String(http.StatusOK, email)
	})

	req1, rec1 := jsonRequest(http.MethodGet, "/test-email-found", nil)
	env.echo.ServeHTTP(rec1, req1)

	cookies := rec1.Result().Cookies()
	req2, rec2 := jsonRequest(http.MethodGet, "/read-email", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	env.echo.ServeHTTP(rec2, req2)
	assert.Equal(t, "emailuser@test.com", email)
}

// --- APICreateClient with valid WireGuard public key ---

func TestAPICreateClient_WithValidPublicKey(t *testing.T) {
	env := setupTestEnv(t)

	// Generate a real WireGuard key for testing
	key, err := wgtypes.GeneratePrivateKey()
	require.NoError(t, err)
	pubKey := key.PublicKey().String()

	body := map[string]interface{}{
		"name":              "External Key",
		"email":             "extkey@test.com",
		"allocated_ips":     []string{"10.252.1.55/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"public_key":        pubKey,
		"enabled":           true,
	}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/clients", body)
	c := env.echo.NewContext(req, rec)
	err = APICreateClient(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var client model.Client
	parseJSON(t, rec, &client)
	assert.Equal(t, pubKey, client.PublicKey)
	assert.Empty(t, client.PrivateKey, "Private key should be empty when public key is provided")
}

// --- APICreateClient with valid preshared key ---

func TestAPICreateClient_WithValidPresharedKey(t *testing.T) {
	env := setupTestEnv(t)

	psk, err := wgtypes.GenerateKey()
	require.NoError(t, err)

	body := map[string]interface{}{
		"name":              "PSK Client",
		"email":             "psk-valid@test.com",
		"allocated_ips":     []string{"10.252.1.56/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"preshared_key":     psk.String(),
		"enabled":           true,
	}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/clients", body)
	c := env.echo.NewContext(req, rec)
	err = APICreateClient(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var client model.Client
	parseJSON(t, rec, &client)
	assert.Equal(t, psk.String(), client.PresharedKey)
}

// --- APICreateClient missing name ---

func TestAPICreateClient_MissingName(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]interface{}{
		"email":             "noname@test.com",
		"allocated_ips":     []string{"10.252.1.57/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"enabled":           true,
	}
	req, rec := jsonRequest(http.MethodPost, "/api/v1/clients", body)
	c := env.echo.NewContext(req, rec)
	err := APICreateClient(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Name is required")
}

// --- APICreateClient duplicate name ---

func TestAPICreateClient_DuplicateName(t *testing.T) {
	env := setupTestEnv(t)

	body1 := map[string]interface{}{
		"name":              "Unique Name",
		"email":             "first@test.com",
		"allocated_ips":     []string{"10.252.1.58/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"enabled":           true,
	}
	req1, rec1 := jsonRequest(http.MethodPost, "/api/v1/clients", body1)
	c1 := env.echo.NewContext(req1, rec1)
	err := APICreateClient(env.db, env.cw)(c1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec1.Code)

	body2 := map[string]interface{}{
		"name":              "unique name", // case-insensitive duplicate
		"email":             "second@test.com",
		"allocated_ips":     []string{"10.252.1.59/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"enabled":           true,
	}
	req2, rec2 := jsonRequest(http.MethodPost, "/api/v1/clients", body2)
	c2 := env.echo.NewContext(req2, rec2)
	err = APICreateClient(env.db, env.cw)(c2)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec2.Code)
	assert.Contains(t, rec2.Body.String(), "name already exists")
}

// --- APICreateClient duplicate public key ---

func TestAPICreateClient_DuplicatePublicKey(t *testing.T) {
	env := setupTestEnv(t)

	key, err := wgtypes.GeneratePrivateKey()
	require.NoError(t, err)
	pubKey := key.PublicKey().String()

	body1 := map[string]interface{}{
		"name":              "Client PKA",
		"email":             "pka@test.com",
		"allocated_ips":     []string{"10.252.1.61/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"public_key":        pubKey,
		"enabled":           true,
	}
	req1, rec1 := jsonRequest(http.MethodPost, "/api/v1/clients", body1)
	c1 := env.echo.NewContext(req1, rec1)
	err = APICreateClient(env.db, env.cw)(c1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec1.Code)

	body2 := map[string]interface{}{
		"name":              "Client PKB",
		"email":             "pkb@test.com",
		"allocated_ips":     []string{"10.252.1.62/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"public_key":        pubKey, // same key
		"enabled":           true,
	}
	req2, rec2 := jsonRequest(http.MethodPost, "/api/v1/clients", body2)
	c2 := env.echo.NewContext(req2, rec2)
	err = APICreateClient(env.db, env.cw)(c2)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec2.Code)
	assert.Contains(t, rec2.Body.String(), "Duplicate public key")
}

// --- APIUpdateClient with valid public key change ---

func TestAPIUpdateClient_ChangePublicKey_Valid(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	now := time.Now().UTC()

	origKey, err := wgtypes.GeneratePrivateKey()
	require.NoError(t, err)

	env.db.SaveClient(model.Client{
		ID: id, Name: "KeyChange", PublicKey: origKey.PublicKey().String(), PrivateKey: origKey.String(),
		AllocatedIPs: []string{"10.252.1.82/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	newKey, err := wgtypes.GeneratePrivateKey()
	require.NoError(t, err)
	newPubKey := newKey.PublicKey().String()

	body := map[string]interface{}{
		"name":              "KeyChange",
		"allocated_ips":     []string{"10.252.1.82/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"public_key":        newPubKey,
		"preshared_key":     "",
		"enabled":           true,
	}
	req, rec := jsonRequest(http.MethodPut, "/api/v1/clients/"+id, body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err = APIUpdateClient(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var client model.Client
	parseJSON(t, rec, &client)
	assert.Equal(t, newPubKey, client.PublicKey)
	assert.Empty(t, client.PrivateKey, "Private key should be cleared when public key changes")
}

// --- APIUpdateClient with valid preshared key change ---

func TestAPIUpdateClient_ChangePresharedKey_Valid(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	now := time.Now().UTC()

	origPSK, err := wgtypes.GenerateKey()
	require.NoError(t, err)

	env.db.SaveClient(model.Client{
		ID: id, Name: "PSKChange", PublicKey: "pubX", PresharedKey: origPSK.String(),
		AllocatedIPs: []string{"10.252.1.83/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	newPSK, err := wgtypes.GenerateKey()
	require.NoError(t, err)

	body := map[string]interface{}{
		"name":              "PSKChange",
		"allocated_ips":     []string{"10.252.1.83/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"public_key":        "pubX",
		"preshared_key":     newPSK.String(),
		"enabled":           true,
	}
	req, rec := jsonRequest(http.MethodPut, "/api/v1/clients/"+id, body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err = APIUpdateClient(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var client model.Client
	parseJSON(t, rec, &client)
	assert.Equal(t, newPSK.String(), client.PresharedKey)
}

// --- APIUpdateClient duplicate name ---

func TestAPIUpdateClient_DuplicateName(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	id1 := xid.New().String()
	id2 := xid.New().String()

	env.db.SaveClient(model.Client{
		ID: id1, Name: "Client Alpha", PublicKey: "alpha-pub",
		AllocatedIPs: []string{"10.252.1.84/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})
	env.db.SaveClient(model.Client{
		ID: id2, Name: "Client Beta", PublicKey: "beta-pub",
		AllocatedIPs: []string{"10.252.1.85/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	// Try to rename Beta to Alpha
	body := map[string]interface{}{
		"name":              "Client Alpha",
		"allocated_ips":     []string{"10.252.1.85/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"public_key":        "beta-pub",
		"preshared_key":     "",
		"enabled":           true,
	}
	req, rec := jsonRequest(http.MethodPut, "/api/v1/clients/"+id2, body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id2)
	err := APIUpdateClient(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "name already exists")
}

// --- APIUpdateClient duplicate public key ---

func TestAPIUpdateClient_DuplicatePublicKey(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	key1, err := wgtypes.GeneratePrivateKey()
	require.NoError(t, err)
	key2, err := wgtypes.GeneratePrivateKey()
	require.NoError(t, err)

	id1 := xid.New().String()
	id2 := xid.New().String()

	env.db.SaveClient(model.Client{
		ID: id1, Name: "DupPK A", PublicKey: key1.PublicKey().String(),
		AllocatedIPs: []string{"10.252.1.86/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})
	env.db.SaveClient(model.Client{
		ID: id2, Name: "DupPK B", PublicKey: key2.PublicKey().String(),
		AllocatedIPs: []string{"10.252.1.87/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	// Try to set B's public key to A's
	body := map[string]interface{}{
		"name":              "DupPK B",
		"allocated_ips":     []string{"10.252.1.87/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"public_key":        key1.PublicKey().String(),
		"preshared_key":     "",
		"enabled":           true,
	}
	req, rec := jsonRequest(http.MethodPut, "/api/v1/clients/"+id2, body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id2)
	err = APIUpdateClient(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Duplicate public key")
}

// --- APIUpdateClient missing name ---

func TestAPIUpdateClient_MissingName(t *testing.T) {
	env := setupTestEnv(t)
	id := xid.New().String()
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: id, Name: "Orig Name", PublicKey: "pubx",
		AllocatedIPs: []string{"10.252.1.88/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	body := map[string]interface{}{
		"name":              "  ",
		"allocated_ips":     []string{"10.252.1.88/32"},
		"allowed_ips":       []string{"0.0.0.0/0"},
		"extra_allowed_ips": []string{},
		"public_key":        "pubx",
	}
	req, rec := jsonRequest(http.MethodPut, "/api/v1/clients/"+id, body)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIUpdateClient(env.db, env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Name is required")
}

// --- APIListClients search by email ---

func TestAPIListClients_SearchByEmail(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: xid.New().String(), Name: "Client A", Email: "unique-email@test.com",
		AllocatedIPs: []string{"10.252.1.34/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})
	env.db.SaveClient(model.Client{
		ID: xid.New().String(), Name: "Client B", Email: "other@test.com",
		AllocatedIPs: []string{"10.252.1.35/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	req, rec := jsonRequest(http.MethodGet, "/api/v1/clients?search=unique-email", nil)
	c := env.echo.NewContext(req, rec)
	c.QueryParams().Set("search", "unique-email")
	err := APIListClients(env.db)(c)
	require.NoError(t, err)
	var clients []model.ClientData
	parseJSON(t, rec, &clients)
	assert.Len(t, clients, 1)
	assert.Equal(t, "Client A", clients[0].Client.Name)
}

// --- APIListClients search by IP ---

func TestAPIListClients_SearchByIP(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: xid.New().String(), Name: "IP Client", Email: "ip@test.com",
		AllocatedIPs: []string{"10.252.1.36/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	req, rec := jsonRequest(http.MethodGet, "/api/v1/clients?search=10.252.1.36", nil)
	c := env.echo.NewContext(req, rec)
	c.QueryParams().Set("search", "10.252.1.36")
	err := APIListClients(env.db)(c)
	require.NoError(t, err)
	var clients []model.ClientData
	parseJSON(t, rec, &clients)
	assert.Len(t, clients, 1)
	assert.Equal(t, "IP Client", clients[0].Client.Name)
}

// --- Non-admin access tests ---
// Register ALL routes before the first ServeHTTP call to avoid Echo router panics.

func TestNonAdmin_ClientAccess(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	now := time.Now().UTC()
	env.db.SaveUser(model.User{Username: "naviewer", Email: "naviewer@test.com", Admin: false, CreatedAt: now, UpdatedAt: now})
	crc := util.GetDBUserCRC32(model.User{Username: "naviewer", Email: "naviewer@test.com", Admin: false, CreatedAt: now, UpdatedAt: now})
	util.DBUsersToCRC32Mutex.Lock()
	util.DBUsersToCRC32["naviewer"] = crc
	util.DBUsersToCRC32Mutex.Unlock()
	defer func() {
		util.DBUsersToCRC32Mutex.Lock()
		delete(util.DBUsersToCRC32, "naviewer")
		util.DBUsersToCRC32Mutex.Unlock()
	}()

	ownID := xid.New().String()
	otherID := xid.New().String()
	env.db.SaveClient(model.Client{
		ID: ownID, Name: "My Own", Email: "naviewer@test.com",
		PublicKey: "myownpub", PrivateKey: "myownpriv",
		AllocatedIPs: []string{"10.252.1.110/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, UseServerDNS: true, CreatedAt: now, UpdatedAt: now,
	})
	env.db.SaveClient(model.Client{
		ID: otherID, Name: "Others", Email: "other@test.com",
		PublicKey: "otherspub", PrivateKey: "otherspriv",
		AllocatedIPs: []string{"10.252.1.111/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	// Register ALL routes before first ServeHTTP
	env.echo.GET("/na-setup", func(c echo.Context) error {
		createSession(c, "naviewer", false, crc, false)
		return c.String(http.StatusOK, "ok")
	})
	env.echo.GET("/na-get/:id", APIGetClient(env.db))
	env.echo.GET("/na-dl/:id", APIDownloadClientConfig(env.db))
	env.echo.GET("/na-qr/:id", APIGetClientQRCode(env.db))
	env.echo.GET("/na-list", APIListClients(env.db))

	// Create session
	req1, rec1 := jsonRequest(http.MethodGet, "/na-setup", nil)
	env.echo.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)
	cookies := rec1.Result().Cookies()

	addCookies := func(req *http.Request) {
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
	}

	// Test: get own client -> OK
	req, rec := jsonRequest(http.MethodGet, "/na-get/"+ownID, nil)
	addCookies(req)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code, "Non-admin should see own client")

	// Test: get other's client -> Forbidden
	req, rec = jsonRequest(http.MethodGet, "/na-get/"+otherID, nil)
	addCookies(req)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code, "Non-admin should not see other's client")

	// Test: download own config -> OK
	req, rec = jsonRequest(http.MethodGet, "/na-dl/"+ownID, nil)
	addCookies(req)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code, "Non-admin should download own config")
	assert.Contains(t, rec.Body.String(), "[Interface]")

	// Test: download other's config -> Forbidden
	req, rec = jsonRequest(http.MethodGet, "/na-dl/"+otherID, nil)
	addCookies(req)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code, "Non-admin should not download other's config")

	// Test: get own QR -> OK
	req, rec = jsonRequest(http.MethodGet, "/na-qr/"+ownID, nil)
	addCookies(req)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code, "Non-admin should see own QR code")

	// Test: get other's QR -> Forbidden
	req, rec = jsonRequest(http.MethodGet, "/na-qr/"+otherID, nil)
	addCookies(req)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code, "Non-admin should not see other's QR code")

	// Test: list clients -> only own
	req, rec = jsonRequest(http.MethodGet, "/na-list", nil)
	addCookies(req)
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	var clients []model.ClientData
	parseJSON(t, rec, &clients)
	assert.Len(t, clients, 1, "Non-admin should only see own clients")
	assert.Equal(t, "My Own", clients[0].Client.Name)
}

// --- Non-admin APIEmailClient ---

func TestAPIEmailClient_NonAdmin_OtherClient(t *testing.T) {
	origDisable := util.DisableLogin
	util.DisableLogin = false
	defer func() { util.DisableLogin = origDisable }()

	env := setupTestEnv(t)
	util.DisableLogin = false

	now := time.Now().UTC()
	env.db.SaveUser(model.User{Username: "emailna", Email: "emailna@test.com", Admin: false, CreatedAt: now, UpdatedAt: now})
	crc := util.GetDBUserCRC32(model.User{Username: "emailna", Email: "emailna@test.com", Admin: false, CreatedAt: now, UpdatedAt: now})
	util.DBUsersToCRC32Mutex.Lock()
	util.DBUsersToCRC32["emailna"] = crc
	util.DBUsersToCRC32Mutex.Unlock()
	defer func() {
		util.DBUsersToCRC32Mutex.Lock()
		delete(util.DBUsersToCRC32, "emailna")
		util.DBUsersToCRC32Mutex.Unlock()
	}()

	id := xid.New().String()
	env.db.SaveClient(model.Client{
		ID: id, Name: "Email Other", Email: "other@test.com",
		PublicKey: "emailothpub", PrivateKey: "emailothpriv",
		AllocatedIPs: []string{"10.252.1.104/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	mailer := &mockEmailer{}

	// Register routes before any ServeHTTP
	env.echo.GET("/email-na-setup", func(c echo.Context) error {
		createSession(c, "emailna", false, crc, false)
		return c.String(http.StatusOK, "ok")
	})
	env.echo.POST("/email-deny/:id", APIEmailClient(env.db, mailer, "Subject", "Body"))

	// Create session
	req1, rec1 := jsonRequest(http.MethodGet, "/email-na-setup", nil)
	env.echo.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)
	cookies := rec1.Result().Cookies()

	body := map[string]string{"email": "test@test.com"}
	req, rec := jsonRequest(http.MethodPost, "/email-deny/"+id, body)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	env.echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// --- APISuggestClientIPs with subnet range parameter ---

func TestAPISuggestClientIPs_WithSubnetRange(t *testing.T) {
	env := setupTestEnv(t)

	// Set up a subnet range within the server's address space
	origRanges := util.SubnetRanges
	origOrder := util.SubnetRangesOrder
	defer func() {
		util.SubnetRanges = origRanges
		util.SubnetRangesOrder = origOrder
	}()

	util.SubnetRanges = util.ParseSubnetRanges("testrange:10.252.1.0/26")

	req, rec := jsonRequest(http.MethodGet, "/api/v1/suggest-client-ips?sr=testrange", nil)
	c := env.echo.NewContext(req, rec)
	c.QueryParams().Set("sr", "testrange")
	err := APISuggestClientIPs(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var ips []string
	parseJSON(t, rec, &ips)
	assert.NotEmpty(t, ips)
}

func TestAPISuggestClientIPs_AllAllocated(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	// Allocate many IPs to exhaust the subnet
	// Server uses 10.252.1.0/24 by default. The server itself uses .1.
	// Let's allocate the first few IPs and check we still get a suggestion
	for i := 2; i < 5; i++ {
		env.db.SaveClient(model.Client{
			ID:              fmt.Sprintf("exhaust-%d", i),
			Name:            fmt.Sprintf("Client %d", i),
			AllocatedIPs:    []string{fmt.Sprintf("10.252.1.%d/32", i)},
			AllowedIPs:      []string{"0.0.0.0/0"},
			ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
			Enabled: true, CreatedAt: now, UpdatedAt: now,
		})
	}

	req, rec := jsonRequest(http.MethodGet, "/api/v1/suggest-client-ips", nil)
	c := env.echo.NewContext(req, rec)
	err := APISuggestClientIPs(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var ips []string
	parseJSON(t, rec, &ips)
	assert.NotEmpty(t, ips)
}

// --- APIListClients search by name with no results ---

func TestAPIListClients_SearchNoMatch(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: xid.New().String(), Name: "Alice", Email: "alice@test.com",
		AllocatedIPs: []string{"10.252.1.120/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	req, rec := jsonRequest(http.MethodGet, "/api/v1/clients?search=zzzznotfound", nil)
	c := env.echo.NewContext(req, rec)
	c.QueryParams().Set("search", "zzzznotfound")
	err := APIListClients(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	var clients []model.ClientData
	parseJSON(t, rec, &clients)
	assert.Len(t, clients, 0)
}

// --- APIListClients with multiple clients and search by partial IP ---

func TestAPIListClients_SearchByPartialIP(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	env.db.SaveClient(model.Client{
		ID: xid.New().String(), Name: "PartialIPClient1", Email: "pip1@test.com",
		AllocatedIPs: []string{"10.252.1.121/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})
	env.db.SaveClient(model.Client{
		ID: xid.New().String(), Name: "PartialIPClient2", Email: "pip2@test.com",
		AllocatedIPs: []string{"10.252.1.122/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})
	env.db.SaveClient(model.Client{
		ID: xid.New().String(), Name: "PartialIPClient3", Email: "pip3@test.com",
		AllocatedIPs: []string{"10.252.2.10/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	// Search with partial IP that matches two clients
	req, rec := jsonRequest(http.MethodGet, "/api/v1/clients?search=252.1.12", nil)
	c := env.echo.NewContext(req, rec)
	c.QueryParams().Set("search", "252.1.12")
	err := APIListClients(env.db)(c)
	require.NoError(t, err)
	var clients []model.ClientData
	parseJSON(t, rec, &clients)
	assert.Len(t, clients, 2)
}

// --- APISuggestClientIPs with unknown subnet range falls back to server addresses ---

func TestAPISuggestClientIPs_UnknownSubnetRange(t *testing.T) {
	env := setupTestEnv(t)

	req, rec := jsonRequest(http.MethodGet, "/api/v1/suggest-client-ips?sr=nonexistent", nil)
	c := env.echo.NewContext(req, rec)
	c.QueryParams().Set("sr", "nonexistent")
	err := APISuggestClientIPs(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var ips []string
	parseJSON(t, rec, &ips)
	assert.NotEmpty(t, ips) // Falls back to server addresses
}

// --- APISuggestClientIPs with exhausted tiny subnet ---

func TestAPISuggestClientIPs_ExhaustedSubnet(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	// Save custom server interface with a tiny /30 subnet (only 2 usable IPs)
	iface := model.ServerInterface{
		Addresses:  []string{"10.253.0.0/30"},
		ListenPort: 51820,
		UpdatedAt:  now,
	}
	require.NoError(t, env.db.SaveServerInterface(iface))

	// Allocate all usable IPs (server takes .0, so .1 and .2 are available)
	env.db.SaveClient(model.Client{
		ID: xid.New().String(), Name: "Exhaust1",
		AllocatedIPs: []string{"10.253.0.1/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})
	env.db.SaveClient(model.Client{
		ID: xid.New().String(), Name: "Exhaust2",
		AllocatedIPs: []string{"10.253.0.2/32"}, AllowedIPs: []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	})

	req, rec := jsonRequest(http.MethodGet, "/api/v1/suggest-client-ips", nil)
	c := env.echo.NewContext(req, rec)
	err := APISuggestClientIPs(env.db)(c)
	require.NoError(t, err)
	// Should return error since all IPs are exhausted
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "No available IPs")
}

// --- APISuggestClientIPs with IPv6 server ---

func TestAPISuggestClientIPs_IPv6(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now().UTC()

	// Set up a dual-stack server
	iface := model.ServerInterface{
		Addresses:  []string{"10.252.1.0/24", "fd00:abcd::1/64"},
		ListenPort: 51820,
		UpdatedAt:  now,
	}
	require.NoError(t, env.db.SaveServerInterface(iface))

	req, rec := jsonRequest(http.MethodGet, "/api/v1/suggest-client-ips", nil)
	c := env.echo.NewContext(req, rec)
	err := APISuggestClientIPs(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var ips []string
	parseJSON(t, rec, &ips)
	assert.NotEmpty(t, ips)
	// Should contain both IPv4 (/32) and IPv6 (/128) suggestions
	hasIPv4 := false
	hasIPv6 := false
	for _, ip := range ips {
		if strings.HasSuffix(ip, "/32") {
			hasIPv4 = true
		}
		if strings.HasSuffix(ip, "/128") {
			hasIPv6 = true
		}
	}
	assert.True(t, hasIPv4, "Should suggest IPv4 address")
	assert.True(t, hasIPv6, "Should suggest IPv6 address")
}

// --- Error path tests using errStore ---

func TestAPIListClients_DBError(t *testing.T) {
	db := &errStore{}
	e := echo.New()

	req, rec := jsonRequest(http.MethodGet, "/api/v1/clients", nil)
	c := e.NewContext(req, rec)
	err := APIListClients(db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAPISuggestClientIPs_DBServerError(t *testing.T) {
	db := &errStore{}
	e := echo.New()

	req, rec := jsonRequest(http.MethodGet, "/api/v1/suggest-client-ips", nil)
	c := e.NewContext(req, rec)
	err := APISuggestClientIPs(db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAPIConfigStatus_DBError(t *testing.T) {
	db := &errStore{}
	e := echo.New()

	req, rec := jsonRequest(http.MethodGet, "/api/v1/server/config-status", nil)
	c := e.NewContext(req, rec)
	err := APIConfigStatus(db)(c)
	require.NoError(t, err)
	// ConfigStatus uses util.HashesChanged which handles DB errors internally
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPIExportClients_DBError(t *testing.T) {
	db := &errStore{}
	e := echo.New()

	req, rec := jsonRequest(http.MethodGet, "/api/v1/clients/export", nil)
	c := e.NewContext(req, rec)
	err := APIExportClients(db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAPIDeleteClient_DBError(t *testing.T) {
	db := &errStore{}
	env := setupTestEnv(t)
	id := xid.New().String()

	req, rec := jsonRequest(http.MethodDelete, "/api/v1/clients/"+id, nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)
	err := APIDeleteClient(db, env.cw)(c)
	require.NoError(t, err)
	// errStore fails on GetClientByID (lookup before delete), returns 404
	assert.Equal(t, http.StatusNotFound, rec.Code)
}


func TestAPIApplyServerConfig_Error(t *testing.T) {
	env := setupTestEnv(t)

	// Set config file path to impossible location to make ApplyNow fail
	gs, _ := env.db.GetGlobalSettings()
	gs.ConfigFilePath = "/dev/null/impossible/path/wg0.conf"
	env.db.SaveGlobalSettings(gs)

	req, rec := jsonRequest(http.MethodPost, "/api/v1/server/apply-config", nil)
	c := env.echo.NewContext(req, rec)
	err := APIApplyServerConfig(env.cw)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Cannot apply config")
}

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

// Regression: peers that never connected (zero handshake time) must not be
// reported as connected. Both connectedPeerKeys and APIServerStatus use
// isConnected() which guards against time.Since(zero).
func TestIsConnected_ZeroHandshake(t *testing.T) {
	assert.False(t, isConnected(time.Time{}), "zero time must not be connected")
}

func TestIsConnected_RecentHandshake(t *testing.T) {
	assert.True(t, isConnected(time.Now().Add(-30*time.Second)), "30s ago must be connected")
}

func TestIsConnected_OldHandshake(t *testing.T) {
	assert.False(t, isConnected(time.Now().Add(-10*time.Minute)), "10min ago must be disconnected")
}

func TestIsConnected_ExactlyAtThreshold(t *testing.T) {
	assert.False(t, isConnected(time.Now().Add(-connectedThreshold)), "exactly at threshold must be disconnected")
}

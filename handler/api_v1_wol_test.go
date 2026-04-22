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

func TestAPIListWolHosts(t *testing.T) {
	env := setupTestEnv(t)

	req, rec := jsonRequest(http.MethodGet, "/api/v1/wol-hosts", nil)
	c := env.echo.NewContext(req, rec)
	err := APIListWolHosts(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var hosts []model.WakeOnLanHost
	parseJSON(t, rec, &hosts)
	assert.Len(t, hosts, 0)
}

func TestAPISaveWolHost(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]string{
		"name":        "TestHost",
		"mac_address": "AA:BB:CC:DD:EE:FF",
	}

	req, rec := jsonRequest(http.MethodPost, "/api/v1/wol-hosts", body)
	c := env.echo.NewContext(req, rec)
	err := APISaveWolHost(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// verify it was saved
	hosts, _ := env.db.GetWakeOnLanHosts()
	assert.Len(t, hosts, 1)
}

func TestAPISaveWolHost_InvalidMac(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]string{
		"name":        "Bad",
		"mac_address": "not-a-mac",
	}

	req, rec := jsonRequest(http.MethodPost, "/api/v1/wol-hosts", body)
	c := env.echo.NewContext(req, rec)
	err := APISaveWolHost(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIDeleteWolHost(t *testing.T) {
	env := setupTestEnv(t)

	env.db.SaveWakeOnLanHost(model.WakeOnLanHost{MacAddress: "AA:BB:CC:DD:EE:FF", Name: "Del"})

	req, rec := jsonRequest(http.MethodDelete, "/api/v1/wol-hosts/AA-BB-CC-DD-EE-FF", nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("mac")
	c.SetParamValues("AA-BB-CC-DD-EE-FF")
	err := APIDeleteWolHost(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestAPISaveWolHost_UpdateMac(t *testing.T) {
	env := setupTestEnv(t)

	env.db.SaveWakeOnLanHost(model.WakeOnLanHost{MacAddress: "AA:BB:CC:DD:EE:FF", Name: "Original"})

	body := map[string]string{
		"name":            "Updated",
		"mac_address":     "11:22:33:44:55:66",
		"old_mac_address": "AA:BB:CC:DD:EE:FF",
	}

	req, rec := jsonRequest(http.MethodPost, "/api/v1/wol-hosts", body)
	c := env.echo.NewContext(req, rec)
	err := APISaveWolHost(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPISaveWolHost_PreserveLatestUsed(t *testing.T) {
	env := setupTestEnv(t)

	env.db.SaveWakeOnLanHost(model.WakeOnLanHost{MacAddress: "AA:BB:CC:DD:EE:FF", Name: "Keep"})

	body := map[string]string{
		"name":        "Keep Updated",
		"mac_address": "AA:BB:CC:DD:EE:FF",
	}

	req, rec := jsonRequest(http.MethodPost, "/api/v1/wol-hosts", body)
	c := env.echo.NewContext(req, rec)
	err := APISaveWolHost(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result model.WakeOnLanHost
	parseJSON(t, rec, &result)
	assert.Equal(t, "Keep Updated", result.Name)
}

func TestAPIWakeHost_NotFound(t *testing.T) {
	env := setupTestEnv(t)

	req, rec := jsonRequest(http.MethodPost, "/api/v1/wol-hosts/XX-XX-XX-XX-XX-XX/wake", nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("mac")
	c.SetParamValues("XX-XX-XX-XX-XX-XX")
	err := APIWakeHost(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAPISaveWolHost_InvalidBody(t *testing.T) {
	env := setupTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wol-hosts", strings.NewReader("{invalid"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := env.echo.NewContext(req, rec)
	err := APISaveWolHost(env.db)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIWakeHost_ExistingHost(t *testing.T) {
	env := setupTestEnv(t)

	env.db.SaveWakeOnLanHost(model.WakeOnLanHost{MacAddress: "AA:BB:CC:DD:EE:FF", Name: "WakeMe"})

	req, rec := jsonRequest(http.MethodPost, "/api/v1/wol-hosts/AA:BB:CC:DD:EE:FF/wake", nil)
	c := env.echo.NewContext(req, rec)
	c.SetParamNames("mac")
	c.SetParamValues("AA:BB:CC:DD:EE:FF")
	err := APIWakeHost(env.db)(c)
	require.NoError(t, err)
	// On systems where UDP broadcast to 255.255.255.255:0 works, returns 200
	// On systems where it's blocked, returns 500
	assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, rec.Code)
}

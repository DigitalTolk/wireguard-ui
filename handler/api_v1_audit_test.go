package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DigitalTolk/wireguard-ui/audit"
)

func TestAPIListAuditLogs(t *testing.T) {
	env := setupTestEnv(t)

	// add some audit entries
	env.auditLog.Log(audit.Entry{Actor: "admin", Action: "test.action", IPAddress: "10.0.0.1"})
	env.auditLog.Log(audit.Entry{Actor: "admin", Action: "test.action2", IPAddress: "10.0.0.1"})

	req, rec := jsonRequest(http.MethodGet, "/api/v1/audit-logs", nil)
	c := env.echo.NewContext(req, rec)
	err := APIListAuditLogs(env.auditLog)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]interface{}
	parseJSON(t, rec, &result)
	assert.Equal(t, float64(2), result["total"])
}

func TestAPIListAuditLogs_WithFilter(t *testing.T) {
	env := setupTestEnv(t)

	env.auditLog.Log(audit.Entry{Actor: "admin", Action: "user.create", IPAddress: "10.0.0.1"})
	env.auditLog.Log(audit.Entry{Actor: "user1", Action: "client.create", IPAddress: "10.0.0.2"})

	req, rec := jsonRequest(http.MethodGet, "/api/v1/audit-logs?actor=admin", nil)
	c := env.echo.NewContext(req, rec)
	c.QueryParams().Set("actor", "admin")
	err := APIListAuditLogs(env.auditLog)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]interface{}
	parseJSON(t, rec, &result)
	assert.Equal(t, float64(1), result["total"])
}

func TestAPIListAuditLogs_EmptyResults(t *testing.T) {
	env := setupTestEnv(t)

	// Query with no matching entries - returns nil entries
	req, rec := jsonRequest(http.MethodGet, "/api/v1/audit-logs?actor=nonexistent", nil)
	c := env.echo.NewContext(req, rec)
	c.QueryParams().Set("actor", "nonexistent")
	err := APIListAuditLogs(env.auditLog)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]interface{}
	parseJSON(t, rec, &result)
	assert.Equal(t, float64(0), result["total"])
	// Should return empty array, not null
	data := result["data"].([]interface{})
	assert.NotNil(t, data)
}

func TestAPIListAuditLogs_Pagination(t *testing.T) {
	env := setupTestEnv(t)

	for i := 0; i < 5; i++ {
		env.auditLog.Log(audit.Entry{Actor: "admin", Action: "test", IPAddress: "10.0.0.1"})
	}

	req, rec := jsonRequest(http.MethodGet, "/api/v1/audit-logs?page=0&per_page=2", nil)
	c := env.echo.NewContext(req, rec)
	c.QueryParams().Set("page", "0")
	c.QueryParams().Set("per_page", "2")
	err := APIListAuditLogs(env.auditLog)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPIExportAuditLogs(t *testing.T) {
	env := setupTestEnv(t)

	env.auditLog.Log(audit.Entry{Actor: "admin", Action: "test", IPAddress: "10.0.0.1"})

	req, rec := jsonRequest(http.MethodGet, "/api/v1/audit-logs/export", nil)
	c := env.echo.NewContext(req, rec)
	err := APIExportAuditLogs(env.auditLog)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "spreadsheetml")
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "audit-logs.xlsx")
	assert.Greater(t, rec.Body.Len(), 0)
}

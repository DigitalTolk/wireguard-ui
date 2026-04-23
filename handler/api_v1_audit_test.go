package handler

import (
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
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

func TestAPIExportAuditLogs_Empty(t *testing.T) {
	env := setupTestEnv(t)

	req, rec := jsonRequest(http.MethodGet, "/api/v1/audit-logs/export", nil)
	c := env.echo.NewContext(req, rec)
	err := APIExportAuditLogs(env.auditLog)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "spreadsheetml")
}

func TestAPIExportAuditLogs_WithFilters(t *testing.T) {
	env := setupTestEnv(t)

	env.auditLog.Log(audit.Entry{Actor: "admin", Action: "user.create", IPAddress: "10.0.0.1"})
	env.auditLog.Log(audit.Entry{Actor: "user1", Action: "client.create", IPAddress: "10.0.0.2"})

	req, rec := jsonRequest(http.MethodGet, "/api/v1/audit-logs/export?actor=admin", nil)
	c := env.echo.NewContext(req, rec)
	c.QueryParams().Set("actor", "admin")
	err := APIExportAuditLogs(env.auditLog)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPIAuditLogFilters(t *testing.T) {
	env := setupTestEnv(t)

	env.auditLog.Log(audit.Entry{Actor: "admin", Action: "user.create", IPAddress: "10.0.0.1"})
	env.auditLog.Log(audit.Entry{Actor: "manager", Action: "client.create", IPAddress: "10.0.0.2"})
	env.auditLog.Log(audit.Entry{Actor: "admin", Action: "client.delete", IPAddress: "10.0.0.1"})

	req, rec := jsonRequest(http.MethodGet, "/api/v1/audit-logs/filters", nil)
	c := env.echo.NewContext(req, rec)
	err := APIAuditLogFilters(env.auditLog)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]interface{}
	parseJSON(t, rec, &result)
	actors := result["actors"].([]interface{})
	actions := result["actions"].([]interface{})
	assert.Len(t, actors, 2)
	assert.Len(t, actions, 3)
}

func TestAPIAuditLogFilters_Empty(t *testing.T) {
	env := setupTestEnv(t)

	req, rec := jsonRequest(http.MethodGet, "/api/v1/audit-logs/filters", nil)
	c := env.echo.NewContext(req, rec)
	err := APIAuditLogFilters(env.auditLog)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]interface{}
	parseJSON(t, rec, &result)
	// actors and actions may be null (nil slices)
	assert.NotNil(t, result)
}

func TestAPIListAuditLogs_WithSearch(t *testing.T) {
	env := setupTestEnv(t)

	env.auditLog.Log(audit.Entry{Actor: "admin", Action: "test", ResourceID: "res-abc", IPAddress: "10.0.0.1"})
	env.auditLog.Log(audit.Entry{Actor: "admin", Action: "test", ResourceID: "res-xyz", IPAddress: "10.0.0.1"})

	req, rec := jsonRequest(http.MethodGet, "/api/v1/audit-logs?search=abc", nil)
	c := env.echo.NewContext(req, rec)
	c.QueryParams().Set("search", "abc")
	err := APIListAuditLogs(env.auditLog)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]interface{}
	parseJSON(t, rec, &result)
	assert.Equal(t, float64(1), result["total"])
}

func TestAPIAuditLogFilters_WithPopulatedData(t *testing.T) {
	env := setupTestEnv(t)

	// Add diverse audit entries
	env.auditLog.Log(audit.Entry{Actor: "admin", Action: "user.create", IPAddress: "10.0.0.1"})
	env.auditLog.Log(audit.Entry{Actor: "admin", Action: "client.create", IPAddress: "10.0.0.1"})
	env.auditLog.Log(audit.Entry{Actor: "manager", Action: "client.delete", IPAddress: "10.0.0.2"})
	env.auditLog.Log(audit.Entry{Actor: "viewer", Action: "settings.update", IPAddress: "10.0.0.3"})
	env.auditLog.Log(audit.Entry{Actor: "admin", Action: "server.config.apply", IPAddress: "10.0.0.1"})

	req, rec := jsonRequest(http.MethodGet, "/api/v1/audit-logs/filters", nil)
	c := env.echo.NewContext(req, rec)
	err := APIAuditLogFilters(env.auditLog)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]interface{}
	parseJSON(t, rec, &result)
	actors := result["actors"].([]interface{})
	actions := result["actions"].([]interface{})
	assert.Len(t, actors, 3) // admin, manager, viewer
	assert.Len(t, actions, 5) // user.create, client.create, client.delete, settings.update, server.config.apply
}

func TestAPIListAuditLogs_WithDateRange(t *testing.T) {
	env := setupTestEnv(t)

	env.auditLog.Log(audit.Entry{Actor: "admin", Action: "test", IPAddress: "10.0.0.1"})

	// Use SQLite datetime format (YYYY-MM-DD HH:MM:SS) matching CURRENT_TIMESTAMP format
	from := time.Now().Add(-1 * time.Hour).UTC().Format("2006-01-02 15:04:05")
	to := time.Now().Add(1 * time.Hour).UTC().Format("2006-01-02 15:04:05")
	req, rec := jsonRequest(http.MethodGet, "/api/v1/audit-logs", nil)
	c := env.echo.NewContext(req, rec)
	c.QueryParams().Set("from", from)
	c.QueryParams().Set("to", to)
	err := APIListAuditLogs(env.auditLog)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]interface{}
	parseJSON(t, rec, &result)
	assert.GreaterOrEqual(t, result["total"].(float64), float64(1))
}

func TestAPIAuditLogFilters_DBError(t *testing.T) {
	// Create an audit logger with a closed DB to trigger error
	env := setupTestEnv(t)
	closedDB := env.db.DB()
	closedDB.Close() // close the underlying DB

	brokenLogger := audit.NewLogger(closedDB)

	e := echo.New()
	req, rec := jsonRequest(http.MethodGet, "/api/v1/audit-logs/filters", nil)
	c := e.NewContext(req, rec)
	err := APIAuditLogFilters(brokenLogger)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAPIListAuditLogs_DBError(t *testing.T) {
	env := setupTestEnv(t)
	closedDB := env.db.DB()
	closedDB.Close()

	brokenLogger := audit.NewLogger(closedDB)

	e := echo.New()
	req, rec := jsonRequest(http.MethodGet, "/api/v1/audit-logs", nil)
	c := e.NewContext(req, rec)
	err := APIListAuditLogs(brokenLogger)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAPIExportAuditLogs_DBError(t *testing.T) {
	env := setupTestEnv(t)
	closedDB := env.db.DB()
	closedDB.Close()

	brokenLogger := audit.NewLogger(closedDB)

	e := echo.New()
	req, rec := jsonRequest(http.MethodGet, "/api/v1/audit-logs/export", nil)
	c := e.NewContext(req, rec)
	err := APIExportAuditLogs(brokenLogger)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAPIListAuditLogs_WithActionFilter(t *testing.T) {
	env := setupTestEnv(t)

	env.auditLog.Log(audit.Entry{Actor: "admin", Action: "user.create", IPAddress: "10.0.0.1"})
	env.auditLog.Log(audit.Entry{Actor: "admin", Action: "client.delete", IPAddress: "10.0.0.1"})

	req, rec := jsonRequest(http.MethodGet, "/api/v1/audit-logs?action=client.delete", nil)
	c := env.echo.NewContext(req, rec)
	c.QueryParams().Set("action", "client.delete")
	err := APIListAuditLogs(env.auditLog)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]interface{}
	parseJSON(t, rec, &result)
	assert.Equal(t, float64(1), result["total"])
}

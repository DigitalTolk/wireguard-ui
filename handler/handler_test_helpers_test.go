package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/DigitalTolk/wireguard-ui/audit"
	"github.com/DigitalTolk/wireguard-ui/store/sqlitedb"
	"github.com/DigitalTolk/wireguard-ui/util"
)

type testEnv struct {
	db       *sqlitedb.SqliteDB
	auditLog *audit.Logger
	echo     *echo.Echo
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	os.Setenv("WGUI_ENDPOINT_ADDRESS", "10.0.0.1")

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := sqlitedb.New(dbPath)
	require.NoError(t, err)
	require.NoError(t, db.Init())

	auditLog := audit.NewLogger(db.DB())

	e := echo.New()
	secret := [64]byte{}
	copy(secret[:], "testsecret")
	cookieStore := sessions.NewCookieStore(secret[:32], secret[32:])
	cookieStore.Options.HttpOnly = true
	e.Use(session.Middleware(cookieStore))
	e.Use(WithAuditLogger(auditLog))

	util.DisableLogin = true // simplify testing

	return &testEnv{db: db, auditLog: auditLog, echo: e}
}

func jsonRequest(method, path string, body interface{}) (*http.Request, *httptest.ResponseRecorder) {
	var reqBody string
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = string(b)
	}
	req := httptest.NewRequest(method, path, strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	return req, rec
}

func parseJSON(t *testing.T, rec *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	err := json.NewDecoder(rec.Body).Decode(v)
	require.NoError(t, err)
}

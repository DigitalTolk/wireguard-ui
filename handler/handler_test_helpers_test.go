package handler

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/DigitalTolk/wireguard-ui/audit"
	"github.com/DigitalTolk/wireguard-ui/model"
	"github.com/DigitalTolk/wireguard-ui/store/sqlitedb"
	"github.com/DigitalTolk/wireguard-ui/util"
)

// errStore is a mock store that returns errors for all read methods.
// This is used to test error paths in handler functions.
type errStore struct{}

func (e *errStore) Init() error                                              { return fmt.Errorf("db error") }
func (e *errStore) GetUsers() ([]model.User, error)                         { return nil, fmt.Errorf("db error") }
func (e *errStore) GetUserByName(string) (model.User, error)                { return model.User{}, fmt.Errorf("db error") }
func (e *errStore) GetUserByOIDCSub(string) (model.User, error)             { return model.User{}, fmt.Errorf("db error") }
func (e *errStore) SaveUser(model.User) error                               { return fmt.Errorf("db error") }
func (e *errStore) DeleteUser(string) error                                  { return fmt.Errorf("db error") }
func (e *errStore) GetGlobalSettings() (model.GlobalSetting, error)         { return model.GlobalSetting{}, fmt.Errorf("db error") }
func (e *errStore) GetServer() (model.Server, error)                        { return model.Server{}, fmt.Errorf("db error") }
func (e *errStore) GetClients(bool) ([]model.ClientData, error)             { return nil, fmt.Errorf("db error") }
func (e *errStore) GetClientByID(string, model.QRCodeSettings) (model.ClientData, error) {
	return model.ClientData{}, fmt.Errorf("db error")
}
func (e *errStore) SaveClient(model.Client) error                           { return fmt.Errorf("db error") }
func (e *errStore) DeleteClient(string) error                               { return fmt.Errorf("db error") }
func (e *errStore) SaveServerInterface(model.ServerInterface) error          { return fmt.Errorf("db error") }
func (e *errStore) SaveServerKeyPair(model.ServerKeypair) error              { return fmt.Errorf("db error") }
func (e *errStore) SaveGlobalSettings(model.GlobalSetting) error             { return fmt.Errorf("db error") }
func (e *errStore) GetAllocatedIPs(string) ([]string, error)                { return nil, fmt.Errorf("db error") }
func (e *errStore) GetWakeOnLanHosts() ([]model.WakeOnLanHost, error)       { return nil, fmt.Errorf("db error") }
func (e *errStore) GetWakeOnLanHost(string) (*model.WakeOnLanHost, error)   { return nil, fmt.Errorf("db error") }
func (e *errStore) DeleteWakeOnHostLanHost(string) error                     { return fmt.Errorf("db error") }
func (e *errStore) SaveWakeOnLanHost(model.WakeOnLanHost) error              { return fmt.Errorf("db error") }
func (e *errStore) DeleteWakeOnHost(model.WakeOnLanHost) error               { return fmt.Errorf("db error") }
func (e *errStore) GetPath() string                                          { return "/tmp" }
func (e *errStore) SaveHashes(model.ClientServerHashes) error                { return fmt.Errorf("db error") }
func (e *errStore) GetHashes() (model.ClientServerHashes, error)             { return model.ClientServerHashes{}, fmt.Errorf("db error") }

// saveFailStore is a mock where reads succeed but writes fail.
// Used to test error paths where a lookup succeeds but saving fails.
type saveFailStore struct {
	user model.User // user returned by GetUserByName
}

func (s *saveFailStore) Init() error                                              { return nil }
func (s *saveFailStore) GetUsers() ([]model.User, error)                         { return []model.User{s.user}, nil }
func (s *saveFailStore) GetUserByName(string) (model.User, error)                { return s.user, nil }
func (s *saveFailStore) GetUserByOIDCSub(string) (model.User, error)             { return s.user, nil }
func (s *saveFailStore) SaveUser(model.User) error                               { return fmt.Errorf("save error") }
func (s *saveFailStore) DeleteUser(string) error                                  { return fmt.Errorf("save error") }
func (s *saveFailStore) GetGlobalSettings() (model.GlobalSetting, error)         { return model.GlobalSetting{}, nil }
func (s *saveFailStore) GetServer() (model.Server, error)                        { return model.Server{}, nil }
func (s *saveFailStore) GetClients(bool) ([]model.ClientData, error)             { return nil, nil }
func (s *saveFailStore) GetClientByID(string, model.QRCodeSettings) (model.ClientData, error) {
	return model.ClientData{}, nil
}
func (s *saveFailStore) SaveClient(model.Client) error                           { return fmt.Errorf("save error") }
func (s *saveFailStore) DeleteClient(string) error                               { return fmt.Errorf("save error") }
func (s *saveFailStore) SaveServerInterface(model.ServerInterface) error          { return fmt.Errorf("save error") }
func (s *saveFailStore) SaveServerKeyPair(model.ServerKeypair) error              { return fmt.Errorf("save error") }
func (s *saveFailStore) SaveGlobalSettings(model.GlobalSetting) error             { return fmt.Errorf("save error") }
func (s *saveFailStore) GetAllocatedIPs(string) ([]string, error)                { return nil, nil }
func (s *saveFailStore) GetWakeOnLanHosts() ([]model.WakeOnLanHost, error)       { return nil, nil }
func (s *saveFailStore) GetWakeOnLanHost(string) (*model.WakeOnLanHost, error)   { return nil, nil }
func (s *saveFailStore) DeleteWakeOnHostLanHost(string) error                     { return fmt.Errorf("save error") }
func (s *saveFailStore) SaveWakeOnLanHost(model.WakeOnLanHost) error              { return fmt.Errorf("save error") }
func (s *saveFailStore) DeleteWakeOnHost(model.WakeOnLanHost) error               { return fmt.Errorf("save error") }
func (s *saveFailStore) GetPath() string                                          { return "/tmp" }
func (s *saveFailStore) SaveHashes(model.ClientServerHashes) error                { return fmt.Errorf("save error") }
func (s *saveFailStore) GetHashes() (model.ClientServerHashes, error)             { return model.ClientServerHashes{}, nil }

type testEnv struct {
	db       *sqlitedb.SqliteDB
	auditLog *audit.Logger
	echo     *echo.Echo
	cw       *ConfigWriter
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

	// config writer with very long delay so tests don't trigger real writes
	tmplFS := fs.FS(os.DirFS(filepath.Join("..", "templates")))
	cw := NewConfigWriter(db, tmplFS, 24*time.Hour)

	// set config file path to temp dir so any accidental writes don't fail
	gs, _ := db.GetGlobalSettings()
	gs.ConfigFilePath = filepath.Join(dir, "wg0.conf")
	db.SaveGlobalSettings(gs)

	return &testEnv{db: db, auditLog: auditLog, echo: e, cw: cw}
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

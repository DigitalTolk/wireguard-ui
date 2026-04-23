package handler

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DigitalTolk/wireguard-ui/store/sqlitedb"
)

func TestConfigWriter_Trigger(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := sqlitedb.New(dbPath)
	require.NoError(t, err)
	require.NoError(t, db.Init())

	// Point config file to temp dir
	gs, _ := db.GetGlobalSettings()
	gs.ConfigFilePath = filepath.Join(dir, "wg0.conf")
	db.SaveGlobalSettings(gs)

	tmplFS := os.DirFS("../templates")
	cw := NewConfigWriter(db, tmplFS, 100*time.Millisecond)

	cw.Trigger()

	// Wait for the debounce + a little extra
	time.Sleep(300 * time.Millisecond)

	_, err = os.Stat(filepath.Join(dir, "wg0.conf"))
	assert.NoError(t, err, "Config file should be written after debounce")
}

func TestConfigWriter_Debounce(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := sqlitedb.New(dbPath)
	require.NoError(t, err)
	require.NoError(t, db.Init())

	gs, _ := db.GetGlobalSettings()
	gs.ConfigFilePath = filepath.Join(dir, "wg0.conf")
	db.SaveGlobalSettings(gs)

	tmplFS := os.DirFS("../templates")
	cw := NewConfigWriter(db, tmplFS, 200*time.Millisecond)

	// Trigger rapidly — should coalesce
	cw.Trigger()
	time.Sleep(50 * time.Millisecond)
	cw.Trigger()
	time.Sleep(50 * time.Millisecond)
	cw.Trigger()

	// At this point, file should NOT exist yet (debounce hasn't fired)
	_, err = os.Stat(filepath.Join(dir, "wg0.conf"))
	assert.True(t, os.IsNotExist(err), "Config should not be written during debounce window")

	// Wait for the final debounce to fire
	time.Sleep(400 * time.Millisecond)

	_, err = os.Stat(filepath.Join(dir, "wg0.conf"))
	assert.NoError(t, err, "Config file should be written after debounce settles")
}

func TestConfigWriter_ApplyNow_CancelsPendingTimer(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := sqlitedb.New(dbPath)
	require.NoError(t, err)
	require.NoError(t, db.Init())

	gs, _ := db.GetGlobalSettings()
	gs.ConfigFilePath = filepath.Join(dir, "wg0.conf")
	db.SaveGlobalSettings(gs)

	tmplFS := os.DirFS("../templates")
	cw := NewConfigWriter(db, tmplFS, 24*time.Hour)

	// Trigger a debounced write (won't fire for 24h)
	cw.Trigger()

	// ApplyNow should cancel the pending timer and write immediately
	err = cw.ApplyNow()
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "wg0.conf"))
	assert.NoError(t, err, "ApplyNow should write config immediately even with pending timer")
}

func TestConfigWriter_ApplyNow_NoTimer(t *testing.T) {
	// Test ApplyNow when no timer has been set (cw.timer is nil)
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := sqlitedb.New(dbPath)
	require.NoError(t, err)
	require.NoError(t, db.Init())

	gs, _ := db.GetGlobalSettings()
	gs.ConfigFilePath = filepath.Join(dir, "wg0.conf")
	db.SaveGlobalSettings(gs)

	tmplFS := os.DirFS("../templates")
	cw := NewConfigWriter(db, tmplFS, 24*time.Hour)

	// ApplyNow without ever calling Trigger first (timer is nil)
	err = cw.ApplyNow()
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "wg0.conf"))
	assert.NoError(t, err, "ApplyNow should write config even when no timer was set")
}

func TestConfigWriter_Apply_InvalidConfigPath(t *testing.T) {
	// Test apply() when config file path is unwritable
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := sqlitedb.New(dbPath)
	require.NoError(t, err)
	require.NoError(t, db.Init())

	gs, _ := db.GetGlobalSettings()
	gs.ConfigFilePath = "/dev/null/impossible/path/wg0.conf"
	db.SaveGlobalSettings(gs)

	tmplFS := os.DirFS("../templates")
	cw := NewConfigWriter(db, tmplFS, 100*time.Millisecond)

	err = cw.ApplyNow()
	assert.Error(t, err, "ApplyNow should fail when config path is unwritable")
}

func TestConfigWriter_Trigger_ErrorInApply(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := sqlitedb.New(dbPath)
	require.NoError(t, err)
	require.NoError(t, db.Init())

	gs, _ := db.GetGlobalSettings()
	gs.ConfigFilePath = "/dev/null/impossible/path/wg0.conf"
	db.SaveGlobalSettings(gs)

	tmplFS := os.DirFS("../templates")
	cw := NewConfigWriter(db, tmplFS, 100*time.Millisecond)

	// Trigger with a path that will cause apply() to fail
	cw.Trigger()

	// Wait for debounce to fire — apply() will fail and log the error
	time.Sleep(300 * time.Millisecond)

	// No assertion on error since it's logged, not returned.
	// This test exercises the error path inside the AfterFunc callback.
}

func TestConfigWriter_TriggerMultipleTimes(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := sqlitedb.New(dbPath)
	require.NoError(t, err)
	require.NoError(t, db.Init())

	gs, _ := db.GetGlobalSettings()
	gs.ConfigFilePath = filepath.Join(dir, "wg0.conf")
	db.SaveGlobalSettings(gs)

	tmplFS := os.DirFS("../templates")
	cw := NewConfigWriter(db, tmplFS, 100*time.Millisecond)

	// Trigger multiple times rapidly, then let debounce settle
	for i := 0; i < 10; i++ {
		cw.Trigger()
	}

	// Wait for debounce to fire
	time.Sleep(300 * time.Millisecond)

	_, err = os.Stat(filepath.Join(dir, "wg0.conf"))
	assert.NoError(t, err, "Config should be written after rapid triggers settle")
}

func TestConfigWriter_ApplyNow(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := sqlitedb.New(dbPath)
	require.NoError(t, err)
	require.NoError(t, db.Init())

	gs, _ := db.GetGlobalSettings()
	gs.ConfigFilePath = filepath.Join(dir, "wg0.conf")
	db.SaveGlobalSettings(gs)

	tmplFS := os.DirFS("../templates")
	cw := NewConfigWriter(db, tmplFS, 24*time.Hour) // very long debounce

	// ApplyNow should write immediately
	err = cw.ApplyNow()
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "wg0.conf"))
	assert.NoError(t, err, "ApplyNow should write config immediately")
}

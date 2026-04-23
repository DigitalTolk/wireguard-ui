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

package handler

import (
	"io/fs"
	"sync"
	"time"

	"github.com/labstack/gommon/log"

	"github.com/DigitalTolk/wireguard-ui/store"
	"github.com/DigitalTolk/wireguard-ui/util"
)

// ConfigWriter debounces WireGuard config writes so that rapid successive
// mutations (create, edit, delete, enable/disable) produce only one file write
// after a quiet period. This prevents overwhelming systemd path watchers
// (e.g. wgui.path with PathChanged) that restart wg-quick on every change.
type ConfigWriter struct {
	mu      sync.Mutex // protects timer
	writeMu sync.Mutex // serializes actual file writes
	timer   *time.Timer
	delay   time.Duration
	db      store.IStore
	tmplDir fs.FS
}

// NewConfigWriter creates a debounced config writer. The delay parameter
// controls how long to wait after the last Trigger() before writing.
// A typical value is 2 seconds — long enough to coalesce rapid changes,
// short enough that the config is applied promptly.
func NewConfigWriter(db store.IStore, tmplDir fs.FS, delay time.Duration) *ConfigWriter {
	return &ConfigWriter{db: db, tmplDir: tmplDir, delay: delay}
}

// Trigger schedules a config write after the debounce delay. If called again
// before the delay expires, the timer resets. This is non-blocking.
func (cw *ConfigWriter) Trigger() {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	if cw.timer != nil {
		cw.timer.Stop()
	}
	cw.timer = time.AfterFunc(cw.delay, func() {
		if err := cw.apply(); err != nil {
			log.Errorf("Auto-apply config failed: %v", err)
		}
	})
}

// ApplyNow cancels any pending debounced write and writes immediately.
// Returns an error if the write fails.
func (cw *ConfigWriter) ApplyNow() error {
	cw.mu.Lock()
	if cw.timer != nil {
		cw.timer.Stop()
		cw.timer = nil
	}
	cw.mu.Unlock()
	return cw.apply()
}

func (cw *ConfigWriter) apply() error {
	cw.writeMu.Lock()
	defer cw.writeMu.Unlock()

	server, err := cw.db.GetServer()
	if err != nil {
		return err
	}
	clients, err := cw.db.GetClients(false)
	if err != nil {
		return err
	}
	users, err := cw.db.GetUsers()
	if err != nil {
		return err
	}
	settings, err := cw.db.GetGlobalSettings()
	if err != nil {
		return err
	}

	if err := util.WriteWireGuardServerConfig(cw.tmplDir, server, clients, users, settings); err != nil {
		return err
	}

	if err := util.UpdateHashes(cw.db); err != nil {
		log.Warnf("Config written but hash update failed: %v", err)
	}

	log.Info("WireGuard config applied")
	return nil
}

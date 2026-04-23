package sqlitedb

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/DigitalTolk/wireguard-ui/model"
	"github.com/labstack/gommon/log"
)

// MigrateFromJSON checks if an old JSON database exists and migrates data to SQLite.
// It detects the old DB by checking for ./db/server/ directory.
// After migration, it renames the old DB directory to ./db.json.bak/
func MigrateFromJSON(sqliteDB *SqliteDB, jsonDBPath string) error {
	serverDir := filepath.Join(jsonDBPath, "server")
	if _, err := os.Stat(serverDir); os.IsNotExist(err) {
		return nil // no old JSON DB to migrate
	}

	log.Info("Found legacy JSON database, starting migration to SQLite...")

	// migrate server interface
	if err := migrateServerInterface(sqliteDB, jsonDBPath); err != nil {
		return fmt.Errorf("migrate server interface: %w", err)
	}

	// migrate server keypair
	if err := migrateServerKeypair(sqliteDB, jsonDBPath); err != nil {
		return fmt.Errorf("migrate server keypair: %w", err)
	}

	// migrate global settings
	if err := migrateGlobalSettings(sqliteDB, jsonDBPath); err != nil {
		return fmt.Errorf("migrate global settings: %w", err)
	}

	// migrate hashes
	if err := migrateHashes(sqliteDB, jsonDBPath); err != nil {
		return fmt.Errorf("migrate hashes: %w", err)
	}

	// migrate users
	if err := migrateUsers(sqliteDB, jsonDBPath); err != nil {
		return fmt.Errorf("migrate users: %w", err)
	}

	// migrate clients
	if err := migrateClients(sqliteDB, jsonDBPath); err != nil {
		return fmt.Errorf("migrate clients: %w", err)
	}

	// migrate wake-on-lan hosts
	if err := migrateWakeOnLanHosts(sqliteDB, jsonDBPath); err != nil {
		return fmt.Errorf("migrate wake-on-lan hosts: %w", err)
	}

	// rename old DB directory
	backupPath := jsonDBPath + ".json.bak"
	if err := os.Rename(jsonDBPath, backupPath); err != nil {
		log.Warnf("Could not rename old JSON DB directory: %v", err)
		log.Warn("Old data remains at:", jsonDBPath)
	} else {
		log.Infof("Legacy JSON database backed up to %s", backupPath)
	}

	log.Info("Migration to SQLite completed successfully")
	return nil
}

func readJSONFile(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// legacy structs handle the old `,string` JSON tags from the original codebase
type legacyServerInterface struct {
	Addresses  []string  `json:"addresses"`
	ListenPort int       `json:"listen_port,string"`
	UpdatedAt  time.Time `json:"updated_at"`
	PostUp     string    `json:"post_up"`
	PreDown    string    `json:"pre_down"`
	PostDown   string    `json:"post_down"`
}

type legacyGlobalSetting struct {
	EndpointAddress     string    `json:"endpoint_address"`
	DNSServers          []string  `json:"dns_servers"`
	MTU                 int       `json:"mtu,string"`
	PersistentKeepalive int       `json:"persistent_keepalive,string"`
	FirewallMark        string    `json:"firewall_mark"`
	Table               string    `json:"table"`
	ConfigFilePath      string    `json:"config_file_path"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func migrateServerInterface(db *SqliteDB, jsonDBPath string) error {
	filePath := filepath.Join(jsonDBPath, "server", "interfaces.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	var iface legacyServerInterface
	if err := readJSONFile(filePath, &iface); err != nil {
		return err
	}

	addrJSON, _ := json.Marshal(iface.Addresses)
	_, err := db.db.Exec(
		`INSERT OR REPLACE INTO server_interface (id, addresses, listen_port, post_up, pre_down, post_down, updated_at)
		 VALUES (1, ?, ?, ?, ?, ?, ?)`,
		string(addrJSON), iface.ListenPort, iface.PostUp, iface.PreDown, iface.PostDown, iface.UpdatedAt,
	)
	if err != nil {
		return err
	}
	log.Info("  Migrated server interface")
	return nil
}

func migrateServerKeypair(db *SqliteDB, jsonDBPath string) error {
	filePath := filepath.Join(jsonDBPath, "server", "keypair.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	var kp model.ServerKeypair
	if err := readJSONFile(filePath, &kp); err != nil {
		return err
	}

	_, err := db.db.Exec(
		`INSERT OR REPLACE INTO server_keypair (id, private_key, public_key, updated_at) VALUES (1, ?, ?, ?)`,
		kp.PrivateKey, kp.PublicKey, kp.UpdatedAt,
	)
	if err != nil {
		return err
	}
	log.Info("  Migrated server keypair")
	return nil
}

func migrateGlobalSettings(db *SqliteDB, jsonDBPath string) error {
	filePath := filepath.Join(jsonDBPath, "server", "global_settings.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	var gs legacyGlobalSetting
	if err := readJSONFile(filePath, &gs); err != nil {
		return err
	}

	dnsJSON, _ := json.Marshal(gs.DNSServers)
	_, err := db.db.Exec(
		`INSERT OR REPLACE INTO global_settings (id, endpoint_address, dns_servers, mtu, persistent_keepalive, firewall_mark, "table", config_file_path, updated_at)
		 VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?)`,
		gs.EndpointAddress, string(dnsJSON), gs.MTU, gs.PersistentKeepalive,
		gs.FirewallMark, gs.Table, gs.ConfigFilePath, gs.UpdatedAt,
	)
	if err != nil {
		return err
	}
	log.Info("  Migrated global settings")
	return nil
}

func migrateHashes(db *SqliteDB, jsonDBPath string) error {
	filePath := filepath.Join(jsonDBPath, "server", "hashes.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	var h model.ClientServerHashes
	if err := readJSONFile(filePath, &h); err != nil {
		return err
	}

	_, err := db.db.Exec(
		`INSERT OR REPLACE INTO hashes (id, client, server) VALUES (1, ?, ?)`,
		h.Client, h.Server,
	)
	if err != nil {
		return err
	}
	log.Info("  Migrated hashes")
	return nil
}

func migrateUsers(db *SqliteDB, jsonDBPath string) error {
	usersDir := filepath.Join(jsonDBPath, "users")
	entries, err := os.ReadDir(usersDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		var u model.User
		if err := readJSONFile(filepath.Join(usersDir, entry.Name()), &u); err != nil {
			log.Warnf("  Skipping user file %s: %v", entry.Name(), err)
			continue
		}

		now := time.Now().UTC()
		_, err := db.db.Exec(
			`INSERT OR REPLACE INTO users (username, email, display_name, admin, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			u.Username, u.Email, u.DisplayName, u.Admin, now, now,
		)
		if err != nil {
			return fmt.Errorf("migrate user %s: %w", u.Username, err)
		}
		count++
	}
	log.Infof("  Migrated %d users", count)
	return nil
}

func migrateClients(db *SqliteDB, jsonDBPath string) error {
	clientsDir := filepath.Join(jsonDBPath, "clients")
	entries, err := os.ReadDir(clientsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		var c model.Client
		if err := readJSONFile(filepath.Join(clientsDir, entry.Name()), &c); err != nil {
			log.Warnf("  Skipping client file %s: %v", entry.Name(), err)
			continue
		}

		if err := db.SaveClient(c); err != nil {
			return fmt.Errorf("migrate client %s: %w", c.ID, err)
		}
		count++
	}
	log.Infof("  Migrated %d clients", count)
	return nil
}

func migrateWakeOnLanHosts(db *SqliteDB, jsonDBPath string) error {
	wolDir := filepath.Join(jsonDBPath, "wake_on_lan_hosts")
	entries, err := os.ReadDir(wolDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		var h model.WakeOnLanHost
		if err := readJSONFile(filepath.Join(wolDir, entry.Name()), &h); err != nil {
			log.Warnf("  Skipping WoL host file %s: %v", entry.Name(), err)
			continue
		}

		if err := db.SaveWakeOnLanHost(h); err != nil {
			return fmt.Errorf("migrate WoL host %s: %w", h.MacAddress, err)
		}
		count++
	}
	log.Infof("  Migrated %d Wake-on-LAN hosts", count)
	return nil
}

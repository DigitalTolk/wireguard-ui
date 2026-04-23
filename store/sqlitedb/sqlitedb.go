package sqlitedb

import (
	"database/sql"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/labstack/gommon/log"
	_ "modernc.org/sqlite"

	"github.com/skip2/go-qrcode"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/DigitalTolk/wireguard-ui/model"
	"github.com/DigitalTolk/wireguard-ui/util"
)

const (
	userColumns         = "username, email, display_name, COALESCE(oidc_sub, ''), admin, created_at, updated_at"
	qrCodeDataURIPrefix = "data:image/png;base64,"
)

//go:embed schema.sql
var schemaFS embed.FS

// SqliteDB implements store.IStore using SQLite
type SqliteDB struct {
	db     *sql.DB
	dbPath string
}

// New creates a new SqliteDB instance
func New(dbPath string) (*SqliteDB, error) {
	// ensure parent directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("cannot create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("cannot open database: %w", err)
	}

	// apply schema
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return nil, fmt.Errorf("cannot read schema: %w", err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		return nil, fmt.Errorf("cannot apply schema: %w", err)
	}

	return &SqliteDB{db: db, dbPath: dbPath}, nil
}

// migrate applies incremental schema changes to existing databases
func (o *SqliteDB) migrate() {
	log.Info("Running database migrations...")

	if _, err := o.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_clients_name ON clients(name)`); err != nil {
		log.Warnf("migrate: create idx_clients_name: %v", err)
	}
	if _, err := o.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_clients_public_key ON clients(public_key) WHERE public_key != ''`); err != nil {
		log.Warnf("migrate: create idx_clients_public_key: %v", err)
	}

	// remove legacy password-only users (cannot log in with SSO-only auth)
	res, _ := o.db.Exec(`DELETE FROM users WHERE oidc_sub IS NULL OR oidc_sub = ''`)
	if n, _ := res.RowsAffected(); n > 0 {
		log.Infof("migrate: removed %d legacy user(s) without OIDC subject", n)
	}

	// if no admin exists after cleanup, promote the first user
	var adminCount int
	o.db.QueryRow(`SELECT COUNT(*) FROM users WHERE admin = 1`).Scan(&adminCount)
	if adminCount == 0 {
		var username string
		if o.db.QueryRow(`SELECT username FROM users ORDER BY created_at ASC LIMIT 1`).Scan(&username) == nil {
			o.db.Exec(`UPDATE users SET admin = 1 WHERE username = ?`, username)
			log.Infof("migrate: promoted user %s to admin (no admin existed)", username)
		}
	}

	// derive missing public keys from private keys
	type keyPair struct{ id, privKey string }
	var missing []keyPair
	rows, err := o.db.Query(`SELECT id, private_key FROM clients WHERE public_key = '' AND private_key != ''`)
	if err == nil {
		for rows.Next() {
			var kp keyPair
			if rows.Scan(&kp.id, &kp.privKey) == nil {
				missing = append(missing, kp)
			}
		}
		rows.Close()
	}
	for _, kp := range missing {
		if key, err := wgtypes.ParseKey(kp.privKey); err == nil {
			o.db.Exec(`UPDATE clients SET public_key = ? WHERE id = ?`, key.PublicKey().String(), kp.id)
			log.Infof("migrate: derived public key for client %s", kp.id)
		}
	}

	log.Info("Database migrations complete")
}

// Init initializes the database with default values if they don't exist
func (o *SqliteDB) Init() error {
	// schema migrations for existing databases
	o.migrate()

	// server interface
	var ifaceCount int
	o.db.QueryRow("SELECT COUNT(*) FROM server_interface").Scan(&ifaceCount)
	if ifaceCount == 0 {
		addresses := util.LookupEnvOrStrings(util.ServerAddressesEnvVar, []string{util.DefaultServerAddress})
		listenPort := util.LookupEnvOrInt(util.ServerListenPortEnvVar, util.DefaultServerPort)
		postUp := util.LookupEnvOrString(util.ServerPostUpScriptEnvVar, "")
		postDown := util.LookupEnvOrString(util.ServerPostDownScriptEnvVar, "")
		addrJSON, _ := json.Marshal(addresses)
		_, err := o.db.Exec(
			`INSERT INTO server_interface (id, addresses, listen_port, post_up, post_down, updated_at) VALUES (1, ?, ?, ?, ?, ?)`,
			string(addrJSON), listenPort, postUp, postDown, time.Now().UTC(),
		)
		if err != nil {
			return fmt.Errorf("cannot init server interface: %w", err)
		}
	}

	// server keypair
	var kpCount int
	o.db.QueryRow("SELECT COUNT(*) FROM server_keypair").Scan(&kpCount)
	if kpCount == 0 {
		key, err := wgtypes.GeneratePrivateKey()
		if err != nil {
			return fmt.Errorf("cannot generate server keypair: %w", err)
		}
		_, err = o.db.Exec(
			`INSERT INTO server_keypair (id, private_key, public_key, updated_at) VALUES (1, ?, ?, ?)`,
			key.String(), key.PublicKey().String(), time.Now().UTC(),
		)
		if err != nil {
			return fmt.Errorf("cannot init server keypair: %w", err)
		}
	}

	// global settings
	var gsCount int
	o.db.QueryRow("SELECT COUNT(*) FROM global_settings").Scan(&gsCount)
	if gsCount == 0 {
		endpointAddress := util.LookupEnvOrString(util.EndpointAddressEnvVar, "")
		if endpointAddress == "" {
			publicInterface, err := util.GetPublicIP()
			if err != nil {
				return fmt.Errorf("cannot detect public IP: %w", err)
			}
			endpointAddress = publicInterface.IPAddress
		}
		dnsServers := util.LookupEnvOrStrings(util.DNSEnvVar, []string{util.DefaultDNS})
		dnsJSON, _ := json.Marshal(dnsServers)
		_, err := o.db.Exec(
			`INSERT INTO global_settings (id, endpoint_address, dns_servers, mtu, persistent_keepalive, firewall_mark, "table", config_file_path, updated_at)
			 VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?)`,
			endpointAddress,
			string(dnsJSON),
			util.LookupEnvOrInt(util.MTUEnvVar, util.DefaultMTU),
			util.LookupEnvOrInt(util.PersistentKeepaliveEnvVar, util.DefaultPersistentKeepalive),
			util.LookupEnvOrString(util.FirewallMarkEnvVar, util.DefaultFirewallMark),
			util.LookupEnvOrString(util.TableEnvVar, util.DefaultTable),
			util.LookupEnvOrString(util.ConfigFilePathEnvVar, util.DefaultConfigFilePath),
			time.Now().UTC(),
		)
		if err != nil {
			return fmt.Errorf("cannot init global settings: %w", err)
		}
	}

	// hashes
	var hashCount int
	o.db.QueryRow("SELECT COUNT(*) FROM hashes").Scan(&hashCount)
	if hashCount == 0 {
		o.db.Exec(`INSERT INTO hashes (id, client, server) VALUES (1, 'none', 'none')`)
	}

	// init caches (first OIDC login auto-provisions admin user)
	users, err := o.GetUsers()
	if err == nil {
		util.DBUsersToCRC32Mutex.Lock()
		for _, user := range users {
			util.DBUsersToCRC32[user.Username] = util.GetDBUserCRC32(user)
		}
		util.DBUsersToCRC32Mutex.Unlock()
	}

	return nil
}

// GetUsers returns all users
func (o *SqliteDB) GetUsers() ([]model.User, error) {
	rows, err := o.db.Query("SELECT " + userColumns + " FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.Username, &u.Email, &u.DisplayName, &u.OIDCSub, &u.Admin, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// GetUserByName returns a single user by username
func (o *SqliteDB) GetUserByName(username string) (model.User, error) {
	var u model.User
	err := o.db.QueryRow(
		"SELECT "+userColumns+" FROM users WHERE username = ?",
		username,
	).Scan(&u.Username, &u.Email, &u.DisplayName, &u.OIDCSub, &u.Admin, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return u, err
	}
	return u, nil
}

// SaveUser creates or updates a user
func (o *SqliteDB) SaveUser(user model.User) error {
	now := time.Now().UTC()
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}

	_, err := o.db.Exec(
		`INSERT INTO users (username, email, display_name, oidc_sub, admin, created_at, updated_at)
		 VALUES (?, ?, ?, NULLIF(?, ''), ?, ?, ?)
		 ON CONFLICT(username) DO UPDATE SET
		   email = excluded.email,
		   display_name = excluded.display_name,
		   oidc_sub = excluded.oidc_sub,
		   admin = excluded.admin,
		   updated_at = excluded.updated_at`,
		user.Username, user.Email, user.DisplayName, user.OIDCSub, user.Admin, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return err
	}
	util.DBUsersToCRC32Mutex.Lock()
	util.DBUsersToCRC32[user.Username] = util.GetDBUserCRC32(user)
	util.DBUsersToCRC32Mutex.Unlock()
	return nil
}

// DeleteUser removes a user by username
func (o *SqliteDB) DeleteUser(username string) error {
	util.DBUsersToCRC32Mutex.Lock()
	delete(util.DBUsersToCRC32, username)
	util.DBUsersToCRC32Mutex.Unlock()
	_, err := o.db.Exec("DELETE FROM users WHERE username = ?", username)
	return err
}

// GetGlobalSettings returns global WireGuard settings
func (o *SqliteDB) GetGlobalSettings() (model.GlobalSetting, error) {
	var gs model.GlobalSetting
	var dnsJSON string
	err := o.db.QueryRow(
		`SELECT endpoint_address, dns_servers, mtu, persistent_keepalive, firewall_mark, "table", config_file_path, updated_at
		 FROM global_settings WHERE id = 1`,
	).Scan(&gs.EndpointAddress, &dnsJSON, &gs.MTU, &gs.PersistentKeepalive, &gs.FirewallMark, &gs.Table, &gs.ConfigFilePath, &gs.UpdatedAt)
	if err != nil {
		return gs, err
	}
	json.Unmarshal([]byte(dnsJSON), &gs.DNSServers)
	return gs, nil
}

// GetServer returns the server config (interface + keypair)
func (o *SqliteDB) GetServer() (model.Server, error) {
	server := model.Server{}

	// interface
	iface := model.ServerInterface{}
	var addrJSON string
	err := o.db.QueryRow(
		"SELECT addresses, listen_port, post_up, pre_down, post_down, updated_at FROM server_interface WHERE id = 1",
	).Scan(&addrJSON, &iface.ListenPort, &iface.PostUp, &iface.PreDown, &iface.PostDown, &iface.UpdatedAt)
	if err != nil {
		return server, fmt.Errorf("cannot read server interface: %w", err)
	}
	json.Unmarshal([]byte(addrJSON), &iface.Addresses)
	server.Interface = &iface

	// keypair
	kp := model.ServerKeypair{}
	err = o.db.QueryRow(
		"SELECT private_key, public_key, updated_at FROM server_keypair WHERE id = 1",
	).Scan(&kp.PrivateKey, &kp.PublicKey, &kp.UpdatedAt)
	if err != nil {
		return server, fmt.Errorf("cannot read server keypair: %w", err)
	}
	server.KeyPair = &kp

	return server, nil
}

// GetClients returns all clients, optionally with QR codes
func (o *SqliteDB) GetClients(hasQRCode bool) ([]model.ClientData, error) {
	rows, err := o.db.Query(
		`SELECT id, private_key, public_key, preshared_key, name, email,
		        subnet_ranges, allocated_ips, allowed_ips, extra_allowed_ips,
		        endpoint, additional_notes, use_server_dns, enabled, created_at, updated_at
		 FROM clients`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// fetch server/settings once outside the loop for QR generation
	var server model.Server
	var globalSettings model.GlobalSetting
	if hasQRCode {
		server, _ = o.GetServer()
		globalSettings, _ = o.GetGlobalSettings()
	}

	var clients []model.ClientData
	for rows.Next() {
		client, err := scanClientFrom(rows)
		if err != nil {
			return nil, err
		}

		clientData := model.ClientData{Client: &client}

		if hasQRCode && client.PrivateKey != "" {
			png, err := qrcode.Encode(util.BuildClientConfig(client, server, globalSettings), qrcode.Medium, 256)
			if err == nil {
				clientData.QRCode = qrCodeDataURIPrefix + base64.StdEncoding.EncodeToString(png)
			}
		}

		clients = append(clients, clientData)
	}
	return clients, rows.Err()
}

// GetClientByID returns a single client by ID
func (o *SqliteDB) GetClientByID(clientID string, qrCodeSettings model.QRCodeSettings) (model.ClientData, error) {
	clientData := model.ClientData{}

	row := o.db.QueryRow(
		`SELECT id, private_key, public_key, preshared_key, name, email,
		        subnet_ranges, allocated_ips, allowed_ips, extra_allowed_ips,
		        endpoint, additional_notes, use_server_dns, enabled, created_at, updated_at
		 FROM clients WHERE id = ?`, clientID,
	)

	client, err := scanClientFrom(row)
	if err != nil {
		return clientData, err
	}

	if qrCodeSettings.Enabled && client.PrivateKey != "" {
		server, _ := o.GetServer()
		globalSettings, _ := o.GetGlobalSettings()
		if !qrCodeSettings.IncludeDNS {
			globalSettings.DNSServers = []string{}
		}
		if !qrCodeSettings.IncludeMTU {
			globalSettings.MTU = 0
		}
		png, err := qrcode.Encode(util.BuildClientConfig(client, server, globalSettings), qrcode.Medium, 256)
		if err == nil {
			clientData.QRCode = qrCodeDataURIPrefix + base64.StdEncoding.EncodeToString(png)
		}
	}

	clientData.Client = &client
	return clientData, nil
}

// SaveClient creates or updates a client
func (o *SqliteDB) SaveClient(client model.Client) error {
	subnetJSON, _ := json.Marshal(client.SubnetRanges)
	allocJSON, _ := json.Marshal(client.AllocatedIPs)
	allowJSON, _ := json.Marshal(client.AllowedIPs)
	extraJSON, _ := json.Marshal(client.ExtraAllowedIPs)

	_, err := o.db.Exec(
		`INSERT INTO clients (id, private_key, public_key, preshared_key, name, email,
		                      subnet_ranges, allocated_ips, allowed_ips, extra_allowed_ips,
		                      endpoint, additional_notes, use_server_dns, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   private_key = excluded.private_key,
		   public_key = excluded.public_key,
		   preshared_key = excluded.preshared_key,
		   name = excluded.name,
		   email = excluded.email,
		   subnet_ranges = excluded.subnet_ranges,
		   allocated_ips = excluded.allocated_ips,
		   allowed_ips = excluded.allowed_ips,
		   extra_allowed_ips = excluded.extra_allowed_ips,
		   endpoint = excluded.endpoint,
		   additional_notes = excluded.additional_notes,
		   use_server_dns = excluded.use_server_dns,
		   enabled = excluded.enabled,
		   updated_at = excluded.updated_at`,
		client.ID, client.PrivateKey, client.PublicKey, client.PresharedKey,
		client.Name, client.Email,
		string(subnetJSON), string(allocJSON), string(allowJSON), string(extraJSON),
		client.Endpoint, client.AdditionalNotes, client.UseServerDNS, client.Enabled,
		client.CreatedAt, client.UpdatedAt,
	)
	return err
}

// DeleteClient removes a client by ID
func (o *SqliteDB) DeleteClient(clientID string) error {
	_, err := o.db.Exec("DELETE FROM clients WHERE id = ?", clientID)
	return err
}

// SaveServerInterface updates the server interface config
func (o *SqliteDB) SaveServerInterface(serverInterface model.ServerInterface) error {
	addrJSON, _ := json.Marshal(serverInterface.Addresses)
	_, err := o.db.Exec(
		`UPDATE server_interface SET addresses = ?, listen_port = ?, post_up = ?, pre_down = ?, post_down = ?, updated_at = ? WHERE id = 1`,
		string(addrJSON), serverInterface.ListenPort, serverInterface.PostUp, serverInterface.PreDown, serverInterface.PostDown, serverInterface.UpdatedAt,
	)
	return err
}

// SaveServerKeyPair updates the server keypair
func (o *SqliteDB) SaveServerKeyPair(serverKeyPair model.ServerKeypair) error {
	_, err := o.db.Exec(
		`UPDATE server_keypair SET private_key = ?, public_key = ?, updated_at = ? WHERE id = 1`,
		serverKeyPair.PrivateKey, serverKeyPair.PublicKey, serverKeyPair.UpdatedAt,
	)
	return err
}

// SaveGlobalSettings updates global settings
func (o *SqliteDB) SaveGlobalSettings(globalSettings model.GlobalSetting) error {
	dnsJSON, _ := json.Marshal(globalSettings.DNSServers)
	_, err := o.db.Exec(
		`UPDATE global_settings SET endpoint_address = ?, dns_servers = ?, mtu = ?, persistent_keepalive = ?,
		 firewall_mark = ?, "table" = ?, config_file_path = ?, updated_at = ? WHERE id = 1`,
		globalSettings.EndpointAddress, string(dnsJSON), globalSettings.MTU, globalSettings.PersistentKeepalive,
		globalSettings.FirewallMark, globalSettings.Table, globalSettings.ConfigFilePath, globalSettings.UpdatedAt,
	)
	return err
}

// GetAllocatedIPs returns all IP addresses allocated to clients and server
func (o *SqliteDB) GetAllocatedIPs(excludeClientID string) ([]string, error) {
	allocatedIPs := make([]string, 0)

	// server addresses
	var addrJSON string
	err := o.db.QueryRow("SELECT addresses FROM server_interface WHERE id = 1").Scan(&addrJSON)
	if err != nil {
		return nil, err
	}
	var serverAddrs []string
	json.Unmarshal([]byte(addrJSON), &serverAddrs)
	for _, cidr := range serverAddrs {
		ip, _, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, err
		}
		allocatedIPs = append(allocatedIPs, ip.String())
	}

	// client addresses
	rows, err := o.db.Query("SELECT allocated_ips FROM clients WHERE id != ?", excludeClientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var ipsJSON string
		if err := rows.Scan(&ipsJSON); err != nil {
			return nil, err
		}
		var ips []string
		json.Unmarshal([]byte(ipsJSON), &ips)
		for _, cidr := range ips {
			ip, _, err := net.ParseCIDR(cidr)
			if err != nil {
				return nil, err
			}
			allocatedIPs = append(allocatedIPs, ip.String())
		}
	}

	return allocatedIPs, rows.Err()
}

// GetPath returns the database file path
func (o *SqliteDB) GetPath() string {
	return filepath.Dir(o.dbPath)
}

// GetHashes returns stored hashes
func (o *SqliteDB) GetHashes() (model.ClientServerHashes, error) {
	var h model.ClientServerHashes
	err := o.db.QueryRow("SELECT client, server FROM hashes WHERE id = 1").Scan(&h.Client, &h.Server)
	return h, err
}

// SaveHashes updates stored hashes
func (o *SqliteDB) SaveHashes(hashes model.ClientServerHashes) error {
	_, err := o.db.Exec("UPDATE hashes SET client = ?, server = ? WHERE id = 1", hashes.Client, hashes.Server)
	return err
}

// DB returns the underlying sql.DB for direct access (e.g., audit logs)
func (o *SqliteDB) DB() *sql.DB {
	return o.db
}

// GetUserByOIDCSub returns a user by their OIDC subject identifier
func (o *SqliteDB) GetUserByOIDCSub(sub string) (model.User, error) {
	var u model.User
	err := o.db.QueryRow(
		"SELECT "+userColumns+" FROM users WHERE oidc_sub = ?", sub,
	).Scan(&u.Username, &u.Email, &u.DisplayName, &u.OIDCSub, &u.Admin, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

// scanner is satisfied by both *sql.Rows and *sql.Row
type scanner interface {
	Scan(dest ...interface{}) error
}

func scanClientFrom(s scanner) (model.Client, error) {
	var c model.Client
	var subnetJSON, allocJSON, allowJSON, extraJSON string
	err := s.Scan(
		&c.ID, &c.PrivateKey, &c.PublicKey, &c.PresharedKey,
		&c.Name, &c.Email,
		&subnetJSON, &allocJSON, &allowJSON, &extraJSON,
		&c.Endpoint, &c.AdditionalNotes, &c.UseServerDNS, &c.Enabled,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return c, err
	}
	json.Unmarshal([]byte(subnetJSON), &c.SubnetRanges)
	json.Unmarshal([]byte(allocJSON), &c.AllocatedIPs)
	json.Unmarshal([]byte(allowJSON), &c.AllowedIPs)
	json.Unmarshal([]byte(extraJSON), &c.ExtraAllowedIPs)
	return c, nil
}

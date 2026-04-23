package sqlitedb

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DigitalTolk/wireguard-ui/model"
	"github.com/DigitalTolk/wireguard-ui/util"
)

func newTestDB(t *testing.T) *SqliteDB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := New(dbPath)
	require.NoError(t, err)
	return db
}

func initTestDB(t *testing.T) *SqliteDB {
	t.Helper()
	// set env vars so Init doesn't try to detect public IP
	os.Setenv("WGUI_ENDPOINT_ADDRESS", "10.0.0.1")
	db := newTestDB(t)
	err := db.Init()
	require.NoError(t, err)
	return db
}

// --- User Tests ---

func TestSaveAndGetUser(t *testing.T) {
	db := newTestDB(t)
	now := time.Now().UTC()

	user := model.User{
		Username:  "testuser",
		Email:     "test@example.com",
		Admin:     true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := db.SaveUser(user)
	require.NoError(t, err)

	got, err := db.GetUserByName("testuser")
	require.NoError(t, err)
	assert.Equal(t, "testuser", got.Username)
	assert.Equal(t, "test@example.com", got.Email)
	assert.True(t, got.Admin)
}

func TestGetUsers(t *testing.T) {
	db := newTestDB(t)
	now := time.Now().UTC()

	db.SaveUser(model.User{Username: "user1", CreatedAt: now, UpdatedAt: now})
	db.SaveUser(model.User{Username: "user2", CreatedAt: now, UpdatedAt: now})

	users, err := db.GetUsers()
	require.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestDeleteUser(t *testing.T) {
	db := newTestDB(t)
	now := time.Now().UTC()

	db.SaveUser(model.User{Username: "delme", CreatedAt: now, UpdatedAt: now})

	err := db.DeleteUser("delme")
	require.NoError(t, err)

	_, err = db.GetUserByName("delme")
	assert.Error(t, err)
}

func TestSaveUser_Upsert(t *testing.T) {
	db := newTestDB(t)
	now := time.Now().UTC()

	db.SaveUser(model.User{Username: "user1", Email: "old@test.com", CreatedAt: now, UpdatedAt: now})
	db.SaveUser(model.User{Username: "user1", Email: "new@test.com", CreatedAt: now, UpdatedAt: now})

	got, err := db.GetUserByName("user1")
	require.NoError(t, err)
	assert.Equal(t, "new@test.com", got.Email)
}

func TestGetUserByName_NotFound(t *testing.T) {
	db := newTestDB(t)
	_, err := db.GetUserByName("nonexistent")
	assert.Error(t, err)
}

func TestSaveUser_ZeroTimestamps(t *testing.T) {
	db := newTestDB(t)

	// Save a user without setting timestamps - they should be auto-filled
	user := model.User{
		Username: "autotime",
		Email:    "auto@test.com",
		Admin:    false,
	}

	err := db.SaveUser(user)
	require.NoError(t, err)

	got, err := db.GetUserByName("autotime")
	require.NoError(t, err)
	assert.False(t, got.CreatedAt.IsZero())
	assert.False(t, got.UpdatedAt.IsZero())
}

func TestSaveUser_WithOIDCSub(t *testing.T) {
	db := newTestDB(t)
	now := time.Now().UTC()

	user := model.User{
		Username:  "oidcuser",
		Email:     "oidc@test.com",
		OIDCSub:   "sub-12345",
		Admin:     false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := db.SaveUser(user)
	require.NoError(t, err)

	got, err := db.GetUserByOIDCSub("sub-12345")
	require.NoError(t, err)
	assert.Equal(t, "oidcuser", got.Username)
}

func TestSaveUser_WithDisplayName(t *testing.T) {
	db := newTestDB(t)
	now := time.Now().UTC()

	user := model.User{
		Username:    "displayuser",
		Email:       "display@test.com",
		DisplayName: "Display Name",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := db.SaveUser(user)
	require.NoError(t, err)

	got, err := db.GetUserByName("displayuser")
	require.NoError(t, err)
	assert.Equal(t, "Display Name", got.DisplayName)
}

// --- Client Tests ---

func TestSaveAndGetClient(t *testing.T) {
	db := newTestDB(t)
	now := time.Now().UTC()

	client := model.Client{
		ID:              "test123",
		Name:            "Test Client",
		Email:           "client@test.com",
		PublicKey:       "pubkey123",
		PrivateKey:      "privkey123",
		PresharedKey:    "psk123",
		AllocatedIPs:    []string{"10.0.0.2/32"},
		AllowedIPs:      []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{},
		SubnetRanges:    []string{},
		UseServerDNS:    true,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	err := db.SaveClient(client)
	require.NoError(t, err)

	got, err := db.GetClientByID("test123", model.QRCodeSettings{Enabled: false})
	require.NoError(t, err)
	assert.Equal(t, "Test Client", got.Client.Name)
	assert.Equal(t, "client@test.com", got.Client.Email)
	assert.Equal(t, []string{"10.0.0.2/32"}, got.Client.AllocatedIPs)
	assert.True(t, got.Client.Enabled)
}

func TestGetClients(t *testing.T) {
	db := initTestDB(t)
	now := time.Now().UTC()

	db.SaveClient(model.Client{ID: "c1", Name: "Client 1", AllocatedIPs: []string{}, AllowedIPs: []string{}, ExtraAllowedIPs: []string{}, SubnetRanges: []string{}, CreatedAt: now, UpdatedAt: now})
	db.SaveClient(model.Client{ID: "c2", Name: "Client 2", AllocatedIPs: []string{}, AllowedIPs: []string{}, ExtraAllowedIPs: []string{}, SubnetRanges: []string{}, CreatedAt: now, UpdatedAt: now})

	clients, err := db.GetClients(false)
	require.NoError(t, err)
	assert.Len(t, clients, 2)
}

func TestDeleteClient(t *testing.T) {
	db := newTestDB(t)
	now := time.Now().UTC()

	db.SaveClient(model.Client{ID: "delclient", Name: "Del", AllocatedIPs: []string{}, AllowedIPs: []string{}, ExtraAllowedIPs: []string{}, SubnetRanges: []string{}, CreatedAt: now, UpdatedAt: now})

	err := db.DeleteClient("delclient")
	require.NoError(t, err)

	_, err = db.GetClientByID("delclient", model.QRCodeSettings{Enabled: false})
	assert.Error(t, err)
}

func TestSaveClient_Upsert(t *testing.T) {
	db := newTestDB(t)
	now := time.Now().UTC()

	db.SaveClient(model.Client{ID: "c1", Name: "Original", AllocatedIPs: []string{}, AllowedIPs: []string{}, ExtraAllowedIPs: []string{}, SubnetRanges: []string{}, CreatedAt: now, UpdatedAt: now})
	db.SaveClient(model.Client{ID: "c1", Name: "Updated", AllocatedIPs: []string{}, AllowedIPs: []string{}, ExtraAllowedIPs: []string{}, SubnetRanges: []string{}, CreatedAt: now, UpdatedAt: now})

	got, err := db.GetClientByID("c1", model.QRCodeSettings{Enabled: false})
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Client.Name)
}

// --- Server Tests ---

func TestGetServer(t *testing.T) {
	db := initTestDB(t)

	server, err := db.GetServer()
	require.NoError(t, err)
	assert.NotNil(t, server.Interface)
	assert.NotNil(t, server.KeyPair)
	assert.NotEmpty(t, server.KeyPair.PublicKey)
	assert.NotEmpty(t, server.KeyPair.PrivateKey)
	assert.Greater(t, server.Interface.ListenPort, 0)
}

func TestSaveServerInterface(t *testing.T) {
	db := initTestDB(t)

	iface := model.ServerInterface{
		Addresses:  []string{"10.0.0.0/24", "fd00::1/64"},
		ListenPort: 12345,
		PostUp:     "iptables -A",
		PostDown:   "iptables -D",
		UpdatedAt:  time.Now().UTC(),
	}

	err := db.SaveServerInterface(iface)
	require.NoError(t, err)

	server, err := db.GetServer()
	require.NoError(t, err)
	assert.Equal(t, []string{"10.0.0.0/24", "fd00::1/64"}, server.Interface.Addresses)
	assert.Equal(t, 12345, server.Interface.ListenPort)
	assert.Equal(t, "iptables -A", server.Interface.PostUp)
}

func TestSaveServerKeyPair(t *testing.T) {
	db := initTestDB(t)

	kp := model.ServerKeypair{
		PrivateKey: "newpriv",
		PublicKey:  "newpub",
		UpdatedAt:  time.Now().UTC(),
	}

	err := db.SaveServerKeyPair(kp)
	require.NoError(t, err)

	server, err := db.GetServer()
	require.NoError(t, err)
	assert.Equal(t, "newpub", server.KeyPair.PublicKey)
	assert.Equal(t, "newpriv", server.KeyPair.PrivateKey)
}

func TestSaveServerInterface_WithPreDown(t *testing.T) {
	db := initTestDB(t)

	iface := model.ServerInterface{
		Addresses:  []string{"10.0.0.0/24"},
		ListenPort: 51820,
		PostUp:     "iptables -A FORWARD -i wg0 -j ACCEPT",
		PreDown:    "iptables -D FORWARD -i wg0 -j ACCEPT",
		PostDown:   "iptables -D FORWARD -i wg0 -j ACCEPT",
		UpdatedAt:  time.Now().UTC(),
	}

	err := db.SaveServerInterface(iface)
	require.NoError(t, err)

	server, err := db.GetServer()
	require.NoError(t, err)
	assert.Equal(t, "iptables -D FORWARD -i wg0 -j ACCEPT", server.Interface.PreDown)
}

// --- Global Settings Tests ---

func TestGetGlobalSettings(t *testing.T) {
	db := initTestDB(t)

	gs, err := db.GetGlobalSettings()
	require.NoError(t, err)
	assert.Equal(t, "10.0.0.1", gs.EndpointAddress)
	assert.NotEmpty(t, gs.DNSServers)
	assert.Greater(t, gs.MTU, 0)
}

func TestSaveGlobalSettings(t *testing.T) {
	db := initTestDB(t)

	gs := model.GlobalSetting{
		EndpointAddress:     "vpn.example.com",
		DNSServers:          []string{"8.8.8.8", "8.8.4.4"},
		MTU:                 1400,
		PersistentKeepalive: 25,
		FirewallMark:        "0x1234",
		Table:               "auto",
		ConfigFilePath:      "/etc/wireguard/wg0.conf",
		UpdatedAt:           time.Now().UTC(),
	}

	err := db.SaveGlobalSettings(gs)
	require.NoError(t, err)

	got, err := db.GetGlobalSettings()
	require.NoError(t, err)
	assert.Equal(t, "vpn.example.com", got.EndpointAddress)
	assert.Equal(t, []string{"8.8.8.8", "8.8.4.4"}, got.DNSServers)
	assert.Equal(t, 1400, got.MTU)
	assert.Equal(t, 25, got.PersistentKeepalive)
}

// --- Hashes Tests ---

func TestSaveAndGetHashes(t *testing.T) {
	db := initTestDB(t)

	hashes := model.ClientServerHashes{
		Client: "abc123",
		Server: "def456",
	}

	err := db.SaveHashes(hashes)
	require.NoError(t, err)

	got, err := db.GetHashes()
	require.NoError(t, err)
	assert.Equal(t, "abc123", got.Client)
	assert.Equal(t, "def456", got.Server)
}

// --- AllocatedIPs Tests ---

func TestGetAllocatedIPs(t *testing.T) {
	db := initTestDB(t)
	now := time.Now().UTC()

	// save a client with allocated IPs
	db.SaveClient(model.Client{
		ID:              "c1",
		Name:            "Client1",
		AllocatedIPs:    []string{"10.252.1.2/32"},
		AllowedIPs:      []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{},
		SubnetRanges:    []string{},
		CreatedAt:       now,
		UpdatedAt:       now,
	})

	ips, err := db.GetAllocatedIPs("")
	require.NoError(t, err)
	// should include server address + client address
	assert.Contains(t, ips, "10.252.1.2")
}

func TestGetAllocatedIPs_ExcludeClient(t *testing.T) {
	db := initTestDB(t)
	now := time.Now().UTC()

	db.SaveClient(model.Client{
		ID: "c1", AllocatedIPs: []string{"10.252.1.2/32"},
		AllowedIPs: []string{}, ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		CreatedAt: now, UpdatedAt: now,
	})
	db.SaveClient(model.Client{
		ID: "c2", AllocatedIPs: []string{"10.252.1.3/32"},
		AllowedIPs: []string{}, ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		CreatedAt: now, UpdatedAt: now,
	})

	ips, err := db.GetAllocatedIPs("c1")
	require.NoError(t, err)
	assert.NotContains(t, ips, "10.252.1.2")
	assert.Contains(t, ips, "10.252.1.3")
}

// --- GetPath ---

func TestGetPath(t *testing.T) {
	db := initTestDB(t)
	path := db.GetPath()
	assert.NotEmpty(t, path)
}

// --- Init Tests ---

func TestInit_CreatesDefaults(t *testing.T) {
	os.Setenv("WGUI_ENDPOINT_ADDRESS", "10.0.0.1")
	defer os.Unsetenv("WGUI_ENDPOINT_ADDRESS")

	db := newTestDB(t)
	err := db.Init()
	require.NoError(t, err)

	// no default user — first OIDC login creates admin
	users, err := db.GetUsers()
	require.NoError(t, err)
	assert.Len(t, users, 0)

	// should have created server config
	server, err := db.GetServer()
	require.NoError(t, err)
	assert.NotEmpty(t, server.KeyPair.PublicKey)

	// should have created global settings
	gs, err := db.GetGlobalSettings()
	require.NoError(t, err)
	assert.NotEmpty(t, gs.EndpointAddress)

	// should have created hashes
	h, err := db.GetHashes()
	require.NoError(t, err)
	assert.Equal(t, "none", h.Client)
}

func TestInit_Idempotent(t *testing.T) {
	os.Setenv("WGUI_ENDPOINT_ADDRESS", "10.0.0.1")
	defer os.Unsetenv("WGUI_ENDPOINT_ADDRESS")

	db := newTestDB(t)
	require.NoError(t, db.Init())
	require.NoError(t, db.Init()) // second call should not error
}

// --- QR Code Tests ---

func TestGetClients_WithQRCode(t *testing.T) {
	db := initTestDB(t)
	now := time.Now().UTC()

	db.SaveClient(model.Client{
		ID:              "qr1",
		Name:            "QR Client",
		PrivateKey:      "privkey123",
		PublicKey:       "pubkey123",
		AllocatedIPs:    []string{"10.252.1.10/32"},
		AllowedIPs:      []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{},
		SubnetRanges:    []string{},
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	})

	clients, err := db.GetClients(true)
	require.NoError(t, err)
	require.Len(t, clients, 1)
	// Client with a private key should get a QR code when hasQRCode=true
	assert.NotEmpty(t, clients[0].QRCode)
	assert.Contains(t, clients[0].QRCode, "data:image/png;base64,")
}

func TestGetClients_WithQRCode_NoPrivateKey(t *testing.T) {
	db := initTestDB(t)
	now := time.Now().UTC()

	db.SaveClient(model.Client{
		ID:              "noqr1",
		Name:            "No QR Client",
		PrivateKey:      "", // empty private key
		PublicKey:       "pubkey123",
		AllocatedIPs:    []string{"10.252.1.10/32"},
		AllowedIPs:      []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{},
		SubnetRanges:    []string{},
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	})

	clients, err := db.GetClients(true)
	require.NoError(t, err)
	require.Len(t, clients, 1)
	// Client without private key should have no QR code
	assert.Empty(t, clients[0].QRCode)
}

func TestGetClientByID_WithQRCode(t *testing.T) {
	db := initTestDB(t)
	now := time.Now().UTC()

	db.SaveClient(model.Client{
		ID:              "qr2",
		Name:            "QR Client 2",
		PrivateKey:      "privkey456",
		PublicKey:       "pubkey456",
		AllocatedIPs:    []string{"10.252.1.11/32"},
		AllowedIPs:      []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{},
		SubnetRanges:    []string{},
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	})

	// With QR enabled
	clientData, err := db.GetClientByID("qr2", model.QRCodeSettings{
		Enabled:    true,
		IncludeDNS: true,
		IncludeMTU: true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, clientData.QRCode)

	// With QR disabled
	clientData2, err := db.GetClientByID("qr2", model.QRCodeSettings{
		Enabled: false,
	})
	require.NoError(t, err)
	assert.Empty(t, clientData2.QRCode)
}

func TestGetClientByID_QRCode_NoDNS_NoMTU(t *testing.T) {
	db := initTestDB(t)
	now := time.Now().UTC()

	db.SaveClient(model.Client{
		ID:              "qr3",
		Name:            "QR Client 3",
		PrivateKey:      "privkey789",
		PublicKey:       "pubkey789",
		AllocatedIPs:    []string{"10.252.1.12/32"},
		AllowedIPs:      []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{},
		SubnetRanges:    []string{},
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	})

	// With QR enabled but DNS and MTU excluded
	clientData, err := db.GetClientByID("qr3", model.QRCodeSettings{
		Enabled:    true,
		IncludeDNS: false,
		IncludeMTU: false,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, clientData.QRCode)
}

func TestGetClientByID_NotFound(t *testing.T) {
	db := newTestDB(t)

	_, err := db.GetClientByID("nonexistent", model.QRCodeSettings{Enabled: false})
	assert.Error(t, err)
}

// --- DB method ---

func TestDB(t *testing.T) {
	db := newTestDB(t)
	sqlDB := db.DB()
	assert.NotNil(t, sqlDB)

	// verify it can execute a query
	var n int
	err := sqlDB.QueryRow("SELECT 1").Scan(&n)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
}

// --- Init with custom env vars ---

func TestInit_CustomEnvVars(t *testing.T) {
	os.Setenv("WGUI_ENDPOINT_ADDRESS", "vpn.test.com")
	os.Setenv("WGUI_DNS", "8.8.8.8,8.8.4.4")
	os.Setenv("WGUI_MTU", "1400")
	os.Setenv("WGUI_PERSISTENT_KEEPALIVE", "30")
	os.Setenv("WGUI_SERVER_INTERFACE_ADDRESSES", "10.10.0.0/24")
	os.Setenv("WGUI_SERVER_LISTEN_PORT", "12345")
	defer func() {
		os.Unsetenv("WGUI_ENDPOINT_ADDRESS")
		os.Unsetenv("WGUI_DNS")
		os.Unsetenv("WGUI_MTU")
		os.Unsetenv("WGUI_PERSISTENT_KEEPALIVE")
		os.Unsetenv("WGUI_SERVER_INTERFACE_ADDRESSES")
		os.Unsetenv("WGUI_SERVER_LISTEN_PORT")
	}()

	db := newTestDB(t)
	err := db.Init()
	require.NoError(t, err)

	gs, err := db.GetGlobalSettings()
	require.NoError(t, err)
	assert.Equal(t, "vpn.test.com", gs.EndpointAddress)
	assert.Equal(t, []string{"8.8.8.8", "8.8.4.4"}, gs.DNSServers)
	assert.Equal(t, 1400, gs.MTU)
	assert.Equal(t, 30, gs.PersistentKeepalive)

	server, err := db.GetServer()
	require.NoError(t, err)
	assert.Equal(t, []string{"10.10.0.0/24"}, server.Interface.Addresses)
	assert.Equal(t, 12345, server.Interface.ListenPort)
}

// --- Hash operations via store ---

func TestGetCurrentHash_ViaStore(t *testing.T) {
	db := initTestDB(t)

	clientHash, serverHash := util.GetCurrentHash(db)
	assert.NotEmpty(t, clientHash)
	assert.NotEmpty(t, serverHash)
	assert.NotEqual(t, "error", clientHash)
	assert.NotEqual(t, "error", serverHash)
}

func TestHashesChanged_ViaStore(t *testing.T) {
	db := initTestDB(t)

	// Initially hashes should differ (db has 'none', computed is real)
	changed := util.HashesChanged(db)
	assert.True(t, changed)

	// After updating, they should match
	err := util.UpdateHashes(db)
	require.NoError(t, err)

	changed = util.HashesChanged(db)
	assert.False(t, changed)
}

func TestUpdateHashes_ThenChange(t *testing.T) {
	db := initTestDB(t)
	now := time.Now().UTC()

	// Set hashes
	err := util.UpdateHashes(db)
	require.NoError(t, err)
	assert.False(t, util.HashesChanged(db))

	// Add a client to change the hash
	db.SaveClient(model.Client{
		ID: "hashclient", Name: "Hash Client",
		AllocatedIPs:    []string{"10.252.1.50/32"},
		AllowedIPs:      []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		CreatedAt: now, UpdatedAt: now,
	})

	// Now hashes should differ
	assert.True(t, util.HashesChanged(db))
}

// --- ValidateAndFixSubnetRanges ---

func TestValidateAndFixSubnetRanges(t *testing.T) {
	db := initTestDB(t)

	// Set up subnet ranges - the server has 10.252.1.0/24
	util.SubnetRanges = map[string][]*net.IPNet{}
	util.SubnetRangesOrder = nil
	util.IPToSubnetRange = map[string]uint16{}

	util.SubnetRanges = util.ParseSubnetRanges("valid:10.252.1.0/26;invalid:192.168.99.0/24")

	err := util.ValidateAndFixSubnetRanges(db)
	require.NoError(t, err)

	// valid range should remain
	assert.NotNil(t, util.SubnetRanges["valid"])
	// invalid range should be removed (192.168.99.0/24 is outside 10.252.1.0/24)
	_, hasInvalid := util.SubnetRanges["invalid"]
	assert.False(t, hasInvalid)
}

func TestValidateAndFixSubnetRanges_Empty(t *testing.T) {
	db := initTestDB(t)

	util.SubnetRangesOrder = nil
	util.SubnetRanges = map[string][]*net.IPNet{}

	err := util.ValidateAndFixSubnetRanges(db)
	require.NoError(t, err)
}

// --- Migration from JSON ---

func TestMigrateFromJSON_NoOldDB(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()

	// No old JSON DB directory - should return nil
	err := MigrateFromJSON(db, dir)
	assert.NoError(t, err)
}

func TestMigrateFromJSON_FullMigration(t *testing.T) {
	os.Setenv("WGUI_ENDPOINT_ADDRESS", "10.0.0.1")
	defer os.Unsetenv("WGUI_ENDPOINT_ADDRESS")

	db := newTestDB(t)
	require.NoError(t, db.Init())

	// Create a fake JSON database structure
	jsonDBPath := t.TempDir()

	// Create server directory
	serverDir := filepath.Join(jsonDBPath, "server")
	require.NoError(t, os.MkdirAll(serverDir, 0755))

	// Create interfaces.json (ListenPort uses json:",string" tag)
	ifaceJSON := `{"addresses":["10.0.0.1/24"],"listen_port":"51820","post_up":"iptables up","post_down":"iptables down"}`
	require.NoError(t, os.WriteFile(filepath.Join(serverDir, "interfaces.json"), []byte(ifaceJSON), 0644))

	// Create keypair.json
	keypairJSON := `{"private_key":"migprivkey","public_key":"migpubkey"}`
	require.NoError(t, os.WriteFile(filepath.Join(serverDir, "keypair.json"), []byte(keypairJSON), 0644))

	// Create global_settings.json (MTU and PersistentKeepalive use json:",string" tags)
	gsJSON := `{"endpoint_address":"migvpn.test.com","dns_servers":["8.8.8.8"],"mtu":"1400","persistent_keepalive":"25","firewall_mark":"0xca6c","table":"auto","config_file_path":"/etc/wireguard/wg0.conf"}`
	require.NoError(t, os.WriteFile(filepath.Join(serverDir, "global_settings.json"), []byte(gsJSON), 0644))

	// Create hashes.json
	hashesJSON := `{"client":"abc","server":"def"}`
	require.NoError(t, os.WriteFile(filepath.Join(serverDir, "hashes.json"), []byte(hashesJSON), 0644))

	// Create users directory
	usersDir := filepath.Join(jsonDBPath, "users")
	require.NoError(t, os.MkdirAll(usersDir, 0755))
	userJSON := `{"username":"miguser","email":"mig@test.com","admin":true}`
	require.NoError(t, os.WriteFile(filepath.Join(usersDir, "miguser.json"), []byte(userJSON), 0644))

	// Create clients directory
	clientsDir := filepath.Join(jsonDBPath, "clients")
	require.NoError(t, os.MkdirAll(clientsDir, 0755))
	clientJSON := `{"id":"migclient","name":"Mig Client","public_key":"migclientpub","allocated_ips":["10.0.0.2/32"],"allowed_ips":["0.0.0.0/0"],"extra_allowed_ips":[],"subnet_ranges":[],"enabled":true}`
	require.NoError(t, os.WriteFile(filepath.Join(clientsDir, "migclient.json"), []byte(clientJSON), 0644))

	// Create wake_on_lan_hosts directory
	wolDir := filepath.Join(jsonDBPath, "wake_on_lan_hosts")
	require.NoError(t, os.MkdirAll(wolDir, 0755))
	wolJSON := `{"MacAddress":"AA:BB:CC:DD:EE:FF","Name":"Test WOL"}`
	require.NoError(t, os.WriteFile(filepath.Join(wolDir, "host1.json"), []byte(wolJSON), 0644))

	// Run migration
	err := MigrateFromJSON(db, jsonDBPath)
	require.NoError(t, err)

	// Verify data was migrated
	server, err := db.GetServer()
	require.NoError(t, err)
	assert.Equal(t, "migpubkey", server.KeyPair.PublicKey)
	assert.Equal(t, "migprivkey", server.KeyPair.PrivateKey)

	gs, err := db.GetGlobalSettings()
	require.NoError(t, err)
	assert.Equal(t, "migvpn.test.com", gs.EndpointAddress)

	hashes, err := db.GetHashes()
	require.NoError(t, err)
	assert.Equal(t, "abc", hashes.Client)
	assert.Equal(t, "def", hashes.Server)

	user, err := db.GetUserByName("miguser")
	require.NoError(t, err)
	assert.Equal(t, "mig@test.com", user.Email)

	clientData, err := db.GetClientByID("migclient", model.QRCodeSettings{Enabled: false})
	require.NoError(t, err)
	assert.Equal(t, "Mig Client", clientData.Client.Name)

	// Verify old directory was renamed
	_, err = os.Stat(jsonDBPath + ".json.bak")
	assert.NoError(t, err)
}

func TestMigrateFromJSON_MissingOptionalDirs(t *testing.T) {
	os.Setenv("WGUI_ENDPOINT_ADDRESS", "10.0.0.1")
	defer os.Unsetenv("WGUI_ENDPOINT_ADDRESS")

	db := newTestDB(t)
	require.NoError(t, db.Init())

	// Create only the server directory (no users, clients, or wol)
	jsonDBPath := t.TempDir()
	serverDir := filepath.Join(jsonDBPath, "server")
	require.NoError(t, os.MkdirAll(serverDir, 0755))

	// Run migration - should succeed even without optional directories
	err := MigrateFromJSON(db, jsonDBPath)
	require.NoError(t, err)
}

func TestMigrateFromJSON_InvalidJSON(t *testing.T) {
	os.Setenv("WGUI_ENDPOINT_ADDRESS", "10.0.0.1")
	defer os.Unsetenv("WGUI_ENDPOINT_ADDRESS")

	db := newTestDB(t)
	require.NoError(t, db.Init())

	jsonDBPath := t.TempDir()
	serverDir := filepath.Join(jsonDBPath, "server")
	require.NoError(t, os.MkdirAll(serverDir, 0755))

	// Write invalid JSON for interfaces
	require.NoError(t, os.WriteFile(filepath.Join(serverDir, "interfaces.json"), []byte("{invalid"), 0644))

	err := MigrateFromJSON(db, jsonDBPath)
	assert.Error(t, err)
}

// --- Client with AdditionalNotes ---

func TestSaveClient_WithAdditionalNotes(t *testing.T) {
	db := newTestDB(t)
	now := time.Now().UTC()

	client := model.Client{
		ID: "notes-c1", Name: "Notes Client",
		AdditionalNotes: "This client has notes\nMultiple lines",
		AllocatedIPs:    []string{"10.0.0.5/32"},
		AllowedIPs:      []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	}
	err := db.SaveClient(client)
	require.NoError(t, err)

	got, err := db.GetClientByID("notes-c1", model.QRCodeSettings{Enabled: false})
	require.NoError(t, err)
	assert.Contains(t, got.Client.AdditionalNotes, "Multiple lines")
}

func TestSaveClient_WithEndpoint(t *testing.T) {
	db := newTestDB(t)
	now := time.Now().UTC()

	client := model.Client{
		ID: "ep-c1", Name: "Endpoint Client",
		Endpoint:        "vpn.example.com:51820",
		AllocatedIPs:    []string{"10.0.0.6/32"},
		AllowedIPs:      []string{"0.0.0.0/0"},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	}
	err := db.SaveClient(client)
	require.NoError(t, err)

	got, err := db.GetClientByID("ep-c1", model.QRCodeSettings{Enabled: false})
	require.NoError(t, err)
	assert.Equal(t, "vpn.example.com:51820", got.Client.Endpoint)
}

// --- GetUserByOIDCSub Tests ---

func TestGetUserByOIDCSub_Found(t *testing.T) {
	db := newTestDB(t)
	now := time.Now().UTC()

	db.SaveUser(model.User{
		Username:  "oidcuser",
		Email:     "oidc@test.com",
		OIDCSub:   "sub-12345",
		Admin:     true,
		CreatedAt: now,
		UpdatedAt: now,
	})

	user, err := db.GetUserByOIDCSub("sub-12345")
	require.NoError(t, err)
	assert.Equal(t, "oidcuser", user.Username)
	assert.Equal(t, "oidc@test.com", user.Email)
	assert.Equal(t, "sub-12345", user.OIDCSub)
	assert.True(t, user.Admin)
}

func TestGetUserByOIDCSub_NotFound(t *testing.T) {
	db := newTestDB(t)

	_, err := db.GetUserByOIDCSub("nonexistent-sub")
	assert.Error(t, err)
}

// --- GetClients with search by notes ---

func TestGetClients_SearchByNotes(t *testing.T) {
	db := initTestDB(t)
	now := time.Now().UTC()

	db.SaveClient(model.Client{
		ID: "notes1", Name: "Client With Notes",
		AdditionalNotes: "special deployment note",
		AllocatedIPs:    []string{}, AllowedIPs: []string{},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		CreatedAt: now, UpdatedAt: now,
	})
	db.SaveClient(model.Client{
		ID: "notes2", Name: "Other Client",
		AdditionalNotes: "regular note",
		AllocatedIPs:    []string{}, AllowedIPs: []string{},
		ExtraAllowedIPs: []string{}, SubnetRanges: []string{},
		CreatedAt: now, UpdatedAt: now,
	})

	clients, err := db.GetClients(false)
	require.NoError(t, err)
	assert.Len(t, clients, 2)
}

func TestMigrateFromJSON_SkipsNonJSON(t *testing.T) {
	os.Setenv("WGUI_ENDPOINT_ADDRESS", "10.0.0.1")
	defer os.Unsetenv("WGUI_ENDPOINT_ADDRESS")

	db := newTestDB(t)
	require.NoError(t, db.Init())

	jsonDBPath := t.TempDir()
	serverDir := filepath.Join(jsonDBPath, "server")
	require.NoError(t, os.MkdirAll(serverDir, 0755))

	// Create users dir with non-JSON files
	usersDir := filepath.Join(jsonDBPath, "users")
	require.NoError(t, os.MkdirAll(usersDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(usersDir, "readme.txt"), []byte("not json"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(usersDir, "subdir"), 0755))

	// Create clients dir with invalid JSON (should warn and skip)
	clientsDir := filepath.Join(jsonDBPath, "clients")
	require.NoError(t, os.MkdirAll(clientsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(clientsDir, "bad.json"), []byte("{invalid"), 0644))

	// Create wol dir with invalid JSON (should warn and skip)
	wolDir := filepath.Join(jsonDBPath, "wake_on_lan_hosts")
	require.NoError(t, os.MkdirAll(wolDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(wolDir, "bad.json"), []byte("{invalid"), 0644))

	err := MigrateFromJSON(db, jsonDBPath)
	require.NoError(t, err)
}

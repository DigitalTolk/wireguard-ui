package util

import (
	"net"
	"os"
	"testing"

	"github.com/labstack/gommon/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DigitalTolk/wireguard-ui/model"
)

func TestValidateCIDR(t *testing.T) {
	assert.True(t, ValidateCIDR("10.0.0.0/24"))
	assert.True(t, ValidateCIDR("192.168.1.0/32"))
	assert.True(t, ValidateCIDR("fd00::/64"))
	assert.False(t, ValidateCIDR("10.0.0.0"))
	assert.False(t, ValidateCIDR("invalid"))
	assert.False(t, ValidateCIDR(""))
}

func TestValidateCIDRList(t *testing.T) {
	assert.True(t, ValidateCIDRList([]string{"10.0.0.0/24", "192.168.0.0/16"}, false))
	assert.False(t, ValidateCIDRList([]string{"10.0.0.0/24", "invalid"}, false))
	assert.True(t, ValidateCIDRList([]string{"10.0.0.0/24", ""}, true))
	assert.False(t, ValidateCIDRList([]string{"10.0.0.0/24", ""}, false))
	assert.True(t, ValidateCIDRList([]string{}, false))
}

func TestValidateAllowedIPs(t *testing.T) {
	assert.True(t, ValidateAllowedIPs([]string{"0.0.0.0/0"}))
	assert.True(t, ValidateAllowedIPs([]string{"10.0.0.0/24", "192.168.1.0/24"}))
	assert.False(t, ValidateAllowedIPs([]string{"not-a-cidr"}))
}

func TestValidateExtraAllowedIPs(t *testing.T) {
	assert.True(t, ValidateExtraAllowedIPs([]string{"10.0.0.0/24"}))
	assert.True(t, ValidateExtraAllowedIPs([]string{"10.0.0.0/24", ""}))
	assert.False(t, ValidateExtraAllowedIPs([]string{"invalid"}))
}

func TestValidateServerAddresses(t *testing.T) {
	assert.True(t, ValidateServerAddresses([]string{"10.252.1.0/24"}))
	assert.False(t, ValidateServerAddresses([]string{"10.252.1.0"}))
}

func TestValidateIPAddress(t *testing.T) {
	assert.True(t, ValidateIPAddress("10.0.0.1"))
	assert.True(t, ValidateIPAddress("::1"))
	assert.True(t, ValidateIPAddress("192.168.1.1"))
	assert.False(t, ValidateIPAddress("999.0.0.1"))
	assert.False(t, ValidateIPAddress("invalid"))
	assert.False(t, ValidateIPAddress(""))
}

func TestValidateIPAddressList(t *testing.T) {
	assert.True(t, ValidateIPAddressList([]string{"1.1.1.1", "8.8.8.8"}))
	assert.False(t, ValidateIPAddressList([]string{"1.1.1.1", "bad"}))
	assert.True(t, ValidateIPAddressList([]string{}))
}

func TestGetIPFromCIDR(t *testing.T) {
	ip, err := GetIPFromCIDR("10.252.1.5/24")
	assert.NoError(t, err)
	assert.Equal(t, "10.252.1.5", ip)

	ip, err = GetIPFromCIDR("192.168.1.100/32")
	assert.NoError(t, err)
	assert.Equal(t, "192.168.1.100", ip)

	_, err = GetIPFromCIDR("invalid")
	assert.Error(t, err)
}

func TestContainsCIDR(t *testing.T) {
	_, net1, _ := net.ParseCIDR("10.0.0.0/8")
	_, net2, _ := net.ParseCIDR("10.0.0.0/24")
	_, net3, _ := net.ParseCIDR("192.168.0.0/24")

	assert.True(t, ContainsCIDR(net1, net2))
	assert.False(t, ContainsCIDR(net2, net1))
	assert.False(t, ContainsCIDR(net1, net3))
}

func TestGetBroadcastIP(t *testing.T) {
	_, netAddr, _ := net.ParseCIDR("10.0.0.0/24")
	broadcast := GetBroadcastIP(netAddr)
	assert.Equal(t, "10.0.0.255", broadcast.String())

	_, netAddr2, _ := net.ParseCIDR("192.168.1.0/30")
	broadcast2 := GetBroadcastIP(netAddr2)
	assert.Equal(t, "192.168.1.3", broadcast2.String())
}

func TestGetBroadcastAndNetworkAddrsLookup(t *testing.T) {
	lookup := GetBroadcastAndNetworkAddrsLookup([]string{"10.0.0.0/24"})
	assert.True(t, lookup["10.0.0.0"])
	assert.True(t, lookup["10.0.0.255"])
	assert.False(t, lookup["10.0.0.1"])
}

func TestGetAvailableIP(t *testing.T) {
	ip, err := GetAvailableIP("10.0.0.0/24", []string{}, []string{"10.0.0.0/24"})
	assert.NoError(t, err)
	assert.Equal(t, "10.0.0.1", ip)

	ip, err = GetAvailableIP("10.0.0.0/24", []string{"10.0.0.1"}, []string{"10.0.0.0/24"})
	assert.NoError(t, err)
	assert.Equal(t, "10.0.0.2", ip)

	_, err = GetAvailableIP("invalid", []string{}, []string{})
	assert.Error(t, err)

	// exhaust a /30 (only 2 usable IPs: .1 and .2)
	_, err = GetAvailableIP("10.0.0.0/30", []string{"10.0.0.1", "10.0.0.2"}, []string{"10.0.0.0/30"})
	assert.Error(t, err)
}

func TestValidateIPAllocation(t *testing.T) {
	serverAddrs := []string{"10.252.1.0/24"}

	ok, err := ValidateIPAllocation(serverAddrs, []string{}, []string{"10.252.1.5/32"})
	assert.True(t, ok)
	assert.NoError(t, err)

	// already allocated
	ok, err = ValidateIPAllocation(serverAddrs, []string{"10.252.1.5"}, []string{"10.252.1.5/32"})
	assert.False(t, ok)
	assert.Error(t, err)

	// not in server network
	ok, err = ValidateIPAllocation(serverAddrs, []string{}, []string{"192.168.1.5/32"})
	assert.False(t, ok)
	assert.Error(t, err)

	// invalid CIDR
	ok, err = ValidateIPAllocation(serverAddrs, []string{}, []string{"not-cidr"})
	assert.False(t, ok)
	assert.Error(t, err)
}

func TestBuildClientConfig(t *testing.T) {
	client := model.Client{
		AllocatedIPs: []string{"10.252.1.2/32"},
		PrivateKey:   "clientprivkey",
		AllowedIPs:   []string{"0.0.0.0/0"},
		UseServerDNS: true,
		PresharedKey: "psk123",
	}
	server := model.Server{
		KeyPair: &model.ServerKeypair{
			PublicKey: "serverpubkey",
		},
		Interface: &model.ServerInterface{
			ListenPort: 51820,
		},
	}
	setting := model.GlobalSetting{
		EndpointAddress:     "vpn.example.com",
		DNSServers:          []string{"1.1.1.1"},
		MTU:                 1450,
		PersistentKeepalive: 15,
	}

	config := BuildClientConfig(client, server, setting)
	assert.Contains(t, config, "[Interface]")
	assert.Contains(t, config, "Address = 10.252.1.2/32")
	assert.Contains(t, config, "PrivateKey = clientprivkey")
	assert.Contains(t, config, "DNS = 1.1.1.1")
	assert.Contains(t, config, "MTU = 1450")
	assert.Contains(t, config, "[Peer]")
	assert.Contains(t, config, "PublicKey = serverpubkey")
	assert.Contains(t, config, "PresharedKey = psk123")
	assert.Contains(t, config, "AllowedIPs = 0.0.0.0/0")
	assert.Contains(t, config, "Endpoint = vpn.example.com:51820")
	assert.Contains(t, config, "PersistentKeepalive = 15")
}

func TestBuildClientConfig_NoDNS(t *testing.T) {
	client := model.Client{
		AllocatedIPs: []string{"10.0.0.2/32"},
		PrivateKey:   "key",
		AllowedIPs:   []string{"0.0.0.0/0"},
		UseServerDNS: false,
	}
	server := model.Server{
		KeyPair:   &model.ServerKeypair{PublicKey: "pub"},
		Interface: &model.ServerInterface{ListenPort: 51820},
	}
	setting := model.GlobalSetting{
		EndpointAddress: "1.2.3.4",
		DNSServers:      []string{"1.1.1.1"},
	}

	config := BuildClientConfig(client, server, setting)
	assert.NotContains(t, config, "DNS =")
	assert.NotContains(t, config, "MTU =")
	assert.NotContains(t, config, "PresharedKey =")
	assert.NotContains(t, config, "PersistentKeepalive =")
}

func TestBuildClientConfig_EndpointWithPort(t *testing.T) {
	client := model.Client{
		AllocatedIPs: []string{"10.0.0.2/32"},
		PrivateKey:   "key",
		AllowedIPs:   []string{"0.0.0.0/0"},
	}
	server := model.Server{
		KeyPair:   &model.ServerKeypair{PublicKey: "pub"},
		Interface: &model.ServerInterface{ListenPort: 51820},
	}
	setting := model.GlobalSetting{
		EndpointAddress: "vpn.example.com:9999",
	}

	config := BuildClientConfig(client, server, setting)
	assert.Contains(t, config, "Endpoint = vpn.example.com:9999")
}

func TestLookupEnvOrString(t *testing.T) {
	assert.Equal(t, "default", LookupEnvOrString("NONEXISTENT_ENV_VAR_12345", "default"))

	os.Setenv("TEST_LOOKUP_STR", "custom")
	defer os.Unsetenv("TEST_LOOKUP_STR")
	assert.Equal(t, "custom", LookupEnvOrString("TEST_LOOKUP_STR", "default"))
}

func TestLookupEnvOrBool(t *testing.T) {
	assert.True(t, LookupEnvOrBool("NONEXISTENT_ENV_VAR_12345", true))
	assert.False(t, LookupEnvOrBool("NONEXISTENT_ENV_VAR_12345", false))

	os.Setenv("TEST_LOOKUP_BOOL", "true")
	defer os.Unsetenv("TEST_LOOKUP_BOOL")
	assert.True(t, LookupEnvOrBool("TEST_LOOKUP_BOOL", false))
}

func TestLookupEnvOrInt(t *testing.T) {
	assert.Equal(t, 42, LookupEnvOrInt("NONEXISTENT_ENV_VAR_12345", 42))

	os.Setenv("TEST_LOOKUP_INT", "99")
	defer os.Unsetenv("TEST_LOOKUP_INT")
	assert.Equal(t, 99, LookupEnvOrInt("TEST_LOOKUP_INT", 42))
}

func TestLookupEnvOrStrings(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, LookupEnvOrStrings("NONEXISTENT_ENV_VAR_12345", []string{"a", "b"}))

	os.Setenv("TEST_LOOKUP_STRS", "x,y,z")
	defer os.Unsetenv("TEST_LOOKUP_STRS")
	assert.Equal(t, []string{"x", "y", "z"}, LookupEnvOrStrings("TEST_LOOKUP_STRS", []string{}))
}

func TestLookupEnvOrFile(t *testing.T) {
	assert.Equal(t, "default", LookupEnvOrFile("NONEXISTENT_ENV_VAR_12345", "default"))

	tmpFile, err := os.CreateTemp("", "test_lookup")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("secret_from_file")
	tmpFile.Close()

	os.Setenv("TEST_LOOKUP_FILE", tmpFile.Name())
	defer os.Unsetenv("TEST_LOOKUP_FILE")
	assert.Equal(t, "secret_from_file", LookupEnvOrFile("TEST_LOOKUP_FILE", "default"))
}

func TestParseLogLevel(t *testing.T) {
	lvl, err := ParseLogLevel("debug")
	assert.NoError(t, err)
	assert.Equal(t, log.DEBUG, lvl)

	lvl, err = ParseLogLevel("INFO")
	assert.NoError(t, err)
	assert.Equal(t, log.INFO, lvl)

	lvl, err = ParseLogLevel("warn")
	assert.NoError(t, err)
	assert.Equal(t, log.WARN, lvl)

	lvl, err = ParseLogLevel("error")
	assert.NoError(t, err)
	assert.Equal(t, log.ERROR, lvl)

	lvl, err = ParseLogLevel("off")
	assert.NoError(t, err)
	assert.Equal(t, log.OFF, lvl)

	_, err = ParseLogLevel("invalid")
	assert.Error(t, err)
}

func TestRandomString(t *testing.T) {
	s := RandomString(32)
	assert.Len(t, s, 32)

	s2 := RandomString(32)
	assert.Len(t, s2, 32)
	// extremely unlikely to be equal
	assert.NotEqual(t, s, s2)

	assert.Len(t, RandomString(0), 0)
}

func TestGetCookiePath(t *testing.T) {
	original := BasePath
	defer func() { BasePath = original }()

	BasePath = ""
	assert.Equal(t, "/", GetCookiePath())

	BasePath = "/app"
	assert.Equal(t, "/app", GetCookiePath())
}

func TestGetDBUserCRC32(t *testing.T) {
	user1 := model.User{Username: "admin", Admin: true}
	user2 := model.User{Username: "admin", Admin: true}
	user3 := model.User{Username: "other", Admin: false}

	hash1 := GetDBUserCRC32(user1)
	hash2 := GetDBUserCRC32(user2)
	hash3 := GetDBUserCRC32(user3)

	assert.Equal(t, hash1, hash2)
	assert.NotEqual(t, hash1, hash3)
}

func TestConcatMultipleSlices(t *testing.T) {
	result := ConcatMultipleSlices([]byte{1, 2}, []byte{3, 4}, []byte{5})
	assert.Equal(t, []byte{1, 2, 3, 4, 5}, result)

	result = ConcatMultipleSlices()
	assert.Equal(t, []byte{}, result)

	result = ConcatMultipleSlices([]byte{1})
	assert.Equal(t, []byte{1}, result)
}

func TestTgUseridToClientID(t *testing.T) {
	// reset
	TgUseridToClientID = map[int64][]string{}

	AddTgToClientID(123, "client1")
	assert.Equal(t, []string{"client1"}, TgUseridToClientID[123])

	AddTgToClientID(123, "client2")
	assert.Equal(t, []string{"client1", "client2"}, TgUseridToClientID[123])

	UpdateTgToClientID(456, "client1")
	assert.Equal(t, []string{"client2"}, TgUseridToClientID[123])
	assert.Equal(t, []string{"client1"}, TgUseridToClientID[456])

	RemoveTgToClientID("client1")
	_, has456 := TgUseridToClientID[456]
	assert.False(t, has456)

	RemoveTgToClientID("client2")
	_, has123 := TgUseridToClientID[123]
	assert.False(t, has123)
}

func TestFillClientSubnetRange(t *testing.T) {
	// reset global state
	SubnetRanges = map[string][]*net.IPNet{}
	SubnetRangesOrder = nil
	IPToSubnetRange = map[string]uint16{}

	SubnetRanges = ParseSubnetRanges("LAN:10.0.0.0/8")

	client := model.ClientData{
		Client: &model.Client{
			AllocatedIPs: []string{"10.0.0.5/32"},
		},
	}

	result := FillClientSubnetRange(client)
	assert.Contains(t, result.Client.SubnetRanges, "LAN")
}

func TestGetSubnetRangesString(t *testing.T) {
	SubnetRanges = map[string][]*net.IPNet{}
	SubnetRangesOrder = nil
	assert.Equal(t, "", GetSubnetRangesString())

	SubnetRanges = ParseSubnetRanges("LAN:10.0.0.0/24")
	result := GetSubnetRangesString()
	assert.Contains(t, result, "LAN:")
	assert.Contains(t, result, "10.0.0.0/24")
}

func TestGetInterfaceIPs(t *testing.T) {
	ips, err := GetInterfaceIPs()
	assert.NoError(t, err)
	// should return at least an empty list, not error
	assert.NotNil(t, ips)
}

func TestManagePerms(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_perms")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	err = ManagePerms(tmpFile.Name())
	assert.NoError(t, err)

	err = ManagePerms("/nonexistent/path")
	assert.Error(t, err)
}

func TestWriteWireGuardServerConfig(t *testing.T) {
	tmpDir := t.TempDir()
	confPath := tmpDir + "/wg0.conf"

	serverConfig := model.Server{
		Interface: &model.ServerInterface{
			Addresses:  []string{"10.0.0.1/24"},
			ListenPort: 51820,
			PostUp:     "iptables -A",
			PostDown:   "iptables -D",
		},
		KeyPair: &model.ServerKeypair{
			PrivateKey: "serverprivkey",
			PublicKey:  "serverpubkey",
		},
	}
	clientDataList := []model.ClientData{
		{
			Client: &model.Client{
				ID:              "client1",
				Name:            "Test Client",
				Email:           "test@example.com",
				PublicKey:       "clientpubkey",
				PresharedKey:    "clientpsk",
				AllocatedIPs:    []string{"10.0.0.2/32"},
				ExtraAllowedIPs: []string{},
				Enabled:         true,
				AdditionalNotes: "line1\nline2",
			},
		},
		{
			Client: &model.Client{
				ID:           "client2",
				Name:         "Disabled",
				PublicKey:    "pub2",
				AllocatedIPs: []string{"10.0.0.3/32"},
				Enabled:      false,
			},
		},
	}
	globalSettings := model.GlobalSetting{
		MTU:                 1420,
		PersistentKeepalive: 25,
		Table:               "auto",
		ConfigFilePath:      confPath,
	}

	// Use an in-memory FS with the wg.conf template
	tmplFS := os.DirFS("../templates")

	err := WriteWireGuardServerConfig(tmplFS, serverConfig, clientDataList, nil, globalSettings)
	require.NoError(t, err)

	content, err := os.ReadFile(confPath)
	require.NoError(t, err)
	s := string(content)
	assert.Contains(t, s, "[Interface]")
	assert.Contains(t, s, "PrivateKey = serverprivkey")
	assert.Contains(t, s, "ListenPort = 51820")
	assert.Contains(t, s, "PublicKey = clientpubkey")
	// multiline notes should be escaped
	assert.Contains(t, s, "# line2")
	// disabled client should NOT appear as a [Peer]
	assert.NotContains(t, s, "pub2")
}

func TestWriteWireGuardServerConfig_CustomTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	confPath := tmpDir + "/wg0.conf"
	tmplPath := tmpDir + "/custom.conf"

	// write a custom template
	customTmpl := `[Interface]
PrivateKey = {{ .serverConfig.KeyPair.PrivateKey }}
`
	err := os.WriteFile(tmplPath, []byte(customTmpl), 0644)
	require.NoError(t, err)

	// set WgConfTemplate
	original := WgConfTemplate
	WgConfTemplate = tmplPath
	defer func() { WgConfTemplate = original }()

	serverConfig := model.Server{
		Interface: &model.ServerInterface{ListenPort: 51820},
		KeyPair:   &model.ServerKeypair{PrivateKey: "customprivkey", PublicKey: "pub"},
	}
	globalSettings := model.GlobalSetting{ConfigFilePath: confPath}

	err = WriteWireGuardServerConfig(nil, serverConfig, nil, nil, globalSettings)
	require.NoError(t, err)

	content, err := os.ReadFile(confPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "customprivkey")
}

func TestWriteWireGuardServerConfig_InvalidPath(t *testing.T) {
	serverConfig := model.Server{
		Interface: &model.ServerInterface{ListenPort: 51820},
		KeyPair:   &model.ServerKeypair{PrivateKey: "priv", PublicKey: "pub"},
	}
	globalSettings := model.GlobalSetting{ConfigFilePath: "/nonexistent/dir/wg0.conf"}
	tmplFS := os.DirFS("../templates")

	err := WriteWireGuardServerConfig(tmplFS, serverConfig, nil, nil, globalSettings)
	assert.Error(t, err)
}

func TestStringFromEmbedFile(t *testing.T) {
	tmplFS := os.DirFS("../templates")
	content, err := StringFromEmbedFile(tmplFS, "wg.conf")
	require.NoError(t, err)
	assert.Contains(t, content, "[Interface]")

	_, err = StringFromEmbedFile(tmplFS, "nonexistent.conf")
	assert.Error(t, err)
}

func TestClientDefaultsFromEnv(t *testing.T) {
	// test defaults
	os.Unsetenv(DefaultClientAllowedIpsEnvVar)
	os.Unsetenv(DefaultClientExtraAllowedIpsEnvVar)
	os.Unsetenv(DefaultClientUseServerDNSEnvVar)
	os.Unsetenv(DefaultClientEnableAfterCreationEnvVar)

	defaults := ClientDefaultsFromEnv()
	assert.Equal(t, []string{"0.0.0.0/0"}, defaults.AllowedIps)
	assert.Equal(t, []string{}, defaults.ExtraAllowedIps)
	assert.True(t, defaults.UseServerDNS)
	assert.True(t, defaults.EnableAfterCreation)

	// test with env overrides
	os.Setenv(DefaultClientAllowedIpsEnvVar, "10.0.0.0/8,192.168.0.0/16")
	os.Setenv(DefaultClientUseServerDNSEnvVar, "false")
	os.Setenv(DefaultClientEnableAfterCreationEnvVar, "false")
	defer os.Unsetenv(DefaultClientAllowedIpsEnvVar)
	defer os.Unsetenv(DefaultClientUseServerDNSEnvVar)
	defer os.Unsetenv(DefaultClientEnableAfterCreationEnvVar)

	defaults = ClientDefaultsFromEnv()
	assert.Equal(t, []string{"10.0.0.0/8", "192.168.0.0/16"}, defaults.AllowedIps)
	assert.False(t, defaults.UseServerDNS)
	assert.False(t, defaults.EnableAfterCreation)
}

func TestGetCurrentHash_WithMockStore(t *testing.T) {
	store := newMockStore()

	clientHash, serverHash := GetCurrentHash(store)
	assert.NotEmpty(t, clientHash)
	assert.NotEmpty(t, serverHash)
	assert.NotEqual(t, "error", clientHash)
	assert.NotEqual(t, "error", serverHash)
}

func TestHashesChanged_WithMockStore(t *testing.T) {
	store := newMockStore()

	// Initially hashes are "none" which differs from computed
	changed := HashesChanged(store)
	assert.True(t, changed)

	// After updating, they should match
	err := UpdateHashes(store)
	require.NoError(t, err)
	changed = HashesChanged(store)
	assert.False(t, changed)
}

func TestUpdateHashes_WithMockStore(t *testing.T) {
	store := newMockStore()

	err := UpdateHashes(store)
	require.NoError(t, err)

	hashes, err := store.GetHashes()
	require.NoError(t, err)
	assert.NotEqual(t, "none", hashes.Client)
	assert.NotEqual(t, "none", hashes.Server)
}

func TestValidateAndFixSubnetRanges_WithMockStore(t *testing.T) {
	store := newMockStore()
	// Server has 10.0.0.0/24

	SubnetRanges = map[string][]*net.IPNet{}
	SubnetRangesOrder = nil
	IPToSubnetRange = map[string]uint16{}

	SubnetRanges = ParseSubnetRanges("valid:10.0.0.0/26;invalid:192.168.99.0/24")

	err := ValidateAndFixSubnetRanges(store)
	require.NoError(t, err)

	// valid range should remain
	assert.NotNil(t, SubnetRanges["valid"])
	// invalid range should be removed
	_, hasInvalid := SubnetRanges["invalid"]
	assert.False(t, hasInvalid)
}

func TestValidateAndFixSubnetRanges_AllInvalid(t *testing.T) {
	store := newMockStore()
	// Server has 10.0.0.0/24

	SubnetRanges = map[string][]*net.IPNet{}
	SubnetRangesOrder = nil
	IPToSubnetRange = map[string]uint16{}

	SubnetRanges = ParseSubnetRanges("outside:192.168.0.0/16")

	err := ValidateAndFixSubnetRanges(store)
	require.NoError(t, err)

	_, hasOutside := SubnetRanges["outside"]
	assert.False(t, hasOutside)
}

func TestValidateAndFixSubnetRanges_EmptyRanges(t *testing.T) {
	store := newMockStore()
	SubnetRangesOrder = nil
	SubnetRanges = map[string][]*net.IPNet{}

	err := ValidateAndFixSubnetRanges(store)
	require.NoError(t, err)
}

func TestGetBroadcastIP_IPv6(t *testing.T) {
	_, netAddr, _ := net.ParseCIDR("fd00::/120")
	broadcast := GetBroadcastIP(netAddr)
	assert.Equal(t, "fd00::ff", broadcast.String())
}

func TestGetBroadcastAndNetworkAddrsLookup_Invalid(t *testing.T) {
	lookup := GetBroadcastAndNetworkAddrsLookup([]string{"invalid"})
	assert.Empty(t, lookup)
}

func TestGetBroadcastAndNetworkAddrsLookup_Multiple(t *testing.T) {
	lookup := GetBroadcastAndNetworkAddrsLookup([]string{"10.0.0.0/24", "192.168.1.0/30"})
	assert.True(t, lookup["10.0.0.0"])
	assert.True(t, lookup["10.0.0.255"])
	assert.True(t, lookup["192.168.1.0"])
	assert.True(t, lookup["192.168.1.3"])
}

func TestBuildClientConfig_EndpointInvalidPort(t *testing.T) {
	client := model.Client{
		AllocatedIPs: []string{"10.0.0.2/32"},
		PrivateKey:   "key",
		AllowedIPs:   []string{"0.0.0.0/0"},
	}
	server := model.Server{
		KeyPair:   &model.ServerKeypair{PublicKey: "pub"},
		Interface: &model.ServerInterface{ListenPort: 51820},
	}
	setting := model.GlobalSetting{
		EndpointAddress: "vpn.example.com:notanumber",
	}

	config := BuildClientConfig(client, server, setting)
	// should fall back to server listen port
	assert.Contains(t, config, "Endpoint = vpn.example.com:51820")
}

func TestLookupEnvOrBool_InvalidValue(t *testing.T) {
	os.Setenv("TEST_BOOL_INVALID", "notabool")
	defer os.Unsetenv("TEST_BOOL_INVALID")
	// should return default-ish false since ParseBool fails
	result := LookupEnvOrBool("TEST_BOOL_INVALID", true)
	assert.False(t, result) // ParseBool returns false on error
}

func TestLookupEnvOrInt_InvalidValue(t *testing.T) {
	os.Setenv("TEST_INT_INVALID", "notanint")
	defer os.Unsetenv("TEST_INT_INVALID")
	result := LookupEnvOrInt("TEST_INT_INVALID", 42)
	assert.Equal(t, 0, result) // Atoi returns 0 on error
}

func TestLookupEnvOrFile_InvalidFilePath(t *testing.T) {
	os.Setenv("TEST_FILE_INVALID", "/nonexistent/file/path")
	defer os.Unsetenv("TEST_FILE_INVALID")
	result := LookupEnvOrFile("TEST_FILE_INVALID", "default")
	assert.Equal(t, "default", result) // file open fails, returns default
}

func TestFindSubnetRangeForIP(t *testing.T) {
	// reset global state
	SubnetRanges = map[string][]*net.IPNet{}
	SubnetRangesOrder = nil
	IPToSubnetRange = map[string]uint16{}

	SubnetRanges = ParseSubnetRanges("LAN:10.0.0.0/24;WAN:192.168.0.0/16")

	// test finding a matching subnet range
	client := model.ClientData{
		Client: &model.Client{
			AllocatedIPs: []string{"10.0.0.5/32"},
		},
	}
	result := FillClientSubnetRange(client)
	assert.Contains(t, result.Client.SubnetRanges, "LAN")

	// test with IP not in any range
	client2 := model.ClientData{
		Client: &model.Client{
			AllocatedIPs: []string{"172.16.0.1/32"},
		},
	}
	result2 := FillClientSubnetRange(client2)
	assert.Empty(t, result2.Client.SubnetRanges)

	// test invalid CIDR
	client3 := model.ClientData{
		Client: &model.Client{
			AllocatedIPs: []string{"not-a-cidr"},
		},
	}
	result3 := FillClientSubnetRange(client3)
	assert.Empty(t, result3.Client.SubnetRanges)

	// test cached lookup (call again with same IP)
	client4 := model.ClientData{
		Client: &model.Client{
			AllocatedIPs: []string{"10.0.0.5/32"},
		},
	}
	result4 := FillClientSubnetRange(client4)
	assert.Contains(t, result4.Client.SubnetRanges, "LAN")
}

func TestUpdateTgToClientID_DetachAndReattach(t *testing.T) {
	// reset
	TgUseridToClientID = map[int64][]string{}

	// Add to user 100
	AddTgToClientID(100, "clientA")
	AddTgToClientID(100, "clientB")
	assert.Equal(t, []string{"clientA", "clientB"}, TgUseridToClientID[100])

	// Update clientA to user 200 (should detach from 100)
	UpdateTgToClientID(200, "clientA")
	assert.Equal(t, []string{"clientB"}, TgUseridToClientID[100])
	assert.Equal(t, []string{"clientA"}, TgUseridToClientID[200])

	// Update clientB to user 200 (should remove 100 entirely since empty)
	UpdateTgToClientID(200, "clientB")
	_, has100 := TgUseridToClientID[100]
	assert.False(t, has100)
	assert.Equal(t, []string{"clientA", "clientB"}, TgUseridToClientID[200])

	// Remove non-existent clientID - should be safe
	RemoveTgToClientID("nonexistent")
	assert.Equal(t, []string{"clientA", "clientB"}, TgUseridToClientID[200])
}

// --- SendRequestedConfigsToTelegram Tests ---

func TestSendRequestedConfigsToTelegram_NoUserid(t *testing.T) {
	store := newMockStore()

	// Empty map - should return empty list immediately
	TgUseridToClientID = map[int64][]string{}
	result := SendRequestedConfigsToTelegram(store, 12345)
	assert.Empty(t, result)
}

func TestSendRequestedConfigsToTelegram_ClientNotFound(t *testing.T) {
	store := newMockStore()

	// Set up a userid mapping to a client that doesn't exist
	TgUseridToClientID = map[int64][]string{
		12345: {"nonexistent-client"},
	}

	result := SendRequestedConfigsToTelegram(store, 12345)
	assert.Contains(t, result, "nonexistent-client")
}

func TestSendRequestedConfigsToTelegram_InvalidTgUserid(t *testing.T) {
	store := newMockStore()

	// Add a client with invalid TgUserid
	store.clients = []model.ClientData{
		{
			Client: &model.Client{
				ID:           "client1",
				Name:         "Bad TG Client",
				TgUserid:     "not-a-number",
				PublicKey:    "pub1",
				PrivateKey:   "priv1",
				AllocatedIPs: []string{"10.0.0.2/32"},
				AllowedIPs:   []string{"0.0.0.0/0"},
			},
		},
	}

	TgUseridToClientID = map[int64][]string{
		12345: {"client1"},
	}

	result := SendRequestedConfigsToTelegram(store, 12345)
	assert.Contains(t, result, "Bad TG Client")
}

func TestSendRequestedConfigsToTelegram_ValidClientFailsTelegram(t *testing.T) {
	store := newMockStore()

	// Add a client with a valid TgUserid and private key (for QR code)
	store.clients = []model.ClientData{
		{
			Client: &model.Client{
				ID:           "client-tg",
				Name:         "TG Valid Client",
				TgUserid:     "99999",
				PublicKey:    "pub1",
				PrivateKey:   "priv1",
				AllocatedIPs: []string{"10.0.0.2/32"},
				AllowedIPs:   []string{"0.0.0.0/0"},
			},
		},
	}

	TgUseridToClientID = map[int64][]string{
		99999: {"client-tg"},
	}

	// This will call telegram.SendConfig which will fail (no token/bot configured)
	// but it exercises the config building, QR code generation, and userid parsing paths
	result := SendRequestedConfigsToTelegram(store, 99999)
	// The telegram send will fail, so the client should be in the failed list
	assert.Contains(t, result, "TG Valid Client")
}

func TestSendRequestedConfigsToTelegram_ClientWithoutPrivateKey(t *testing.T) {
	store := newMockStore()

	// Client without private key - no QR code generated
	store.clients = []model.ClientData{
		{
			Client: &model.Client{
				ID:           "client-nopk",
				Name:         "No PK Client",
				TgUserid:     "88888",
				PublicKey:    "pub1",
				PrivateKey:   "", // no private key
				AllocatedIPs: []string{"10.0.0.3/32"},
				AllowedIPs:   []string{"0.0.0.0/0"},
			},
		},
	}

	TgUseridToClientID = map[int64][]string{
		88888: {"client-nopk"},
	}

	result := SendRequestedConfigsToTelegram(store, 88888)
	assert.Contains(t, result, "No PK Client")
}

// --- WriteWireGuardServerConfig with custom template (additional) ---

func TestWriteWireGuardServerConfig_CustomTemplateFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a custom template file
	tmplPath := tmpDir + "/custom.conf"
	os.WriteFile(tmplPath, []byte("[Interface]\n# Custom template\n"), 0644)

	origWgConfTemplate := WgConfTemplate
	WgConfTemplate = tmplPath
	defer func() { WgConfTemplate = origWgConfTemplate }()

	settings := model.GlobalSetting{
		ConfigFilePath: tmpDir + "/wg0.conf",
	}
	server := model.Server{
		Interface: &model.ServerInterface{
			Addresses:  []string{"10.0.0.0/24"},
			ListenPort: 51820,
		},
		KeyPair: &model.ServerKeypair{
			PrivateKey: "privkey",
			PublicKey:  "pubkey",
		},
	}

	err := WriteWireGuardServerConfig(os.DirFS(tmpDir), server, nil, nil, settings)
	require.NoError(t, err)

	content, _ := os.ReadFile(tmpDir + "/wg0.conf")
	assert.Contains(t, string(content), "Custom template")
}

func TestWriteWireGuardServerConfig_WithClientNotes(t *testing.T) {
	tmpDir := t.TempDir()

	settings := model.GlobalSetting{
		ConfigFilePath: tmpDir + "/wg0.conf",
	}
	server := model.Server{
		Interface: &model.ServerInterface{
			Addresses:  []string{"10.0.0.0/24"},
			ListenPort: 51820,
		},
		KeyPair: &model.ServerKeypair{
			PrivateKey: "privkey",
			PublicKey:  "pubkey",
		},
	}

	clients := []model.ClientData{
		{
			Client: &model.Client{
				ID:              "c1",
				Name:            "Client With Notes",
				PublicKey:       "clientpub",
				PresharedKey:    "clientpsk",
				AllocatedIPs:    []string{"10.0.0.2/32"},
				AllowedIPs:      []string{"0.0.0.0/0"},
				AdditionalNotes: "Line one\nLine two\nLine three",
				Enabled:         true,
			},
		},
	}

	err := WriteWireGuardServerConfig(os.DirFS("../templates"), server, clients, nil, settings)
	require.NoError(t, err)

	content, _ := os.ReadFile(tmpDir + "/wg0.conf")
	assert.Contains(t, string(content), "[Peer]")
}

func TestWriteWireGuardServerConfig_InvalidCustomTemplateFile(t *testing.T) {
	origWgConfTemplate := WgConfTemplate
	WgConfTemplate = "/nonexistent/template.conf"
	defer func() { WgConfTemplate = origWgConfTemplate }()

	settings := model.GlobalSetting{
		ConfigFilePath: "/tmp/wg0.conf",
	}
	server := model.Server{
		Interface: &model.ServerInterface{},
		KeyPair:   &model.ServerKeypair{},
	}

	err := WriteWireGuardServerConfig(os.DirFS("."), server, nil, nil, settings)
	assert.Error(t, err)
}

// --- StringFromEmbedFile ---

func TestWriteWireGuardServerConfig_InvalidConfigPath(t *testing.T) {
	settings := model.GlobalSetting{
		ConfigFilePath: "/nonexistent/dir/wg0.conf",
	}
	server := model.Server{
		Interface: &model.ServerInterface{
			Addresses:  []string{"10.0.0.0/24"},
			ListenPort: 51820,
		},
		KeyPair: &model.ServerKeypair{
			PrivateKey: "privkey",
			PublicKey:  "pubkey",
		},
	}

	err := WriteWireGuardServerConfig(os.DirFS("../templates"), server, nil, nil, settings)
	assert.Error(t, err)
}

func TestStringFromEmbedFile_NotFound(t *testing.T) {
	fsys := os.DirFS(t.TempDir())
	_, err := StringFromEmbedFile(fsys, "nonexistent.conf")
	assert.Error(t, err)
}

// --- GetDBUserCRC32 ---

func TestGetDBUserCRC32_Consistent(t *testing.T) {
	user := model.User{
		Username: "testuser",
		Email:    "test@example.com",
		Admin:    true,
	}

	hash1 := GetDBUserCRC32(user)
	hash2 := GetDBUserCRC32(user)
	assert.Equal(t, hash1, hash2, "CRC32 should be consistent for same input")
}

func TestGetDBUserCRC32_Different(t *testing.T) {
	user1 := model.User{Username: "user1", Email: "a@test.com"}
	user2 := model.User{Username: "user2", Email: "b@test.com"}

	hash1 := GetDBUserCRC32(user1)
	hash2 := GetDBUserCRC32(user2)
	assert.NotEqual(t, hash1, hash2)
}

// --- LookupEnvOrFile ---

func TestLookupEnvOrFile_WithFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := tmpDir + "/secret.txt"
	os.WriteFile(filePath, []byte("mysecret"), 0644)

	os.Setenv("TEST_SECRET_FILE", filePath)
	defer os.Unsetenv("TEST_SECRET_FILE")

	result := LookupEnvOrFile("TEST_SECRET_FILE", "default")
	assert.Equal(t, "mysecret", result)
}

func TestLookupEnvOrFile_FileNotFound(t *testing.T) {
	os.Setenv("TEST_SECRET_FILE", "/nonexistent/file")
	defer os.Unsetenv("TEST_SECRET_FILE")

	result := LookupEnvOrFile("TEST_SECRET_FILE", "default")
	assert.Equal(t, "default", result)
}

func TestLookupEnvOrFile_EnvNotSet(t *testing.T) {
	os.Unsetenv("TEST_SECRET_FILE_UNSET")
	result := LookupEnvOrFile("TEST_SECRET_FILE_UNSET", "mydefault")
	assert.Equal(t, "mydefault", result)
}

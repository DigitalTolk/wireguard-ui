package util

import (
	"fmt"

	"github.com/DigitalTolk/wireguard-ui/model"
)

// mockStore implements store.IStore for testing within the util package
type mockStore struct {
	clients        []model.ClientData
	server         model.Server
	globalSettings model.GlobalSetting
	users          []model.User
	hashes         model.ClientServerHashes
	allocatedIPs   []string
}

func newMockStore() *mockStore {
	return &mockStore{
		clients: []model.ClientData{},
		server: model.Server{
			Interface: &model.ServerInterface{
				Addresses:  []string{"10.0.0.0/24"},
				ListenPort: 51820,
			},
			KeyPair: &model.ServerKeypair{
				PrivateKey: "mockpriv",
				PublicKey:  "mockpub",
			},
		},
		globalSettings: model.GlobalSetting{
			EndpointAddress:     "10.0.0.1",
			DNSServers:          []string{"1.1.1.1"},
			MTU:                 1450,
			PersistentKeepalive: 15,
			FirewallMark:        "0xca6c",
			Table:               "auto",
			ConfigFilePath:      "/etc/wireguard/wg0.conf",
		},
		users:  []model.User{},
		hashes: model.ClientServerHashes{Client: "none", Server: "none"},
	}
}

func (m *mockStore) Init() error                     { return nil }
func (m *mockStore) GetUsers() ([]model.User, error) { return m.users, nil }
func (m *mockStore) GetUserByName(username string) (model.User, error) {
	for _, u := range m.users {
		if u.Username == username {
			return u, nil
		}
	}
	return model.User{}, fmt.Errorf("user not found")
}
func (m *mockStore) GetUserByOIDCSub(sub string) (model.User, error) {
	for _, u := range m.users {
		if u.OIDCSub == sub {
			return u, nil
		}
	}
	return model.User{}, fmt.Errorf("not found")
}
func (m *mockStore) SaveUser(user model.User) error {
	for i, u := range m.users {
		if u.Username == user.Username {
			m.users[i] = user
			return nil
		}
	}
	m.users = append(m.users, user)
	return nil
}
func (m *mockStore) DeleteUser(username string) error { return nil }
func (m *mockStore) GetGlobalSettings() (model.GlobalSetting, error) {
	return m.globalSettings, nil
}
func (m *mockStore) GetServer() (model.Server, error) { return m.server, nil }
func (m *mockStore) GetClients(hasQRCode bool) ([]model.ClientData, error) {
	return m.clients, nil
}
func (m *mockStore) GetClientByID(clientID string, qr model.QRCodeSettings) (model.ClientData, error) {
	for _, cd := range m.clients {
		if cd.Client.ID == clientID {
			return cd, nil
		}
	}
	return model.ClientData{}, fmt.Errorf("not found")
}
func (m *mockStore) SaveClient(client model.Client) error               { return nil }
func (m *mockStore) DeleteClient(clientID string) error                 { return nil }
func (m *mockStore) SaveServerInterface(si model.ServerInterface) error { return nil }
func (m *mockStore) SaveServerKeyPair(kp model.ServerKeypair) error     { return nil }
func (m *mockStore) SaveGlobalSettings(gs model.GlobalSetting) error    { return nil }
func (m *mockStore) GetAllocatedIPs(excludeClientID string) ([]string, error) {
	return m.allocatedIPs, nil
}
func (m *mockStore) GetWakeOnLanHosts() ([]model.WakeOnLanHost, error) {
	return nil, nil
}
func (m *mockStore) GetWakeOnLanHost(macAddress string) (*model.WakeOnLanHost, error) {
	return nil, nil
}
func (m *mockStore) DeleteWakeOnHostLanHost(macAddress string) error  { return nil }
func (m *mockStore) SaveWakeOnLanHost(host model.WakeOnLanHost) error { return nil }
func (m *mockStore) DeleteWakeOnHost(host model.WakeOnLanHost) error  { return nil }
func (m *mockStore) GetPath() string                                  { return "/tmp/mock" }
func (m *mockStore) SaveHashes(hashes model.ClientServerHashes) error {
	m.hashes = hashes
	return nil
}
func (m *mockStore) GetHashes() (model.ClientServerHashes, error) {
	return m.hashes, nil
}

package store

import (
	"errors"
	"time"

	"github.com/DigitalTolk/wireguard-ui/model"
)

// ErrAPITokenNotFound is returned by IStore implementations when an API token
// lookup or mutation targets a row that doesn't exist. Handlers use it to map
// onto HTTP 404 vs generic 500.
var ErrAPITokenNotFound = errors.New("api token not found")

type IStore interface {
	Init() error
	GetUsers() ([]model.User, error)
	GetUserByName(username string) (model.User, error)
	GetUserByOIDCSub(sub string) (model.User, error)
	SaveUser(user model.User) error
	DeleteUser(username string) error
	GetGlobalSettings() (model.GlobalSetting, error)
	GetServer() (model.Server, error)
	GetClients(hasQRCode bool) ([]model.ClientData, error)
	GetClientByID(clientID string, qrCode model.QRCodeSettings) (model.ClientData, error)
	SaveClient(client model.Client) error
	DeleteClient(clientID string) error
	SaveServerInterface(serverInterface model.ServerInterface) error
	SaveServerKeyPair(serverKeyPair model.ServerKeypair) error
	SaveGlobalSettings(globalSettings model.GlobalSetting) error
	GetAllocatedIPs(excludeClientID string) ([]string, error)
	GetWakeOnLanHosts() ([]model.WakeOnLanHost, error)
	GetWakeOnLanHost(macAddress string) (*model.WakeOnLanHost, error)
	DeleteWakeOnHostLanHost(macAddress string) error
	SaveWakeOnLanHost(host model.WakeOnLanHost) error
	DeleteWakeOnHost(host model.WakeOnLanHost) error
	GetPath() string
	SaveHashes(hashes model.ClientServerHashes) error
	GetHashes() (model.ClientServerHashes, error)

	// API tokens
	CreateAPIToken(token model.APIToken, tokenHash string) error
	ListAPITokens() ([]model.APIToken, error)
	GetAPITokenByHash(tokenHash string) (model.APIToken, error)
	RevokeAPIToken(id string) error
	TouchAPITokenLastUsed(id string, when time.Time) error
}

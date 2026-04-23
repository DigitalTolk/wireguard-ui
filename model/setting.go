package model

import (
	"time"
)

// GlobalSetting model
type GlobalSetting struct {
	EndpointAddress     string    `json:"endpoint_address"`
	DNSServers          []string  `json:"dns_servers"`
	MTU                 int       `json:"mtu"`
	PersistentKeepalive int       `json:"persistent_keepalive"`
	FirewallMark        string    `json:"firewall_mark"`
	Table               string    `json:"table"`
	ConfigFilePath      string    `json:"config_file_path"`
	UpdatedAt           time.Time `json:"updated_at"`
}

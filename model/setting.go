package model

import (
	"time"
)

// GlobalSetting model
type GlobalSetting struct {
	EndpointAddress         string    `json:"endpoint_address"`
	DNSServers              []string  `json:"dns_servers"`
	MTU                     int       `json:"mtu"`
	PersistentKeepalive     int       `json:"persistent_keepalive"`
	FirewallMark            string    `json:"firewall_mark"`
	Table                   string    `json:"table"`
	ConfigFilePath          string    `json:"config_file_path"`
	ClientNamePattern       string    `json:"client_name_pattern"`
	ClientNameReplacement   string    `json:"client_name_replacement"`
	EmailFilenamePattern    string    `json:"email_filename_pattern"`
	EmailFilenameReplacement string   `json:"email_filename_replacement"`
	UpdatedAt               time.Time `json:"updated_at"`
}

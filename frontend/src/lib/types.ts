export interface User {
  username: string;
  email: string;
  display_name: string;
  oidc_sub?: string;
  admin: boolean;
  created_at: string;
  updated_at: string;
}

export interface Client {
  id: string;
  private_key: string;
  public_key: string;
  preshared_key: string;
  name: string;
  email: string;
  telegram_userid: string;
  subnet_ranges: string[];
  allocated_ips: string[];
  allowed_ips: string[];
  extra_allowed_ips: string[];
  endpoint: string;
  additional_notes: string;
  use_server_dns: boolean;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface ClientData {
  Client: Client;
  QRCode: string;
}

export interface ServerKeypair {
  private_key: string;
  public_key: string;
  updated_at: string;
}

export interface ServerInterface {
  addresses: string[];
  listen_port: number;
  post_up: string;
  pre_down: string;
  post_down: string;
  updated_at: string;
}

export interface Server {
  KeyPair: ServerKeypair;
  Interface: ServerInterface;
}

export interface GlobalSetting {
  endpoint_address: string;
  dns_servers: string[];
  mtu: number;
  persistent_keepalive: number;
  firewall_mark: string;
  table: string;
  config_file_path: string;
  updated_at: string;
}

export interface WakeOnLanHost {
  MacAddress: string;
  Name: string;
  LatestUsed: string | null;
}

export interface PeerStatus {
  name: string;
  email: string;
  public_key: string;
  received_bytes: number;
  transmit_bytes: number;
  last_handshake_time: string;
  last_handshake_rel: number;
  connected: boolean;
  allocated_ip: string;
  endpoint?: string;
}

export interface DeviceStatus {
  name: string;
  peers: PeerStatus[];
}

export interface AuditLog {
  id: number;
  timestamp: string;
  actor: string;
  action: string;
  resource_type: string;
  resource_id: string;
  details: string;
  ip_address: string;
}

export interface MeResponse {
  username: string;
  email: string;
  display_name: string;
  admin: boolean;
}

export interface ClientDefaults {
  AllowedIps: string[];
  ExtraAllowedIps: string[];
  UseServerDNS: boolean;
  EnableAfterCreation: boolean;
}

export interface AppInfo {
  base_path: string;
  client_defaults: ClientDefaults;
}

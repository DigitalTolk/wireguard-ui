/**
 * Pure validation functions for WireGuard UI form inputs.
 */

/** Returns true if the string is a valid IPv4 address. */
export function isValidIPv4(ip: string): boolean {
  const parts = ip.split(".");
  if (parts.length !== 4) return false;
  return parts.every((part) => {
    if (!/^\d{1,3}$/.test(part)) return false;
    const n = Number(part);
    return n >= 0 && n <= 255;
  });
}

/** Returns true if the string is a valid IPv6 address (basic check). */
export function isValidIPv6(ip: string): boolean {
  // Handle :: shorthand and full form
  if (ip === "::") return true;
  const parts = ip.split("::");
  if (parts.length > 2) return false;

  const hexGroups: string[] = [];
  for (const part of parts) {
    if (part === "") continue;
    hexGroups.push(...part.split(":"));
  }

  if (hexGroups.length > 8) return false;
  if (parts.length === 1 && hexGroups.length !== 8) return false;
  if (parts.length === 2 && hexGroups.length >= 8) return false;

  return hexGroups.every((g) => /^[0-9a-fA-F]{1,4}$/.test(g));
}

/** Returns true if the string is a valid IP address (v4 or v6). */
export function isValidIP(ip: string): boolean {
  return isValidIPv4(ip.trim()) || isValidIPv6(ip.trim());
}

/** Returns true if the string is a valid CIDR notation (IPv4 or IPv6). */
export function isValidCIDR(cidr: string): boolean {
  const trimmed = cidr.trim();
  const slashIdx = trimmed.lastIndexOf("/");
  if (slashIdx === -1) return false;

  const ip = trimmed.substring(0, slashIdx);
  const prefixStr = trimmed.substring(slashIdx + 1);
  if (!/^\d{1,3}$/.test(prefixStr)) return false;
  const prefix = Number(prefixStr);

  if (isValidIPv4(ip)) {
    return prefix >= 0 && prefix <= 32;
  }
  if (isValidIPv6(ip)) {
    return prefix >= 0 && prefix <= 128;
  }
  return false;
}

/** Returns true if the string is a valid email address. */
export function isValidEmail(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim());
}

/** Returns true if the string is a valid MAC address (AA:BB:CC:DD:EE:FF or AA-BB-CC-DD-EE-FF). */
export function isValidMAC(mac: string): boolean {
  return /^([0-9A-Fa-f]{2}[:-]){5}[0-9A-Fa-f]{2}$/.test(mac.trim());
}

/** Returns true if the port number is valid (1-65535). */
export function isValidPort(port: number): boolean {
  return Number.isInteger(port) && port >= 1 && port <= 65535;
}

/**
 * Returns true if the input string is a valid comma-separated list of CIDR addresses.
 * Each entry must be a valid CIDR. Empty string returns false.
 */
export function isValidCIDRList(input: string): boolean {
  const items = input
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
  if (items.length === 0) return false;
  return items.every(isValidCIDR);
}

/**
 * Returns true if the input string is a valid comma-separated list of IP addresses.
 * Each entry must be a valid IP. Empty string returns false.
 */
export function isValidIPList(input: string): boolean {
  const items = input
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
  if (items.length === 0) return false;
  return items.every(isValidIP);
}

/**
 * Returns true if the string is a valid host:port or IP:port endpoint.
 * Accepts hostname:port, IPv4:port, or [IPv6]:port.
 */
export function isValidEndpoint(endpoint: string): boolean {
  const trimmed = endpoint.trim();
  if (!trimmed) return false;

  // [IPv6]:port
  const ipv6Match = trimmed.match(/^\[(.+)\]:(\d+)$/);
  if (ipv6Match) {
    return isValidIPv6(ipv6Match[1]) && isValidPort(Number(ipv6Match[2]));
  }

  // host:port or IPv4:port
  const lastColon = trimmed.lastIndexOf(":");
  if (lastColon === -1) return false;

  const host = trimmed.substring(0, lastColon);
  const portStr = trimmed.substring(lastColon + 1);
  if (!host || !/^\d+$/.test(portStr)) return false;

  const port = Number(portStr);
  if (!isValidPort(port)) return false;

  // host can be an IP or a hostname
  if (isValidIPv4(host)) return true;
  // Basic hostname validation
  return /^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*$/.test(
    host
  );
}

/**
 * Returns true if the value is a valid firewall mark (hex 0x... or decimal number).
 * Empty string is considered valid (optional field).
 */
export function isValidFirewallMark(value: string): boolean {
  const trimmed = value.trim();
  if (!trimmed) return true;
  if (/^0x[0-9a-fA-F]+$/.test(trimmed)) return true;
  if (/^\d+$/.test(trimmed)) return true;
  return false;
}

import { describe, it, expect } from "vitest";
import {
  isValidIPv4,
  isValidIPv6,
  isValidIP,
  isValidCIDR,
  isValidEmail,
  isValidMAC,
  isValidPort,
  isValidCIDRList,
  isValidIPList,
  isValidEndpoint,
  isValidFirewallMark,
} from "./validation";

describe("isValidIPv4", () => {
  it("accepts valid IPv4 addresses", () => {
    expect(isValidIPv4("0.0.0.0")).toBe(true);
    expect(isValidIPv4("127.0.0.1")).toBe(true);
    expect(isValidIPv4("192.168.1.1")).toBe(true);
    expect(isValidIPv4("255.255.255.255")).toBe(true);
    expect(isValidIPv4("10.0.0.1")).toBe(true);
  });

  it("rejects invalid IPv4 addresses", () => {
    expect(isValidIPv4("")).toBe(false);
    expect(isValidIPv4("256.0.0.1")).toBe(false);
    expect(isValidIPv4("1.2.3")).toBe(false);
    expect(isValidIPv4("1.2.3.4.5")).toBe(false);
    expect(isValidIPv4("abc.def.ghi.jkl")).toBe(false);
    expect(isValidIPv4("1.2.3.999")).toBe(false);
    expect(isValidIPv4("1.2.3.-1")).toBe(false);
    expect(isValidIPv4("::1")).toBe(false);
    // Note: leading zeros like "1.2.3.04" are accepted by the current regex
  });
});

describe("isValidIPv6", () => {
  it("accepts valid IPv6 addresses", () => {
    expect(isValidIPv6("::")).toBe(true);
    expect(isValidIPv6("::1")).toBe(true);
    expect(isValidIPv6("fe80::1")).toBe(true);
    expect(isValidIPv6("2001:0db8:85a3:0000:0000:8a2e:0370:7334")).toBe(true);
    expect(isValidIPv6("2001:db8::1")).toBe(true);
  });

  it("rejects invalid IPv6 addresses", () => {
    expect(isValidIPv6("")).toBe(false);
    expect(isValidIPv6(":::")).toBe(false);
    expect(isValidIPv6("1::2::3")).toBe(false);
    expect(isValidIPv6("gggg::1")).toBe(false);
    expect(isValidIPv6("192.168.1.1")).toBe(false);
    expect(isValidIPv6("1:2:3:4:5:6:7:8:9")).toBe(false);
  });
});

describe("isValidIP", () => {
  it("accepts IPv4 and IPv6", () => {
    expect(isValidIP("10.0.0.1")).toBe(true);
    expect(isValidIP("::1")).toBe(true);
    expect(isValidIP("  10.0.0.1  ")).toBe(true);
  });

  it("rejects invalid IPs", () => {
    expect(isValidIP("")).toBe(false);
    expect(isValidIP("not-an-ip")).toBe(false);
  });
});

describe("isValidCIDR", () => {
  it("accepts valid CIDR notation", () => {
    expect(isValidCIDR("10.0.0.0/8")).toBe(true);
    expect(isValidCIDR("192.168.1.0/24")).toBe(true);
    expect(isValidCIDR("10.0.0.2/32")).toBe(true);
    expect(isValidCIDR("0.0.0.0/0")).toBe(true);
    expect(isValidCIDR("::1/128")).toBe(true);
    expect(isValidCIDR("2001:db8::/32")).toBe(true);
    expect(isValidCIDR("  10.0.0.0/8  ")).toBe(true);
  });

  it("rejects invalid CIDR notation", () => {
    expect(isValidCIDR("")).toBe(false);
    expect(isValidCIDR("10.0.0.0")).toBe(false);
    expect(isValidCIDR("10.0.0.0/33")).toBe(false);
    expect(isValidCIDR("10.0.0.0/-1")).toBe(false);
    expect(isValidCIDR("10.0.0.0/abc")).toBe(false);
    expect(isValidCIDR("/24")).toBe(false);
    expect(isValidCIDR("not-cidr/24")).toBe(false);
    expect(isValidCIDR("::1/129")).toBe(false);
  });
});

describe("isValidEmail", () => {
  it("accepts valid email addresses", () => {
    expect(isValidEmail("test@example.com")).toBe(true);
    expect(isValidEmail("user.name@domain.co")).toBe(true);
    expect(isValidEmail("user+tag@example.org")).toBe(true);
    expect(isValidEmail("  test@example.com  ")).toBe(true);
  });

  it("rejects invalid email addresses", () => {
    expect(isValidEmail("")).toBe(false);
    expect(isValidEmail("not-an-email")).toBe(false);
    expect(isValidEmail("@example.com")).toBe(false);
    expect(isValidEmail("test@")).toBe(false);
    expect(isValidEmail("test @example.com")).toBe(false);
  });
});

describe("isValidMAC", () => {
  it("accepts valid MAC addresses", () => {
    expect(isValidMAC("00:11:22:33:44:55")).toBe(true);
    expect(isValidMAC("AA:BB:CC:DD:EE:FF")).toBe(true);
    expect(isValidMAC("aa:bb:cc:dd:ee:ff")).toBe(true);
    expect(isValidMAC("00-11-22-33-44-55")).toBe(true);
    expect(isValidMAC("  00:11:22:33:44:55  ")).toBe(true);
  });

  it("rejects invalid MAC addresses", () => {
    expect(isValidMAC("")).toBe(false);
    expect(isValidMAC("00:11:22:33:44")).toBe(false);
    expect(isValidMAC("00:11:22:33:44:55:66")).toBe(false);
    expect(isValidMAC("GG:11:22:33:44:55")).toBe(false);
    expect(isValidMAC("not-a-mac")).toBe(false);
  });
});

describe("isValidPort", () => {
  it("accepts valid ports", () => {
    expect(isValidPort(1)).toBe(true);
    expect(isValidPort(80)).toBe(true);
    expect(isValidPort(443)).toBe(true);
    expect(isValidPort(51820)).toBe(true);
    expect(isValidPort(65535)).toBe(true);
  });

  it("rejects invalid ports", () => {
    expect(isValidPort(0)).toBe(false);
    expect(isValidPort(-1)).toBe(false);
    expect(isValidPort(65536)).toBe(false);
    expect(isValidPort(1.5)).toBe(false);
    expect(isValidPort(NaN)).toBe(false);
  });
});

describe("isValidCIDRList", () => {
  it("accepts valid comma-separated CIDRs", () => {
    expect(isValidCIDRList("10.0.0.0/8")).toBe(true);
    expect(isValidCIDRList("10.0.0.0/8, 192.168.1.0/24")).toBe(true);
    expect(isValidCIDRList("0.0.0.0/0")).toBe(true);
  });

  it("rejects invalid CIDR lists", () => {
    expect(isValidCIDRList("")).toBe(false);
    expect(isValidCIDRList("10.0.0.0")).toBe(false);
    expect(isValidCIDRList("10.0.0.0/8, not-cidr")).toBe(false);
  });
});

describe("isValidIPList", () => {
  it("accepts valid comma-separated IPs", () => {
    expect(isValidIPList("10.0.0.1")).toBe(true);
    expect(isValidIPList("10.0.0.1, 192.168.1.1")).toBe(true);
    expect(isValidIPList("::1, 10.0.0.1")).toBe(true);
  });

  it("rejects invalid IP lists", () => {
    expect(isValidIPList("")).toBe(false);
    expect(isValidIPList("not-an-ip")).toBe(false);
    expect(isValidIPList("10.0.0.1, bad")).toBe(false);
  });
});

describe("isValidEndpoint", () => {
  it("accepts valid endpoints", () => {
    expect(isValidEndpoint("vpn.example.com:51820")).toBe(true);
    expect(isValidEndpoint("10.0.0.1:51820")).toBe(true);
    expect(isValidEndpoint("[::1]:51820")).toBe(true);
    expect(isValidEndpoint("my-server.example.com:443")).toBe(true);
    expect(isValidEndpoint("  vpn.example.com:51820  ")).toBe(true);
  });

  it("rejects invalid endpoints", () => {
    expect(isValidEndpoint("")).toBe(false);
    expect(isValidEndpoint("vpn.example.com")).toBe(false);
    expect(isValidEndpoint(":51820")).toBe(false);
    expect(isValidEndpoint("vpn.example.com:0")).toBe(false);
    expect(isValidEndpoint("vpn.example.com:65536")).toBe(false);
    expect(isValidEndpoint("vpn.example.com:abc")).toBe(false);
    expect(isValidEndpoint("[bad-ipv6]:51820")).toBe(false);
  });
});

describe("isValidFirewallMark", () => {
  it("accepts valid firewall marks", () => {
    expect(isValidFirewallMark("")).toBe(true);
    expect(isValidFirewallMark("  ")).toBe(true);
    expect(isValidFirewallMark("0xca6c")).toBe(true);
    expect(isValidFirewallMark("0xFF")).toBe(true);
    expect(isValidFirewallMark("51820")).toBe(true);
    expect(isValidFirewallMark("0")).toBe(true);
  });

  it("rejects invalid firewall marks", () => {
    expect(isValidFirewallMark("abc")).toBe(false);
    expect(isValidFirewallMark("0xZZ")).toBe(false);
    expect(isValidFirewallMark("-1")).toBe(false);
    expect(isValidFirewallMark("not-a-number")).toBe(false);
  });
});

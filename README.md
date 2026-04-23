# digitaltolk/wireguard-ui

A web user interface to manage your WireGuard setup.

Fork of [ngoduykhanh/wireguard-ui](https://github.com/ngoduykhanh/wireguard-ui).

## Features

- Modern React frontend built with shadcn/ui components (WCAG accessible)
- Single Sign-On via OpenID Connect (OIDC)
- Audit logging of all administrative actions
- SQLite database backend
- Client management with QR codes, email delivery, and Excel export
- Multi-platform Docker images (amd64, arm64, armv7)

## Quick Start

### Docker

```sh
docker pull digitaltolk/wireguard-ui
```

### Docker Compose

```yaml
services:
  wireguard-ui:
    image: digitaltolk/wireguard-ui:latest
    container_name: wireguard-ui
    cap_add:
      - NET_ADMIN
    network_mode: host
    environment:
      - WGUI_USERNAME=admin
      - WGUI_PASSWORD=admin
      - WGUI_MANAGE_START=true
      - WGUI_MANAGE_RESTART=true
    volumes:
      - ./db:/app/db
      - /etc/wireguard:/etc/wireguard
```

> The default username and password are `admin`. Change them to secure your setup.

See [`examples/docker-compose`](examples/docker-compose) for more configurations
(LinuxServer, BoringTun, host networking).

### Binary

Download the binary from the [Releases](https://github.com/DigitalTolk/wireguard-ui/releases) page:

```sh
./wireguard-ui
```

## Environment Variables

Refer to the environment variable tables in the upstream documentation or inspect
the source for a full list. Key variables:

| Variable | Description | Default |
|---|---|---|
| `BIND_ADDRESS` | Listen address | `0.0.0.0:80` |
| `SESSION_SECRET` | Secret for session cookies | N/A |
| `WGUI_USERNAME` | Initial admin username | `admin` |
| `WGUI_PASSWORD` | Initial admin password | `admin` |
| `WGUI_CONFIG_FILE_PATH` | WireGuard config path | `/etc/wireguard/wg0.conf` |
| `WGUI_MANAGE_START` | Start WireGuard with container | `false` |
| `WGUI_MANAGE_RESTART` | Restart WireGuard on config apply | `false` |

## Build

```sh
# Build everything (frontend + Go binary)
make build

# Build Docker image
docker build -t wireguard-ui .
```

## License

MIT. See [LICENSE](LICENSE).

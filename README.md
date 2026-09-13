# nexus-tui — TUI for Sonatype Nexus Repository 3

A read-mostly terminal UI for managing Nexus Repository 3. Built with Go + Bubble Tea. Single static binary. Linux only.

## Features

- Browse repositories → components/assets (two-pane)
- Search components by name/repository/format/version
- Tasks list with state/detail
- Admin read views: users, roles, privileges, blob stores
- Gated deletes: repository + user, typed confirmation required
- Gated cache invalidation for proxy/group repositories, typed confirmation required
- Config file + env overrides
- `--insecure` for self-signed certs
- Status header with writable/read-only indicator

## Build

```bash
go build -o nexus-tui ./cmd/nexus-tui
```

## Usage

```bash
./nexus-tui --profile prod --allow-writes
./nexus-tui --profile prod --insecure          # self-signed TLS
./nexus-tui                                    # env-only mode
```

## Config

`~/.config/nexus-tui/config.yaml`

```yaml
current: prod
profiles:
  prod:
    url: https://nexus.example.com
    username: admin
    insecure: false
```

Secrets (password/token) go in env only, never in config:

```
NEXUS_URL=https://nexus.example.com
NEXUS_USER=admin
NEXUS_PASS=secret
NEXUS_INSECURE=true
```

## Key bindings

| Key | Action |
|---|---|
| `1-4` | Switch view (Browse/Search/Tasks/Admin) |
| `j/k` or `up/down` | Move cursor |
| `enter` | Select / open |
| `tab` | Switch pane (browse) |
| `/` | Focus search |
| `d` | Delete (repo or user) |
| `i` | Invalidate cache (proxy/group repo) |
| `r` | Refresh |
| `esc` | Back |
| `q` | Quit |

## Testing

```bash
go test ./...
go vet ./...
```

No integration tests — unit tests use `httptest` with recorded JSON fixtures. Provide a live Nexus for manual smoke.

## License

TODO

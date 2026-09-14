# nexus-tui — TUI for Sonatype Nexus Repository 3

A read-mostly terminal UI for managing Nexus Repository 3. Built with Go + Bubble Tea. Single static binary. Linux only.

## Features

- Browse repositories → components/assets (two-pane)
- Search components by name/repository/format/version
- Tasks list with state/detail
- Admin read views: users, roles, privileges, blob stores
- Gated deletes: repository + user, typed confirmation required
- Gated cache invalidation for proxy/group repositories (no confirmation; cache re-populates on demand)
- Multi-instance: `ctrl+p` switches between configured profiles at runtime
- Config file + env overrides
- `--insecure` for self-signed certs
- Rich status header: Nexus server version (`swagger.json` `info.version`, Cloudflare-safe), profile/host, writable/read-only + freeze reason, aggregated health checks (`ok/total`), blob-store health + total free space (graceful narrow degrade)

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
  staging:
    url: https://staging.nexus.example.com
    username: admin
```

Switch between profiles at runtime with `ctrl+p` (needs 2+ profiles with a `url`).

Secrets (password/token) go in env only, never in config:

```
NEXUS_URL=https://nexus.example.com
NEXUS_USER=admin
NEXUS_PASS=secret
NEXUS_INSECURE=true
```

## Key bindings (k9s-style)

| Key | Action |
|---|---|
| `:` | Command bar (`:repos`, `:comp <repo>`, `:search <q>`, `:tasks`, `:users`, `:roles`, `:privs`, `:blobs`, `:health`, `:ctx`, `:q`) |
| `/` | Live filter current view (regex) |
| `enter` | Open (repos→components, ctx→switch) / describe row |
| `d` / `y` | Describe selected row |
| `o` / `O` | Cycle sort column / toggle direction |
| `e` | Edit search query (in search view) |
| `j/k` or `up/down` | Move cursor (`g`/`G` first/last) |
| `ctrl+d` | Delete (repo or user, typed confirm) |
| `i` | Invalidate cache (proxy/group repo, immediate) |
| `r` | Refresh |
| `ctrl+p` | Contexts (same as `:ctx`) |
| `?` | Help overlay |
| `esc` | Back (pop crumbs) |
| `q` | Quit |

Append `/<filter>` to pre-filter, e.g. `:repos /maven`. Legacy `1-5` shortcuts still work.

## Testing

```bash
go test ./...
go vet ./...
```

No integration tests — unit tests use `httptest` with recorded JSON fixtures. Provide a live Nexus for manual smoke.

## License

MIT — see [LICENSE](LICENSE).

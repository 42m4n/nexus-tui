# nexus-tui — AI agent notes

## Project conventions

- Go 1.26+, single `linux` binary target.
- Charm stack for TUI: `bubbletea`, `lipgloss`, `bubbles`.
- YAML config via `gopkg.in/yaml.v3` (one approved dep; prefer stdlib encoding/json for anything else).
- HTTP client in `internal/nexus/client.go` — all Nexus API calls go through it.
- Multi-instance: `main.go` builds one `*nexus.Client` per config profile; `ui.New(clients, current)` takes the map. `ctrl+p` opens the profile picker (`scrSwitch`); switching clears all cached state and reloads browse. Switching is session-only — never writes `current:` back to config.
- Read-only by default; writes gated by `Client.Writes` flag + typed confirmation in UI.
- Pagination uses continuation tokens; `getPage` caps at 1000 pages (ponytail: raise or stream if a repo ever exceeds ~100k items).

## Live test instance

- Config: `~/.config/nexus-tui/config.yaml` (profile `prod`); run `./nexus-tui` against it for live testing.
- Health screens worth checking there: `5` (status checks + blob stores), `1` (browse).
- Drive the TUI headless via tmux: `tmux new-session -d -s nxt -x 120 -y 40 './nexus-tui'`, then `tmux send-keys -t nxt 5` and `tmux capture-pane -t nxt -p`.
- Verify API shapes against the live server before writing types — swagger guesses caused two fixture bugs (see commit `67142c2`). No per-repo health endpoint exists on OSS Nexus; use `GET /v1/status/check` (needs `nexus:metrics:read`). Server version via `GET /service/rest/swagger.json` `info.version` (unauthenticated) — `Server: Nexus/...` header is stripped by Cloudflare.
- Never run write actions against the live instance (delete, invalidate) without explicit user request.

## Testing

- Unit tests with `httptest` + recorded JSON fixtures in `internal/nexus/client_test.go` and `internal/config/config_test.go`.
- No framework beyond stdlib `testing`.
- Table-driven tests with `t.Run` subtests; case names state the scenario.
- Test helpers stay in `_test.go` files, same package; no `internal/testutil`.
- Failure messages name got vs want: `t.Errorf("got %q, want %q", got, want)`.
- New non-trivial logic (branch, parser, HTTP path) ships with its test in the same change.
- Before every commit: `gofmt -l .` must print nothing, then `go test ./...` and `go vet ./...` must pass. Do not commit if any fails. Run these without being asked.

## Versioning

- SemVer: `feat` → minor, `fix` → patch, breaking change → major. Pre-1.0 may break freely.
- `var version = "dev"` in `cmd/nexus-tui/main.go`; release builds set it via
  `go build -ldflags "-X main.version=vX.Y.Z"`.
- Release = annotated tag only: `git tag -a vX.Y.Z -m "Release vX.Y.Z"`.
- Tag/bump only when asked — never tag unprompted.

## Commits

- Conventional Commits: `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `test:`.
- `type: lowercase imperative summary` — no scope needed at this repo size.
- One logical change per commit; never mix fix + refactor.

## Adding an API endpoint

1. Add type to `internal/nexus/types.go`.
2. Add method to `internal/nexus/client.go` using `do`/`getPage`.
3. Add message + command in `internal/ui/app.go`.
4. Add test case in `client_test.go`.

## UI patterns (k9s-style)

- `internal/ui/` split: `app.go` (Model, messages, Update, keys), `resources.go` (viewKind registry, aliases, rows, describe), `table.go` (regex filter, numeric-aware sort), `views.go` (header, crumbs, tables, help). Header caches `nxVer`/`checks`/`readOnly`/`blobs` via `loadOverview` at `Init` + on profile switch; degrades to dim `nx:-` when unloaded and drops the storage segment first on narrow screens.
- Navigation is a crumbs stack (`stack []viewState`); `enter` pushes (repos→components, any row→describe), `esc` pops. Per-view cursor/filter/sort live in `viewState`.
- `:` opens the command bar (`resolveAlias` in `resources.go`); `/` filters the current view; `?` help overlay. `ctrl+p` = `:ctx` shortcut.
- Keys: `d`/`y` describe, `ctrl+d` delete (typed confirm, `Writes`-gated), `o`/`O` sort cycle/toggle, `r` refresh. Search view is modal: `e` edits query, `esc` stops editing.
- Confirm modal for destructive actions (delete); `esc` cancels, `enter` only when typed name matches target. Cache invalidation is gated by `Writes` but needs no confirmation (non-destructive).

## Known simplifications (ponytail)

- Full admin CRUD (create/update forms) deferred — only read views + repo/user delete + proxy/group cache invalidation in v1.
- Component/asset delete deferred.
- Artifact upload deferred.
- Format-specific search fields (maven groupId, npm scope) deferred — generic name/version only.
- No Windows/macOS cross-compile.
- No Nexus Cloud support — self-hosted only.
- Config is YAML via `gopkg.in/yaml.v3`; secrets live in env vars only, never in config files.
- In-TUI profile editing (add/remove profiles) deferred — config stays hand-edited; switcher is read-only.

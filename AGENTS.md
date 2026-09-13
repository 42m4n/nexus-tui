# nexus-tui — AI agent notes

## Project conventions

- Go 1.26+, single `linux` binary target.
- Charm stack for TUI: `bubbletea`, `lipgloss`, `bubbles`.
- YAML config via `gopkg.in/yaml.v3` (one approved dep; prefer stdlib encoding/json for anything else).
- HTTP client in `internal/nexus/client.go` — all Nexus API calls go through it.
- Read-only by default; writes gated by `Client.Writes` flag + typed confirmation in UI.
- Pagination uses continuation tokens; `getPage` caps at 1000 pages (ponytail: raise or stream if a repo ever exceeds ~100k items).

## Testing

- Unit tests with `httptest` + recorded JSON fixtures in `internal/nexus/client_test.go` and `internal/config/config_test.go`.
- No framework beyond stdlib `testing`.
- `go test ./...` + `go vet ./...` must pass before merge.

## Adding an API endpoint

1. Add type to `internal/nexus/types.go`.
2. Add method to `internal/nexus/client.go` using `do`/`getPage`.
3. Add message + command in `internal/ui/app.go`.
4. Add test case in `client_test.go`.

## UI patterns

- `screen` enum controls active view.
- `focus` int toggles left/right panes in browse.
- `window[T]` helper renders scrollable lists; cursor stays in viewport.
- Confirm modal for all destructive/write actions (delete, invalidate cache); `esc` cancels, `enter` only when typed name matches target.

## Known simplifications (ponytail)

- Full admin CRUD (create/update forms) deferred — only read views + repo/user delete + proxy/group cache invalidation in v1.
- Component/asset delete deferred.
- Artifact upload deferred.
- Format-specific search fields (maven groupId, npm scope) deferred — generic name/version only.
- No Windows/macOS cross-compile.
- No Nexus Cloud support — self-hosted only.
- Config is YAML via `gopkg.in/yaml.v3`; secrets live in env vars only, never in config files.

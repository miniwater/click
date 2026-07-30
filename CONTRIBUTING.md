# Contributing to 全民打工

Thank you for considering contributing!

## Local Development

1. Ensure Go 1.26.3+ is installed.
2. Clone the repository.
3. Run `go run .` from the root directory.
4. Open http://localhost:3001 in your browser.

## Before Submitting a Pull Request

- Run `gofmt -w <changed .go files>`.
- Run `go vet ./...` and ensure zero warnings.
- Run `go test ./...` and ensure all tests pass.
- If you added a new WebSocket message type, update both `game/engine.go` (server handler) and `static/js/app.js` (client handler).
- If you changed persisted fields, add a migration case in `game/migration_test.go`.

## Facility Catalog Rules

- Facility IDs are persisted. **Append new IDs**; never renumber or delete existing ones without an explicit data migration.
- Keep `FacilityDefs` in order; `normalizeFacilities` relies on stable positions.

## Commit Messages

Use concise, descriptive commit messages in English or Chinese. There is no strict convention, but consistency is appreciated.

## Testing

Tests must never write to the live `data/game.db`. Use `t.TempDir()` for temporary databases.

```go
store, err := NewStore(filepath.Join(t.TempDir(), "game.db"))
```

## Code of Conduct

Be respectful and constructive. This is a small open-source project — kindness goes a long way.

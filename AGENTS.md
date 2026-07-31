# Repository Guide

## Commands

- This module requires Go 1.26.5 (`go.mod`). Run commands from the repository root; the server opens `data/game.db` relative to the current working directory.
- Start locally with `go run .`; it listens on `:3001` unless `PORT` is set.
- Run all checks with `gofmt -w <changed .go files>`, then `go vet ./...`, then `go test ./...`.
- Run the only test package with `go test ./game`; focus a test with `go test ./game -run '^TestName$'`.
- Build the embedded web app with `go build -trimpath -ldflags="-s -w" -o click .`. For cross-builds, set both `GOOS` and an appropriate output name explicitly, for example `$env:GOOS='linux'; go build -trimpath -ldflags='-s -w' -o click-linux .` in PowerShell.
- `click`, `click-linux`, `click.exe`, and `data/game.db` are ignored runtime/build artifacts. Do not edit or commit them.

## Architecture

- `main.go` is the composition root: it embeds `templates/*` and `static/*`, opens SQLite, starts the game engine and WebSocket hub, and serves `/` plus `/ws`.
- `game/engine.go` owns the shared game state and inbound WebSocket actions. `game/hub.go` owns client lifecycle and outbound message envelopes. `static/js/app.js` is the matching browser protocol client; keep message types and payload fields synchronized across these files.
- The game is global, not per-user: every connected client reads and mutates the same `Engine`, while IP-derived names/colors identify participants.
- `game/store.go` owns schema creation and in-place migration. Preserve compatibility with existing `data/game.db` files when changing persisted fields, and add migration coverage in `game/migration_test.go`.
- `game/facilities.go` is the authoritative facility catalog and economy formula source; the browser renders definitions received in server snapshots rather than maintaining a second catalog.

## Invariants

- Currency uses the fixed 128-bit `Amount` type and bounded scientific strings at persistence/JSON boundaries. Keep legacy plain-decimal loading compatible; do not use SQLite `REAL` or parse whole amounts as JavaScript `Number`. `static/js/app.js` compares mantissa/exponent strings.
- Facility state order is normalized against `FacilityDefs`; IDs are persisted. Append new unique IDs and do not renumber existing facilities without an explicit data migration.
- Web assets are compiled into the executable via `go:embed`; rebuild/restart after changing templates, CSS, or JavaScript. `webAssetVersion` hashes only `static/css/style.css` and `static/js/app.js`, so update that list if another independently cached asset is added.
- State saves every five seconds only when dirty and is force-saved on SIGINT/SIGTERM. Tests should use `t.TempDir()` stores, never the repository's live `data/game.db`.

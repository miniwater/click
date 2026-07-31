# 💼 QuanMinDaGong · Clicker Game

[![Go version](https://img.shields.io/badge/go-1.26.5-blue)](go.mod)
[![CI](https://github.com/miniwater/click/actions/workflows/ci.yml/badge.svg)](https://github.com/miniwater/click/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

> **中文文档** → [README.zh-CN.md](README.zh-CN.md)

**Live Demo:** [https://nocodb.krjojo.com/](https://nocodb.krjojo.com/)

A real-time multiplayer idle clicker game where everyone shares the same game world. Click to earn gold, buy facilities, enhance with diamonds, and work your way from a tiny office to a multiversal corporation.

## Features

- **Multiplayer by Default** — All connected players share and mutate the same global game state in real time.
- **Click to Earn** — Tap the big coin to earn gold. Upgrade your click power.
- **60 Facilities** — 30 workplace-themed (AI, servers, space colonies) + 30 cultivation-themed (Qi, tribulations, immortal realms).
- **Diamond Bonus** — Each click has a 1 % chance to drop a diamond. Use diamonds to enhance facilities (+1 % per level).
- **Auto Income** — Facilities produce gold every tick. Income accumulates even while you are offline (up to 7 days).
- **Live Chat** — Real-time chat panel. The last 200 messages are persisted.
- **IP-based Identity** — Your display name is a masked IP and your theme colour is deterministically derived from your IP.
- **Shared Progress** — Every purchase and upgrade is visible to all connected players instantly.
- **Embedded Web UI** — All HTML, CSS and JavaScript are compiled into a single binary via `go:embed`.

## Screenshot

![pc](./data/pc.avif)

![moblie](./data/mobile.avif)

## Game Mechanics

### Currencies

| Currency | Use | Source |
|----------|-----|--------|
| Gold 🪙 | Buy facilities, upgrade click power | Click the big coin, auto-income from facilities |
| Diamond 💎 | Enhance facilities (×1.01 per level) | Random drop (1 % per click) |

### Click Power

Upgrade with gold. Cost and power both grow by ×1.05 per level.

### Facilities

60 facilities, each with:

- **Base cost** (gold)
- **Base CPS** (coins per second)
- **Cost growth multiplier** (1.05 / 1.25 / 1.50 / 1.75 / 2.0 depending on tier)
- **Enhance multiplier** (×1.01 per diamond spent)

### Offline Earnings

When the server restarts, it computes up to 7 days of missed CPS income and adds it to the gold pool.

## Tech Stack

- **Language** Go 1.26.5
- **HTTP Framework** [Gin](https://github.com/gin-gonic/gin)
- **WebSocket** [gorilla/websocket](https://github.com/gorilla/websocket)
- **Database** SQLite via [modernc.org/sqlite](https://modernc.org/sqlite) (pure-Go, no CGO needed)
- **Templating** `html/template`
- **Frontend** Vanilla JavaScript + CSS (embedded in the binary)
- **Financial Arithmetic** Custom 128-bit `Amount` type using `math/big.Float`, serialized as scientific notation strings

## Architecture

```
main.go
├── game/engine.go   — Shared game state, action handlers, tick loop
├── game/hub.go      — WebSocket client lifecycle, message broadcast
├── game/store.go    — SQLite persistence (schema, load, save, chat)
├── game/ip.go       — IP masking and deterministic colour derivation
├── game/facilities.go — Facility definitions and economy formulas
├── game/amount.go   — 128-bit Amount type with scientific-notation IO
├── static/js/app.js — Browser WebSocket client
├── static/css/style.css — Styles
└── templates/index.html — HTML shell
```

The game is **global, not per-user**. Every WebSocket client reads and mutates a single `Engine` instance protected by a mutex. State snapshots are broadcast to all connected clients every second.

## Quick Start

### Prerequisites

- Go 1.26.5+

### Run

```bash
# From the repository root
go run .
```

The server listens on `:3001` by default.

### Open

Visit [http://localhost:3001](http://localhost:3001).

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3001` | HTTP listen port |
| `TRUST_PROXY` | `false` | Trust `X-Forwarded-For`/`X-Real-IP`; enable only when the server is reachable exclusively through a trusted reverse proxy |

The database path is fixed to `data/game.db` relative to the working directory. WebSocket protection limits a single process to 10,000 connections and each source IP to 40 connections. These are safety ceilings, not capacity guarantees; load-test the target host before production traffic.

## Data & Persistence

- Game state is saved to `data/game.db` (SQLite) every 5 seconds if dirty, and force-saved on `SIGINT`/`SIGTERM`.
- Chat messages are persisted; only the most recent 200 rows are kept.
- Legacy databases using a `REAL` gold column are automatically migrated to the bounded Amount format.
- **Do not edit `data/game.db` by hand** while the server is running.

### Container / Deployment

Persist the entire `data/` directory across restarts to preserve progress.

When deploying behind a reverse proxy, keep the application port private and set `TRUST_PROXY=true`. Do not enable it when clients can reach the application directly, because clients could spoof forwarding headers and bypass IP-based controls.

## Building

```bash
# Native
go build -trimpath -ldflags="-s -w" -o click .

# Windows (cross from Linux/macOS)
GOOS=windows go build -trimpath -ldflags="-s -w" -o click.exe .

# Linux
GOOS=linux go build -trimpath -ldflags="-s -w" -o click-linux .
```

The binary is self-contained — no external assets need to be copied alongside it.

## Testing

```bash
go vet ./...
go test ./...
go test ./game -run '^TestName$'   # run a specific test
```

Tests use `t.TempDir()` stores and never touch the live `data/game.db`.

## Project Structure

```
.
├── main.go                  # Entry point, routing, asset embedding
├── go.mod / go.sum          # Dependencies
├── game/
│   ├── engine.go            # Core game loop, action handlers
│   ├── hub.go               # WebSocket client registry, message dispatch
│   ├── store.go             # SQLite schema, CRUD, migration
│   ├── ip.go                # IP masking, theme colour
│   ├── facilities.go        # Facility catalog, economy formulas
│   ├── amount.go            # 128-bit Amount type
│   ├── amount_test.go
│   ├── engine_cache_test.go
│   ├── click_aggregation_test.go
│   └── migration_test.go
├── static/
│   ├── css/style.css
│   └── js/app.js
├── templates/
│   └── index.html
├── data/                    # SQLite database (ignored, created at runtime)
├── .github/workflows/ci.yml
├── .gitattributes
├── .gitignore
├── AGENTS.md
├── CONTRIBUTING.md
├── LICENSE
├── SECURITY.md
├── README.md
└── README.zh-CN.md
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Security

See [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE) © 2026 miniwater

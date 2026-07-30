# 全民打工 · 放置点击游戏

[![Go version](https://img.shields.io/badge/go-1.26.3-blue)](go.mod)
[![CI](https://github.com/miniwater/click/actions/workflows/ci.yml/badge.svg)](https://github.com/miniwater/click/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

> **English** → [README.md](README.md)

一款实时多人在线放置点击游戏。所有玩家共享同一个游戏世界，点击赚金币、购买设施、用钻石强化，一起从打工起步走向多元宇宙集团。

## 功能特点

- **全球共享** — 所有在线玩家实时操作同一份游戏状态，任何人的升级、购买都会即时同步给所有人。
- **点击赚钱** — 点击大金币赚取金币，可升级点击收益。
- **60 种设施** — 30 个打工主题设施（AI、服务器、太空殖民）+ 30 个修仙主题设施（灵气、渡劫、圣人道统）。
- **钻石系统** — 每次点击有 1 % 概率获得钻石，用于强化设施（每级 +1 % 收益）。
- **自动产出** — 设施每秒自动产出金币，离线最多累积 7 天收益。
- **实时聊天** — 内置聊天面板，消息持久化，保留最近 200 条。
- **IP 身份** — 显示名为脱敏 IP，主题色由 IP 稳定生成。
- **单文件部署** — HTML、CSS、JS 全部编译进可执行文件，无需额外依赖。

## 截图

![pc](./data/pc.avif)

![moblie](./data/mobile.avif)

## 游戏机制

### 货币

| 货币 | 用途 | 获取方式 |
|------|------|----------|
| 金币 🪙 | 购买设施、升级点击 | 点击大金币、设施自动产出 |
| 钻石 💎 | 强化设施（每级 ×1.01） | 点击随机掉落（概率 1 %） |

### 点击收益

消耗金币升级，成本和收益均按 ×1.05 递增。

### 设施

60 种设施各有：

- **基础价格**（金币）
- **基础产速**（金币 / 秒）
- **成长倍率**（根据档位为 1.05 / 1.25 / 1.50 / 1.75 / 2.0）
- **强化倍率**（每消耗 1 钻石 ×1.01）

### 离线收益

服务器重启时会计算最多 7 天的离线 CPS 并一次性添加到金币池。

## 技术栈

- **语言** Go 1.26.3
- **HTTP 框架** [Gin](https://github.com/gin-gonic/gin)
- **WebSocket** [gorilla/websocket](https://github.com/gorilla/websocket)
- **数据库** SQLite via [modernc.org/sqlite](https://modernc.org/sqlite)（纯 Go，无需 CGO）
- **模板** `html/template`
- **前端** 原生 JavaScript + CSS（嵌入二进制文件）
- **金额运算** 自定义 128 位 `Amount` 类型（`math/big.Float`），以科学计数法字符串序列化

## 架构

```
main.go
├── game/engine.go   — 核心游戏循环、操作处理器
├── game/hub.go      — WebSocket 客户端生命周期与消息广播
├── game/store.go    — SQLite 持久化（建表、读写、迁移）
├── game/ip.go       — IP 脱敏与主题色生成
├── game/facilities.go — 设施定义与经济公式
├── game/amount.go   — 128 位金额计算类型
├── static/js/app.js — 浏览器 WebSocket 客户端
├── static/css/style.css — 样式
└── templates/index.html — HTML 模板
```

游戏状态是**全局共享**的，所有 WebSocket 客户端读写同一个受互斥锁保护的 `Engine` 实例，状态快照每秒向所有在线客户端广播。

## 快速开始

### 环境要求

- Go 1.26.3+

### 启动

```bash
# 在仓库根目录执行
go run .
```

服务默认监听 `:3001`。

### 打开

访问 [http://localhost:3001](http://localhost:3001)

## 配置

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| `PORT`   | `3001` | HTTP 监听端口 |

数据库路径固定为相对于工作目录的 `data/game.db`。

## 数据与持久化

- 游戏状态每 5 秒保存一次（仅脏数据时），收到 `SIGINT`/`SIGTERM` 时强制保存。
- 聊天消息持久化，仅保留最近 200 条。
- 旧版 `REAL` 类型金币列会自动迁移为精确保存格式。
- **服务运行时请勿手动编辑 `data/game.db`**。

### 容器 / 部署

跨重启需要持久化 `data/` 整个目录。

## 构建

```bash
# 本机
go build -trimpath -ldflags="-s -w" -o click .

# Windows
$env:GOOS = "windows"
go build -trimpath -ldflags="-s -w" -o click.exe .

# Linux
$env:GOOS = "linux"
go build -trimpath -ldflags="-s -w" -o click-linux .
```

产物为单文件，无需复制外部资源。

## 测试

```bash
go vet ./...
go test ./...
go test ./game -run '^TestName$'   # 运行单个测试
```

测试均使用 `t.TempDir()` 创建临时数据库，不会读写运行态 `data/game.db`。

## 项目结构

```
.
├── main.go                  # 入口、路由、静态资源嵌入
├── go.mod / go.sum          # 依赖管理
├── game/
│   ├── engine.go            # 核心循环、操作处理
│   ├── hub.go               # WebSocket 客户端注册与消息分发
│   ├── store.go             # SQLite 建表、读写、迁移
│   ├── ip.go                # IP 脱敏与主题色
│   ├── facilities.go        # 设施目录、经济公式
│   ├── amount.go            # 128 位金额类型
│   ├── amount_test.go
│   ├── engine_cache_test.go
│   ├── click_aggregation_test.go
│   └── migration_test.go
├── static/
│   ├── css/style.css
│   └── js/app.js
├── templates/
│   └── index.html
├── data/                    # SQLite 数据库（运行时生成，已忽略）
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

## 贡献

参见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## 安全

参见 [SECURITY.md](SECURITY.md)。

## 许可证

[MIT](LICENSE) © 2026 miniwater

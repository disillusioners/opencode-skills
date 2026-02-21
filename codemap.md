# Repository Atlas: opencode-skills

## Project Responsibility

A **Go-based daemon-client application** for controlling OpenCode AI agents (Sisyphus, Prometheus, Atlas) via web API. Implements a persistent TCP daemon with session management, enabling external systems to orchestrate AI coding agents through a JSON-based protocol.

**Architecture**: Daemon-Client model with SQLite-backed session persistence

## System Entry Points

| Entry Point | Purpose |
|-------------|---------|
| `opencode-web-skill/opencode_skill_src/main.go` | CLI entry point with dual-mode execution (daemon/client) |
| `opencode-web-skill/opencode_skill_src/Makefile` | Build/install commands (`make build`, `make install`) |
| `opencode-web-skill/opencode_skill_src/go.mod` | Go 1.22.5 module definition |

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CLI Layer (main.go)                             │
│  Commands: start | stop | restart | init-session | prompt | command | answer │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           TCP Client (internal/client)                       │
│  JSON-encoded requests over TCP socket                                       │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Daemon Server (internal/daemon)                     │
│  ┌───────────────┐  ┌──────────────────┐  ┌─────────────────────────────┐   │
│  │ TCP Listener  │  │ Session Registry │  │ Session Managers (per ID)   │   │
│  │ (port 44111)  │  │ (SQLite DB)      │  │ State Machine Orchestrator  │   │
│  └───────────────┘  └──────────────────┘  └─────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          API Client (internal/api)                           │
│  HTTP REST calls to OpenCode service (127.0.0.1:4096)                        │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          OpenCode AI Service                                 │
│  Agents: Sisyphus | Prometheus | Atlas                                       │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Directory Map (Aggregated)

| Directory | Responsibility Summary | Detailed Map |
|-----------|------------------------|--------------|
| `opencode_skill_src/` | CLI gateway with dual-mode execution (daemon/client), session orchestration, message pipeline coordination | [View Map](opencode-web-skill/opencode_skill_src/codemap.md) |
| `internal/api/` | HTTP client adapter for OpenCode API communication with facade pattern | [View Map](opencode-web-skill/opencode_skill_src/internal/api/codemap.md) |
| `internal/client/` | Daemon TCP client with JSON serialization, auto-daemon-start, async operation monitoring | [View Map](opencode-web-skill/opencode_skill_src/internal/client/codemap.md) |
| `internal/config/` | Centralized configuration provider with static constants and dynamic path resolution | [View Map](opencode-web-skill/opencode_skill_src/internal/config/codemap.md) |
| `internal/daemon/` | Persistent TCP server with session lifecycle management and SQLite-backed registry | [View Map](opencode-web-skill/opencode_skill_src/internal/daemon/codemap.md) |
| `internal/manager/` | Per-session state machine orchestrator with async worker pattern and Observer callbacks | [View Map](opencode-web-skill/opencode_skill_src/internal/manager/codemap.md) |
| `internal/types/` | Shared Data Transfer Objects (DTOs) for inter-component communication | [View Map](opencode-web-skill/opencode_skill_src/internal/types/codemap.md) |

## Key Design Patterns

| Pattern | Location | Purpose |
|---------|----------|---------|
| **Repository** | `internal/daemon/registry.go` | SQLite abstraction for session CRUD |
| **State Machine** | `internal/manager/` | IDLE → BUSY → WAITING_FOR_INPUT transitions |
| **Observer** | `internal/daemon/` + `internal/manager/` | State change persistence callbacks |
| **Facade** | `internal/api/client.go` | Simplified OpenCode API interface |
| **Command** | `main.go`, `internal/daemon/` | Action-based request routing |
| **Producer-Consumer** | `internal/manager/` | Buffered channel for request queuing |

## Data Flow Summary

```
User Input → CLI Parser → TCP Client → Daemon Server → Session Manager → API Client → OpenCode AI
                    ↑                                                              ↓
                    └──────────── JSON Response ← State Update ← Questions ←──────┘
```

## Key Integration Points

### Internal Dependencies
- `types` → consumed by all packages (shared DTOs)
- `config` → consumed by all packages (constants, paths)
- `api` → consumed by `manager` (OpenCode API)
- `manager` → consumed by `daemon` (session state)
- `daemon` → consumed by `client` (TCP protocol)
- `client` → consumed by `main.go` (CLI interface)

### External Systems
- **OpenCode API**: `http://127.0.0.1:4096` (AI agent service)
- **SQLite**: `~/.opencode_skill/sessions.db` (session persistence)
- **File System**: PID file, working directories
- **TCP Network**: Port 44111 (daemon communication)

## Configuration Reference

| Constant | Value | Purpose |
|----------|-------|---------|
| `OpenCodeURL` | `http://127.0.0.1:4096` | AI service endpoint |
| `DaemonPort` | `44111` | TCP daemon port |
| `DefaultAgent` | `sisyphus` | Default AI agent |
| `PollInterval` | `2s` | State polling interval |
| `ClientTimeout` | `120s` | Client operation timeout |

## Build & Run Commands

```bash
cd opencode-web-skill/opencode_skill_src

# Build
make build

# Install (build + restart daemon)
make install

# Daemon management
make start      # Start daemon
make stop       # Stop daemon
make restart    # Restart daemon

# Run tests
go test ./...
go test -v ./...
```

## Related Documentation

| Document | Purpose |
|----------|---------|
| `AGENTS.md` | Project overview and AI agent guidelines |
| `opencode-web-skill/SKILL.md` | User-facing skill documentation |
| `OPENCODE_QUESTION_WORKFLOW.md` | Question/answer API documentation |

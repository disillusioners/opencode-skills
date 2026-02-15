# AGENTS.md

This repository contains the **OpenCode Web Skill** - a Go-based daemon-client application for controlling OpenCode AI agents (Sisyphus, Prometheus, Atlas) via web API.

## Project Structure

```
opencode-skills/
├── opencode-web-skill/          # Main Go project
│   ├── SKILL.md                 # Skill documentation and usage guide
│   └── opencode_skill_src/      # Go source code
│       ├── main.go              # CLI entry point
│       ├── main_test.go         # CLI tests
│       ├── Makefile             # Build/install commands
│       ├── go.mod               # Go 1.22.5
│       └── internal/            # Internal packages
│           ├── api/             # OpenCode API client
│           ├── client/          # Daemon TCP client
│           ├── config/          # Configuration constants
│           ├── daemon/          # Daemon server & registry
│           ├── manager/         # Session manager
│           ├── testutil/        # Test utilities
│           └── types/           # Shared request/response types
├── test_proj/                   # Test projects
└── OPENCODE_QUESTION_WORKFLOW.md # Question/answer API documentation
```

## Build Commands

```bash
# Navigate to the Go source directory
cd opencode-web-skill/opencode_skill_src

# Build the binary
make build
# Or: go build -o opencode_skill

# Install to ~/bin (builds, stops old daemon, starts new)
make install

# Daemon management
make start      # Start daemon
make stop       # Stop daemon
make restart    # Restart daemon

# Clean build artifacts
make clean
```

## Test Commands

```bash
cd opencode-web-skill/opencode_skill_src

# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test ./internal/client
go test ./internal/daemon

# Run a single test
go test -v -run TestClient_NewClient ./internal/client
go test -v -run TestRegistry ./internal/daemon

# Run tests with coverage
go test -cover ./...

# Run short tests only (skip integration tests)
go test -short ./...
```

## Lint Commands

```bash
# Format code
go fmt ./...

# Vet code (static analysis)
go vet ./...

# Run both
go fmt ./... && go vet ./...
```

## Code Style Guidelines

### General Principles

- **Go version**: 1.22.5
- **Module name**: `opencode_skill`
- **Package structure**: Internal packages under `internal/`
- **Keep functions focused**: One responsibility per function
- **Prefer composition**: Small, composable functions over large monolithic ones
- **Handle errors explicitly**: Return errors, don't panic in library code

### Imports

```go
// Standard library first
import (
    "encoding/json"
    "fmt"
    "net"

    // Third-party packages second
    "github.com/mattn/go-sqlite3"

    // Local packages last
    "opencode_skill/internal/config"
    "opencode_skill/internal/types"
)
```

### Naming Conventions

- **Packages**: lowercase, single word preferred (`client`, `daemon`, `api`)
- **Types**: PascalCase (`SessionData`, `Registry`, `Client`)
- **Functions/Methods**: PascalCase if exported, camelCase if private
- **Constants**: PascalCase for exported, camelCase for private
- **Interfaces**: typically end with `-er` (`Reader`, `Writer`)
- **Acronyms**: Keep consistent case (`HTTP`, `URL`, `ID` not `Id`)

```go
// Good
type SessionData struct { ... }
func (c *Client) SendRequest(...) { ... }
const DefaultAgent = "sisyphus"

// Private
func getString(m map, key string) string { ... }
const daemonPort = 44111
```

### Struct Definitions

```go
// Group related fields, use consistent ordering
type SessionData struct {
    Project     string `json:"project"`
    SessionName string `json:"session_name"`
    ID          string `json:"session_id"`
    WorkingDir  string `json:"working_dir"`
}

// Use struct tags for JSON/DB serialization
type PromptRequest struct {
    Agent string       `json:"agent"`
    Model ModelDetails `json:"model"`
    Parts []Part       `json:"parts"`
}
```

### Error Handling

- Return errors as the last return value
- Use `errors.New()` for static errors, `fmt.Errorf()` for formatted
- Define sentinel errors at package level

```go
// Sentinel errors
var (
    ErrNotFound  = errors.New("session not found")
    ErrDuplicate = errors.New("session already exists")
)

// Error returns
func (r *Registry) Get(project, sessionName string) (*SessionData, error) {
    // ...
    if err == sql.ErrNoRows {
        return nil, ErrNotFound
    }
    return &session, nil
}

// Wrap errors with context
if err := client.Connect(); err != nil {
    return fmt.Errorf("failed to connect: %w", err)
}
```

### Control Flow

- Prefer early returns over nested if-else
- Use switch for multiple conditions

```go
// Good - early returns
func (s *Server) handleConnection(conn net.Conn) {
    defer conn.Close()
    
    buf := make([]byte, 4096)
    n, err := conn.Read(buf)
    if err != nil {
        return
    }
    // ... continue processing
}

// Good - switch for actions
switch req.Action {
case "PING":
    response = map[string]interface{}{"status": "ok"}
case "START_SESSION":
    // handle
default:
    response = map[string]interface{}{"status": "error"}
}
```

### Testing

- Place tests in same package with `_test.go` suffix
- Use table-driven tests for multiple cases
- Always call `t.Parallel()` at the start of tests where possible
- Use `t.Helper()` for helper functions

```go
func TestClient_fullSessionRef(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name   string
        client *Client
        want   string
    }{
        {
            name:   "with project and session name",
            client: NewClientWithMeta("id-123", "proj", "session"),
            want:   "proj session",
        },
        // ... more cases
    }

    for _, tt := range tests {
        tt := tt  // Capture range variable
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            got := tt.client.fullSessionRef()
            if got != tt.want {
                t.Errorf("fullSessionRef() = %q, want %q", got, tt.want)
            }
        })
    }
}

// Helper function pattern
func newTestServer(t *testing.T, port int) *testServer {
    t.Helper()
    // ...
}
```

### Concurrency

- Use `sync.Mutex` for shared state
- Use `sync.WaitGroup` for goroutine coordination
- Prefer channels for communication

```go
type Registry struct {
    db *sql.DB
    mu sync.Mutex
}

func (r *Registry) Create(...) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    // ... thread-safe operations
}
```

### Defer Usage

- Use `defer` for cleanup (Close, Unlock, etc.)
- Place defer immediately after acquiring resource

```go
func (r *Registry) List() ([]SessionData, error) {
    rows, err := r.db.Query(...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    // ...
}
```

## Architecture Notes

### Daemon-Client Architecture

1. **Daemon** (`internal/daemon/`): Long-running TCP server that manages sessions
2. **Client** (`internal/client/`): CLI that communicates with daemon via TCP
3. **Registry** (`internal/daemon/registry.go`): SQLite-backed session persistence
4. **Manager** (`internal/manager/`): Per-session state machine for request handling

### Key Flows

- **Init Session**: CLI → Daemon → Registry (store) → OpenCode API (create session)
- **Send Prompt**: CLI → Daemon → Manager (queue) → OpenCode API → Wait for result
- **Questions**: OpenCode API → Manager → Daemon → CLI (display to user)

## Important Files

| File | Purpose |
|------|---------|
| `opencode-web-skill/SKILL.md` | User-facing documentation for the skill |
| `OPENCODE_QUESTION_WORKFLOW.md` | API documentation for question/answer flow |
| `internal/config/config.go` | All configuration constants |
| `internal/types/types.go` | Shared request/response structs |

## Notes for AI Agents

1. **Always navigate to `opencode-web-skill/opencode_skill_src` before running Go commands**
2. **The daemon must be running for CLI commands to work** - use `make start` or client auto-starts
3. **Tests use `t.TempDir()` for isolated temp directories** - automatic cleanup
4. **TCP communication uses JSON encoding** - all messages are JSON
5. **Session identification uses project + session_name** - not just session ID

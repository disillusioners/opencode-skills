# Codemap: Configuration Package

## Responsibility

The `config` package serves as the **centralized configuration management module** for the OpenCode Web Skill application. It implements a **configuration provider pattern** responsible for:

- **Static Configuration**: Defining compile-time constants for OpenCode API endpoints, daemon settings, and timing parameters
- **Dynamic Path Resolution**: Determining runtime file system paths through environment-aware logic
- **Initialization Management**: Providing package-level initialization via `init()` function for setup of directory structures and path configuration
- **Project Root Detection**: Implementing upward traversal logic to locate project boundaries using `.git` markers

## Design Patterns

### 1. Constant Provider Pattern
```go
// OpenCode Configuration
const (
    OpenCodeURL  = "http://127.0.0.1:4096"
    DefaultAgent = "sisyphus"
    DefaultModel = "zai-coding-plan/glm-5"
)
```
**Pattern Implementation**: Immutable compile-time constants with semantic grouping.

### 2. Lazy Initialization Pattern
```go
var (
    ProjectRoot    string
    WrapperDir     string
    PidFile        string
    SessionMapFile string
)

func init() {
    // Late-bound initialization
}
```
**Pattern Implementation**: Package-level variables initialized on first access via Go's `init()` semantics.

### 3. Path Resolution Strategy
```go
func getProjectRoot() (string, error) {
    // Upward traversal algorithm
    current := cwd
    for {
        gitPath := filepath.Join(current, ".git")
        if _, err := os.Stat(gitPath); err == nil {
            return current, nil
        }
        // Continue traversal...
    }
}
```
**Pattern Implementation**: Boundary detection using file system markers with fallback strategy.

### 4. Environment-Aware Configuration
```go
homeDir, _ := os.UserHomeDir()
WrapperDir = filepath.Join(homeDir, ".opencode_skill")
```
**Pattern Implementation**: User-specific path construction using environment context.

## Data & Control Flow

### Data Flow Ingress
1. **Runtime Context**: `os.Getwd()` → Current working directory detection
2. **Environment Context**: `os.UserHomeDir()` → User home directory resolution
3. **File System State**: `os.Stat()` → `.git` directory detection for project boundaries

### Data Flow Egress
1. **Path Variables**: `ProjectRoot`, `WrapperDir`, `PidFile`, `SessionMapFile` → Consumed by file system operations
2. **Timing Constants**: `PollInterval`, `ClientTimeout`, `AutoFixTimeout` → Used by timeout and interval logic
3. **Network Constants**: `OpenCodeURL`, `DaemonHost`, `DaemonPort` → Used by network operations
4. **Service Configuration**: `DefaultAgent`, `DefaultModel` → Used by API client initialization

### Control Flow
```
Package Import → init() → getProjectRoot() → Directory Detection → Path Setup
                                   ↓
                              File System Operations
```

### Initialization Sequence
1. **Static Constant Loading**: Compile-time assignment of const values
2. **Runtime Initialization**: `init()` function execution during package import
3. **Project Root Detection**: Upward traversal until `.git` marker found
4. **Directory Creation**: Automatic creation of `~/.opencode_skill` wrapper directory
5. **Path Configuration**: Setting up file paths for PID and session storage

## Integration Points

### Dependencies (External)
- **`os` package**: Environment variable access, process information
- **`path/filepath`**: Cross-platform path manipulation and joining
- **`time` package**: Duration constants for timing configurations

### Dependencies (Internal)
- **None**: This is a leaf module with no internal Go dependencies

### Consumer Modules
1. **`internal/api`**: Consumes `OpenCodeURL`, `DefaultAgent`, `DefaultModel`
2. **`internal/client`**: Uses `DaemonHost`, `DaemonPort`, `ClientTimeout`
3. **`internal/daemon`**: Accesses `DaemonHost`, `DaemonPort`, `PollInterval`, `PidFile`, `SessionMapFile`
4. **`internal/manager`**: References `AutoFixTimeout`, `PollInterval`
5. **`internal/types`**: May consume model and agent constants for type definitions

### External Integration Points
- **File System**: Creates and manages `~/.opencode_skill/` directory structure
- **Process Management**: Provides PID file location for daemon lifecycle management
- **Database Layer**: Supplies session map file path for SQLite storage
- **Network Layer**: Supplies host/port configuration for TCP communication

## Configuration Categories

### API Configuration
- **Purpose**: OpenCode AI service connectivity
- **Constants**: `OpenCodeURL`, `DefaultAgent`, `DefaultModel`
- **Consumers**: API client modules

### Daemon Configuration  
- **Purpose**: TCP server network setup
- **Constants**: `DaemonHost`, `DaemonPort`
- **Consumers**: Server and client networking modules

### Timing Configuration
- **Purpose**: Request timeout and polling intervals
- **Constants**: `PollInterval`, `ClientTimeout`, `AutoFixTimeout`
- **Consumers**: Session manager, network clients

### Path Configuration
- **Purpose**: File system resource management
- **Variables**: `ProjectRoot`, `WrapperDir`, `PidFile`, `SessionMapFile`
- **Consumers**: File system operations, process management

## Error Handling Strategy

- **Graceful Degradation**: Project root detection falls back to current directory
- **Silent Failures**: Directory creation errors ignored (non-critical)
- **Error Propagation**: Path resolution errors returned to callers
- **Defensive Programming**: Null-safe environment variable access

## Thread Safety Considerations

- **Immutable Constants**: All `const` declarations are inherently thread-safe
- **Initialization Ordering**: Package-level `init()` ensures single-threaded setup
- **Stateless Design**: No mutable state after initialization completes
- **Shared Resource Access**: Path variables are read-only after initialization

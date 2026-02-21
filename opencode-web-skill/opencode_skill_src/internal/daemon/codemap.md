# Daemon Package Codemap

## Responsibility

The `daemon` package implements a **persistent TCP server** and **session management system** for the OpenCode Web Skill application. It serves as the central coordination layer between CLI clients and OpenCode AI agents, providing:

- **TCP-based RPC server** for client communication
- **Session lifecycle management** with persistent state
- **Process lifecycle management** (PID tracking, graceful shutdown)
- **State synchronization** between local managers and remote API sessions
- **Resource recovery** on daemon restart

## Design Patterns

### Repository Pattern
- **Implementation**: `Registry` struct encapsulates all SQLite database operations
- **Purpose**: Abstracts data persistence layer behind a clean interface
- **Benefits**: Decouples business logic from storage implementation
- **Usage**: All session CRUD operations flow through the Registry

### Observer Pattern
- **Implementation**: `setupStatePersistence()` method with `OnStateChange` callback
- **Purpose**: Automatically persists session state changes to database
- **Benefits**: Loose coupling between state managers and persistence layer
- **Triggered**: SessionManager state changes → Registry updates

### Facade Pattern
- **Implementation**: `Server` struct as unified entry point
- **Purpose**: Simplifies complex subsystem interactions (network, sessions, persistence)
- **Benefits**: Single interface for all daemon operations
- **Exposed**: TCP server interface + session management

### Command Pattern
- **Implementation**: Action-based request routing in `handleConnection()`
- **Purpose**: Decouples request parsing from request execution
- **Structure**: JSON action strings → specific handler methods
- **Supported Actions**: PING, START_SESSION, INIT_SESSION, ABORT_SESSION, etc.

### Singleton Pattern (Process Level)
- **Implementation**: PID file management and port binding
- **Purpose**: Ensures only one daemon instance runs per host/port
- **Mechanism**: File locking + network port binding
- **Safety**: Cleanup of stale PID files on startup

## Data & Control Flow

### Entry Points
```
Client TCP Connection → handleConnection() → Action Router → Specific Handler
```

### Request Flow
1. **Connection Accept**: `listener.Accept()` creates new TCP connection
2. **Request Parsing**: JSON unmarshal into structured request
3. **Action Dispatch**: Switch statement routes to appropriate handler
4. **Session Lookup**: Session ID → SessionManager instance
5. **State Management**: Session operations via Manager delegation
6. **Response Generation**: JSON response marshaled and sent back

### State Persistence Flow
```
SessionManager State Change → OnStateChange Callback → Registry Update → Database Write
```

### Recovery Flow
```
Daemon Start → Registry.List() → SessionManager Recreation → State Restoration → Active Sessions
```

### Key Data Structures
- **`Server`**: Main orchestrator with sessions map, listener, registry
- **`SessionData`**: SQLite entity with project-scoped session metadata
- **`Registry`**: Thread-safe SQLite repository with mutex protection
- **Request/Response**: JSON-encoded action payloads

## Integration Points

### Dependencies
- **`internal/manager`**: Session lifecycle and state management
  - `SessionManager` instances per active session
  - State persistence callbacks
  - Request processing pipeline

- **`internal/api`**: OpenCode API client integration
  - Session creation/abortion
  - Prompt/command submission
  - Remote session state queries

- **`internal/config`**: Configuration constants
  - Daemon port/host settings
  - File paths (PID file, database)
  - Project root directory

- **`internal/types`**: Shared data structures
  - `PromptRequest`, `CommandRequest`, `AnswerRequest`
  - Standardized request/response formats

### Consumer Modules
- **CLI Client**: Primary consumer via TCP communication
  - Session initialization and management
  - Status queries and state monitoring
  - Prompt/command submission

- **OpenCode API**: Integration point for remote services
  - Session creation and lifecycle
  - Agent communication (Sisyphus, Prometheus, Atlas)
  - Question/answer workflows

### External Systems
- **SQLite Database**: Persistent storage for session metadata
- **File System**: PID file management for process coordination
- **TCP Network**: Client communication protocol

### Critical Synchronization Points
- **Session State**: Registry mutex ensures thread-safe operations
- **Process Lifecycle**: Signal handling for graceful shutdown
- **State Recovery**: Session reconstruction on daemon restart
- **Agent Locking**: Session-level agent state synchronization

## Architecture Notes

### Concurrency Model
- **Goroutine-per-connection**: Each client connection handled in separate goroutine
- **Mutex Protection**: Registry operations protected by sync.Mutex
- **Session Isolation**: Each SessionManager operates independently

### Fault Tolerance
- **Stale PID Cleanup**: Automatic removal of dead process files
- **Session Recovery**: Automatic restoration of sessions on restart
- **Error Handling**: Comprehensive error propagation and logging
- **Graceful Shutdown**: Signal handling with proper resource cleanup

### Performance Considerations
- **Connection Pooling**: Reuse TCP connections where possible
- **State Caching**: Session managers maintain local state for performance
- **Batch Operations**: Database operations optimized for minimal I/O
- **Memory Management**: Limited session lifecycles prevent memory leaks
# CodeMap: opencode_skill Main Entry Point

## Responsibility

**Role**: Command-Line Interface (CLI) Gateway and Daemon Lifecycle Manager

**Primary Responsibilities**:
- **Dual-Mode Execution Controller**: Manages bifurcation between daemon server mode and client interaction mode
- **Daemon Process Lifecycle Management**: Handles start, stop, and restart operations of the background TCP daemon
- **Session Orchestration**: Manages OpenCode AI agent session creation, lookup, and state validation
- **Message Pipeline Coordinator**: Routes user inputs through appropriate handlers (prompts, commands, answers)
- **CLI Argument Parser**: Processes command-line flags and positional arguments with strict Go flag package semantics
- **Error State Management**: Provides user-friendly error messages and session discovery guidance

**Domain-Specific Functions**:
- Session initialization with project-scoped identifiers
- Asynchronous prompt/command submission with polling synchronization
- Agent response waiting mechanisms
- Multi-modal message handling (text prompts, structured commands, Q&A responses)

## Design Patterns

**1. Command Pattern**
```go
switch command {
case "start":
    startDaemon()
case "stop":
    stopDaemon()
case "restart":
    restartDaemon()
case "init-session":
    // Session initialization logic
default:
    // Message/command routing
}
```
*Applies the Command Pattern to handle different CLI operations with distinct execution strategies.*

**2. Factory Method Pattern**
```go
c := client.NewClient("") // For session lookup
c = client.NewClientWithMeta(sessionData.ID, project, sessionName) // For operations
```
*Uses factory methods to create client instances with different initialization contexts.*

**3. Strategy Pattern**
```go
if *sync {
    c.WaitForResult()
} else {
    fmt.Printf("Command sent: %v\n", res["message"])
}
```
*Implements synchronous vs asynchronous execution strategies based on user preference.*

**4. State Machine Pattern**
```go
if *isDaemon {
    // Daemon execution path
} else {
    // Client execution path
}
```
*Manages two distinct execution states with separate control flows.*

**5. Gateway Pattern**
```go
c.SendRequest("START_SESSION", map[string]string{"working_dir": sessionData.WorkingDir})
```
*Provides unified communication interface to the daemon via TCP protocol.*

## Data & Control Flow

**Input Data Flow**:
```
Command Line Arguments
    ↓
Flag Parsing (Go flag package)
    ↓
Mode Determination (--daemon flag)
    ↓
┌─────────────────────────────────────────────┐
│              Execution Path                  │
├─────────────────────────────────────────────┤
│ Daemon Mode:                                │
│   → Registry Creation                        │
│   → TCP Server Initialization                │
│   → Session Management Loop                  │
│                                            │
│ Client Mode:                               │
│   → Command Type Identification             │
│   → Session Lookup/Validation                │
│   → Daemon Connection Establishment          │
│   → Request/Response Handling                │
└─────────────────────────────────────────────┘
    ↓
TCP Communication Layer (JSON encoding)
    ↓
Response Processing & Output Formatting
```

**Data Transformations**:
1. **Argument Parsing**: Raw CLI strings → structured flag values and command arguments
2. **Session Resolution**: Project/session names → SessionData struct with working directory
3. **Message Formatting**: User text → types.PromptRequest/CommandRequest/AnswerRequest
4. **Response Handling**: JSON responses → formatted console output

**Control Flow Patterns**:
- **Early Return Pattern**: Used extensively for error handling and command-specific exits
- **Flag-Precedence Rules**: Go flag package requires flags before positional arguments
- **Session Validation**: Always verifies session existence before operations
- **Asynchronous Synchronization**: /wait command enables blocking for completion

## Integration Points

**Internal Dependencies**:
- `internal/client`: TCP client interface for daemon communication
- `internal/config`: Configuration constants (ports, file paths, defaults)
- `internal/daemon`: Daemon server implementation and registry management
- `internal/types`: Request/response type definitions

**External Dependencies**:
- `flag`: Command-line argument parsing with strict Go semantics
- `os/exec`: Process management for daemon lifecycle control
- `os`: File system operations and process information
- `syscall`: Low-level process signaling (SIGKILL)
- `path/filepath`: Path resolution for working directories
- `strings`: String manipulation for message processing
- `time`: Timing utilities for process management

**System Integration Points**:
- **Port Management**: Uses `lsof` to detect and manage daemon process on config.DaemonPort
- **PID File Management**: Creates/removes PID file at config.PidFile for process tracking
- **Working Directory**: Resolves absolute paths for session working directories
- **Process Lifecycle**: Integrates with operating system process management

**Consumer Modules**:
- **End Users**: Direct CLI interface for session management and AI agent interaction
- **Daemon Process**: TCP client for session management and prompt/command submission
- **Session Registry**: Persistent storage integration for session state
- **OpenCode API**: Gateway to external AI services via internal API client

**Interface Contracts**:
- **TCP Protocol**: JSON-encoded request/response messages with status field validation
- **Session Identification**: Project + session_name tuple for unique session identification
- **Error Handling**: Consistent error reporting with user-friendly messages and recovery suggestions
- **Output Formatting**: Structured console output with status indicators and submission guidance
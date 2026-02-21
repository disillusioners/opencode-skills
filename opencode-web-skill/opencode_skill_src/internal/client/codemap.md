# Client Package Codemap

## Responsibility

The `client` package implements a **Daemon TCP Client** that serves as the primary communication layer between the OpenCode CLI application and the background daemon process. This package encapsulates all TCP socket communication, request/response handling, daemon lifecycle management, and session state monitoring functionality.

### Core Responsibilities:
- **TCP Connection Management**: Establish and maintain persistent connections to the daemon server
- **Request/Response Serialization**: Handle JSON encoding/decoding of protocol messages
- **Daemon Lifecycle Management**: Auto-start daemon if not running, handle connection failures
- **Session Operations**: Init, list, get, and abort sessions through the daemon
- **Asynchronous Operation Monitoring**: Poll daemon status and handle long-running operations
- **User Interface Integration**: Format and display daemon responses, questions, and status information

## Design Patterns

### 1. **Client-Server Pattern**
- The client implements a classic client-server communication model where the CLI client sends structured requests to the daemon server
- Uses JSON-encoded messages over TCP sockets for protocol communication
- Implements request-response semantics with status codes and error handling

### 2. **Factory Method Pattern**
- `NewClient()` and `NewClientWithMeta()` provide alternative constructors for creating client instances
- Allows for different client configurations while maintaining a consistent interface

### 3. **State Machine Interaction Pattern**
- `WaitForResult()` implements polling-based interaction with the daemon's state machine
- Transitions through states defined in `internal/manager` package (IDLE, BUSY, etc.)
- Handles asynchronous operations with timeout and retry logic

### 4. **Command Pattern**
- `SendRequest()` method acts as a command dispatcher that accepts action strings and payloads
- Supports multiple daemon actions: INIT_SESSION, LIST_SESSIONS, GET_STATUS, ABORT_SESSION, etc.
- Encapsulates command execution and response processing

### 5. **Facade Pattern**
- Provides a simplified interface to complex daemon communication functionality
- Hides TCP connection details, JSON serialization, and error handling complexity from CLI consumers

## Data & Control Flow

### Entry Points:
1. **CLI Command Invocation** → Client instantiation via factory methods
2. **Daemon Connection** → `Connect()` method establishes TCP connection
3. **Request Dispatch** → `SendRequest()` method sends structured JSON messages
4. **Response Processing** → JSON decoding and status validation
5. **Session Management** → Session CRUD operations through daemon
6. **Status Monitoring** → Polling mechanism for async operations

### Data Flow Architecture:
```
CLI Command → Client Instance → TCP Connection → JSON Request → Daemon
    ↑                                      ↓
Response Processing ← JSON Response ← Daemon Processing
    ↓
User Display/Session State Updates
```

### Control Flow Patterns:
- **Synchronous Operations**: Direct request-response (status checks, session listing)
- **Asynchronous Operations**: Polling with timeout (long-running agent sessions)
- **Error Recovery**: Automatic daemon restart on connection failure
- **State Synchronization**: Continuous polling to maintain local state consistency

### Critical Data Structures:
- `Client`: Main client struct holding session metadata and connection state
- `SessionData`: Session information container (project, name, ID, working directory)
- JSON Request/Response Maps: Dynamic message structures for daemon communication

## Integration Points

### Internal Dependencies:
- **`internal/config`**: 
  - Provides daemon host/port configuration (`config.DaemonHost`, `config.DaemonPort`)
  - Defines timeout and polling intervals (`config.ClientTimeout`, `config.PollInterval`)
  - Specifies project root for daemon spawning (`config.ProjectRoot`)

- **`internal/manager`**:
  - Imports state constants (`manager.StateIdle`, `manager.StateBusy`)
  - Uses manager-defined state machine logic for operation monitoring

### External Integration Points:
- **Daemon Server**: Primary integration point - TCP communication on configured port
- **CLI Application**: Main consumer - instantiates clients and processes responses
- **Session Registry**: Indirect integration - manages persistent session state
- **User Interface**: Terminal output formatting for status, questions, and results

### Protocol Integration:
- **OpenCode Agent Protocol**: Communicates with agent sessions via daemon proxy
- **Question/Answer Workflow**: Handles interactive agent prompts and user responses
- **Session Lifecycle Management**: Integrates with session creation, execution, and cleanup

### Key Integration Methods:
- `SendRequest()`: Primary communication interface to daemon
- `WaitForResult()`: Integration point for long-running operations
- `Status()`: Status monitoring and display integration
- `printQuestions()`: User interaction integration for agent prompts

### Error Handling Integration:
- Connection failure handling with automatic daemon restart
- JSON serialization/deserialization error handling
- Protocol error response processing and user notification
- Timeout handling for long-running operations

This package serves as the critical bridge between the CLI user interface and the daemon-managed OpenCode agent operations, providing robust, fault-tolerant communication while maintaining a clean abstraction layer.
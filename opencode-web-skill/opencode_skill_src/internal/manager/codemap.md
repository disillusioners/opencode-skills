# Manager Package Codemap

## Responsibility

The `manager` package implements a **session state machine orchestrator** responsible for managing per-session lifecycle and state transitions during OpenCode AI agent interactions. It serves as the core control layer coordinating request processing, state persistence, and asynchronous worker management.

## Design Patterns

### 1. State Machine Pattern
- **Implementation**: Explicit state enumeration with three distinct states
  - `StateIdle`: Session available for new requests
  - `StateBusy`: Session processing prompt/command requests
  - `StateWaitingForInput`: Session awaiting user responses to agent questions
- **State Transitions**: Controlled through mutex-protected state variables with deterministic transition logic

### 2. Worker Pattern
- **Implementation**: Asynchronous goroutine-based task execution (`runWorker`)
- **Features**: 
  - Non-blocking request processing
  - Result communication via `workerDoneChan` channel
  - Worker lifecycle tracking with `isWorkerBusy` flag

### 3. Observer Pattern
- **Implementation**: `OnStateChange` callback function for state persistence
- **Mechanism**: State changes trigger automatic serialization via `SaveState()` method
- **Purpose**: Decouples state management from persistence concerns

### 4. Command Pattern
- **Implementation**: `Request` struct as command abstraction
- **Supported Commands**:
  - `PROMPT`: Submit prompts to OpenCode agents
  - `COMMAND`: Send commands to OpenCode agents  
  - `ANSWER`: Respond to agent questions
  - `FIX`: Auto-recovery mechanism for timeout scenarios

### 5. Producer-Consumer Pattern
- **Implementation**: Buffered `inputChan` for request queuing
- **Benefits**: Decouples request submission from processing, provides backpressure control

## Data & Control Flow

### Data Entry Points
1. **External Request Submission** (`SubmitRequest()`)
   - Accepts `Request` structs from daemon layer
   - Pre-validates and sets optimistic state lock
   - Routes to `inputChan` for processing

2. **Worker Result Channel** (`workerDoneChan`)
   - Receives async completion notifications
   - Contains result/error from OpenCode API calls

3. **Timer Events** (`ticker.C`)
   - Periodic state polling and timeout checking
   - Drives auto-fix and question polling mechanisms

### Internal Processing Flow
1. **Main Event Loop** (`loop()`)
   - Multiplexes input from multiple channels
   - Delegates to specialized handlers (`handleRequest`, `handleWorkerDone`)
   - Drives periodic maintenance operations

2. **Request Processing Pipeline**
   ```
   SubmitRequest → inputChan → handleRequest → runWorker → workerDoneChan → handleWorkerDone
   ```

3. **State Synchronization** (`sync.RWMutex`)
   - Protects critical state variables during transitions
   - Allows concurrent read access for snapshot operations

### Data Exit Points
1. **State Change Callbacks** (`OnStateChange`)
   - Persists state changes to external registry
   - Provides real-time state updates to daemon layer

2. **Snapshot API** (`GetSnapshot()`)
   - Returns current state for external monitoring
   - Used by daemon for state persistence and reporting

## Integration Points

### Dependencies (Upstream)
1. **`internal/api.Client`**
   - Primary interface to OpenCode AI services
   - Handles prompt/command execution and question polling
   - Provides session-specific API communication

2. **`internal/types` Package**
   - Defines request/response structures (`PromptRequest`, `CommandRequest`, `AnswerRequest`)
   - Provides type-safe message passing between components

3. **`internal/config` Package**
   - Configuration constants (`PollInterval`, `AutoFixTimeout`)
   - Tunable parameters for behavior customization

### Consumer Modules (Downstream)
1. **`internal/daemon` Package**
   - Manages multiple `SessionManager` instances
   - Initiates sessions via `NewSessionManager()`
   - Receives state change notifications via `OnStateChange` callback

2. **TCP Client Layer**
   - Submits requests through `SubmitRequest()` method
   - Queries session state via `GetSnapshot()` method

### External Interactions
1. **OpenCode AI API**
   - Primary external dependency for agent communication
   - Two-way interaction: prompts/commands → responses, questions → answers

2. **State Persistence System**
   - SQLite database through daemon registry
   - Automatic state serialization via `PersistedState` struct

3. **User Interaction**
   - Indirect through daemon CLI layer
   - Question-answer cycle coordination
   - Abort mechanism for user interruption

## Key Implementation Details

### Concurrency Control
- **Read-Write Mutex**: Protects state variables during transitions
- **Channel-Based Communication**: Ensures thread-safe producer-consumer patterns
- **Goroutine Coordination**: Worker lifecycle management with result signaling

### Error Handling
- **Worker Error Propagation**: Errors returned via `workerDoneChan`
- **Request Validation**: Type assertions and payload validation
- **Auto-Recovery**: Timeout-based auto-fix mechanism

### Performance Considerations
- **Optimistic Locking**: State pre-setting before request processing
- **Channel Buffering**: Prevents blocking during high request loads
- **Efficient Polling**: Configurable intervals reduce unnecessary API calls

### State Persistence Strategy
- **Delta Serialization**: Only saves changed state components
- **JSON Marshaling**: Structured state representation for storage
- **Atomic Updates**: State changes are atomic and consistent
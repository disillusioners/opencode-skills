# opencode-web-skill/opencode_skill_src/internal/api/

## Responsibility

The **API package** serves as an HTTP client adapter that encapsulates all communication with the OpenCode AI service. It implements a **facade pattern** to provide a clean, abstracted interface for higher-level system components to interact with the OpenCode API without exposing HTTP implementation details. This package handles session lifecycle management, prompt/command transmission, question retrieval, and response processing through a RESTful interface.

## Design Patterns

1. **Client Pattern**: The `Client` struct encapsulates HTTP client configuration and provides a unified interface for all OpenCode API operations.

2. **Facade Pattern**: Simplifies complex API interactions by providing high-level methods (`CreateSession`, `SendPrompt`, etc.) that abstract underlying HTTP request/response handling.

3. **Factory Pattern**: The `NewClient()` function centralizes client instantiation with consistent configuration (10-minute timeout, custom User-Agent).

4. **Template Method Pattern**: The `doRequest()` method implements a reusable template for HTTP operations, with `postAndParse()` specializing for POST requests with JSON response parsing.

5. **Strategy Pattern**: Response parsing in `GetQuestions()` implements fallback strategies to handle different API response formats (direct array vs. wrapped data structure).

6. **Data Transfer Object Pattern**: Type-safe structures (`PromptRequest`, `SessionResponse`, etc.) ensure proper serialization/deserialization of API payloads.

## Data & Control Flow

```
┌─────────────┐      ┌──────────────┐      ┌─────────────────┐
│  Consumer   │      │   API Client │      │  OpenCode API   │
│  (Manager/  │─────▶│  (facade)    │─────▶│  (external)     │
│  Daemon)    │      │              │      │                 │
└─────────────┘      └──────────────┘      └─────────────────┘
       │                    │                      │
       │                    │                      │
       ▼                    ▼                      ▼
┌─────────────┐      ┌──────────────┐      ┌─────────────────┐
│ Session Mgt │      │ HTTP Request │      │ JSON Response    │
│ Prompt Send │◀─────│ JSON Payload │◀─────│ Processing       │
│ Q&A Flow    │      │ Headers      │      │ Error Handling   │
└─────────────┘      └──────────────┘      └─────────────────┘
```

**Data Flow Process**:
1. **Entry Point**: Consumer invokes high-level methods on `Client` struct
2. **Request Preparation**: Data marshaled to JSON with appropriate headers (`Content-Type`, `Accept`, `User-Agent`, `x-opencode-directory`)
3. **HTTP Execution**: Configured `http.Client` with 10-minute timeout executes requests
4. **Response Handling**: HTTP status validation, body reading, and JSON unmarshaling
5. **Error Propagation**: Structured error handling with HTTP status code incorporation
6. **Return**: Parsed data returned to consumer or error propagated upward

**Key Control Flow**:
- **Session Creation**: `CreateSession()` → POST /session → SessionResponse.ID
- **Prompt Transmission**: `SendPrompt()` → POST /session/{id}/message → Interface{} result
- **Question Retrieval**: `GetQuestions()` → GET /question → []Question (with fallback parsing)
- **Answer Submission**: `AnswerQuestion()` → POST /question/{id}/reply → error handling
- **Session Termination**: `AbortSession()` → POST /session/{id}/abort → cleanup

## Integration Points

### Dependencies
- **External**: `net/http` (HTTP client), `encoding/json` (serialization), `fmt` (formatting)
- **Internal Configuration**: `opencode_skill/internal/config` → `OpenCodeURL` (base endpoint configuration)
- **Internal Types**: `opencode_skill/internal/types` → shared request/response structures (`PromptRequest`, `CommandRequest`, `AnswerRequest`)

### Consumer Modules
- **Daemon Layer**: `internal/daemon/` → uses API client for session initialization and prompt routing
- **Manager Layer**: `internal/manager/` → consumes API client for request handling and response processing
- **CLI Layer**: `internal/client/` → indirect consumption through daemon for user-initiated operations

### External Integration
- **OpenCode API**: Primary external dependency, RESTful endpoint at `config.OpenCodeURL`
- **HTTP Protocol**: Standard JSON-over-HTTP communication with custom headers
- **Authentication**: Implicit through API endpoint configuration (no explicit auth handling in this layer)

### Key Integration Characteristics
- **Stateless Design**: Each API call is independent, no client-side state management
- **Context Propagation**: Working directory passed via `x-opencode-directory` header
- **Error Contract**: Structured errors with HTTP status codes for consistent error handling
- **Response Flexibility**: Support for multiple API response formats (direct vs. wrapped data)
- **Configuration Binding**: Runtime configuration through `config.OpenCodeURL` for environment flexibility
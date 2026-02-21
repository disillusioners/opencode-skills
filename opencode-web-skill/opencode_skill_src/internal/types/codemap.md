# opencode-web-skill/opencode_skill_src/internal/types/

## Responsibility

This package serves as a **Data Transfer Object (DTO) registry** for the OpenCode web skill system. It implements a **shared type contract** between the CLI client and daemon server components, providing a **schema contract** for inter-process communication (IPC) and external API integration. The package encapsulates all **request/response payload structures** used across the system, ensuring type safety and consistent data representation across component boundaries.

## Design Patterns

### Core Patterns
- **DTO (Data Transfer Object) Pattern**: All types are designed for serialization/deserialization to JSON, enabling seamless cross-component communication
- **Composite Pattern**: Complex types like `PromptRequest` and `CommandRequest` aggregate multiple related fields into cohesive logical units
- **Builder Pattern Anticipation**: Struct design facilitates potential future builder implementations for complex object construction

### Structural Design
- **Type Safety**: Strong typing with explicit field declarations and JSON tags
- **Immutable Interface**: Read-only access pattern prevents external mutation of internal state
- **Nested Composition**: Hierarchical struct composition (`ModelDetails` within requests, `Part` arrays for content)

### Serialization Strategy
- **JSON-First Design**: All types include `json:"field"` tags for standardized serialization
- **Field Renaming**: JSON field names use snake_case while Go fields use PascalCase, implementing the **Field Mapping Pattern**
- **Optional vs Required**: Explicit field typing distinguishes required from optional data

## Data & Control Flow

### Inbound Flow
1. **External Input**: Data enters the system as JSON from:
   - CLI client commands
   - Network requests to daemon
   - OpenCode API responses

2. **Deserialization**: JSON payload → Go struct unmarshaling
   - `json.Unmarshal()` converts incoming JSON to typed structs
   - Type validation occurs during unmarshaling
   - Field mapping from JSON keys to struct fields

3. **Processing**: Typed structs are passed to:
   - Command handlers
   - Session managers
   - API clients

### Outbound Flow
1. **Serialization**: Go struct → JSON marshaling
   - `json.Marshal()` converts typed structs to JSON
   - Field mapping from struct fields to JSON keys
   - Consistent formatting for network transmission

2. **Output Channels**: JSON data flows to:
   - Network responses to CLI
   - API requests to OpenCode services
   - File persistence (session storage)

### State Management
- **Stateless Operation**: Package maintains no internal state
- **Immutable Objects**: No mutable state within type definitions
- **Pure Data Contract**: Functions as a passive data schema provider

## Integration Points

### Internal Dependencies
- **CLI Component (`internal/`)**: Constructs `PromptRequest`, `CommandRequest`, and `AnswerRequest` for user interactions
- **Daemon Component (`internal/daemon/`)**: Parses incoming requests, validates, and routes to appropriate handlers
- **Manager Component (`internal/manager/`)**: Uses these types to coordinate session state and API communication
- **API Client (`internal/api/`)**: Consumes request types and produces response types for OpenCode service integration

### External Integration
- **TCP Network Layer**: JSON-serialized types are transmitted via TCP sockets between CLI and daemon
- **SQLite Persistence**: Session data containing these types is stored in SQLite database via JSON serialization
- **OpenCode API Integration**: Request types are mapped to API payloads, response types are used for processing

### Interface Contracts
- **Serialization Interface**: All types implement the implicit `json.Marshaler`/`json.Unmarshaler` contract
- **Type Compatibility**: Shared types ensure CLI and daemon can communicate without version mismatches
- **Schema Validation**: Type definitions enforce required fields and data consistency

### Extension Points
- **New Request Types**: Struct composition pattern allows easy addition of new request/response types
- **Field Evolution**: JSON tags allow for field renaming without breaking external compatibility
- **Versioning Strategy**: Package structure supports future versioning through namespace or type aliasing

## Type Hierarchy

### Request Types
- `PromptRequest`: Core AI prompt interaction with agent/model specification and content parts
- `CommandRequest`: Extended prompt with additional command/arguments for action-based requests
- `AnswerRequest`: Response containing multiple answer arrays for interactive sessions

### Supporting Types
- `ModelDetails`: Provider and model identification for AI routing
- `Part`: Typed content segments for flexible request composition

This package establishes a **robust data contract** that ensures type-safe communication across all system components while maintaining flexibility for future extensions and API evolution.
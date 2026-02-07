## OpenCode Web API Cheat Sheet

### Base Configuration
```
Server URL: http://127.0.0.1:4096 (default)
API Docs: /doc endpoint
Health Check: GET /global/health
```

### Authentication (if configured)
```
Username: opencode (default, or OPENCODE_SERVER_USERNAME)
Password: OPENCODE_SERVER_PASSWORD
Method: HTTP Basic Auth
```

### Core Endpoints

**Sessions**
```
GET    /session                List all sessions
POST   /session                Create new session
GET    /session/:id            Get session details
DELETE /session/:id            Delete session
PATCH  /session/:id            Update session title
GET    /session/status         Get status of all sessions
GET    /session/:id/todo       Get session todo list
POST   /session/:id/abort      Abort running session
GET    /session/:id/diff       Get session diff
```

**Messages**
```
GET    /session/:id/message            List messages
POST   /session/:id/message            Send message (sync)
POST   /session/:id/prompt_async       Send message (async)
POST   /session/:id/command            Execute slash command (e.g., /start-work)
GET    /session/:id/message/:messageID  Get specific message
```

**Agents & Models (oh-my-opencode)**
```
GET /agent                   List available agents (Prometheus, Sisyphus, Hephaestus, etc.)
GET /config/models           List available models
```

**Projects & Files**
```
GET /project                 List all projects
GET /project/current         Get current project
GET /file?path=              List directory
GET /file/content?path=      Read file
GET /find?pattern=           Search text in files
GET /find/file?query=        Find files by name
```

### Request/Response Patterns

**Message Body Format**
```json
{
  "messageID": "msg_...",
  "agent": "Prometheus|Sisyphus|Hephaestus",
  "model": "provider:model",
  "noReply": false,
  "system": "optional system prompt",
  "tools": ["tool1", "tool2"],
  "parts": [
    {
      "role": "user|system|assistant",
      "content": {
        "type": "text",
        "text": "Your message here"
      }
    }
  ]
}
```

**Session Creation Body**
```json
{
  "parentID": "parent_session_id",
  "title": "Project Name"
}
```

**Session Update Body**
```json
{
  "title": "New Title"
}
```

**Slash Command Body**
```json
{
  "messageID": "msg_...",
  "agent": "Prometheus|Sisyphus|Hephaestus",
  "model": "provider:model",
  "command": "/start-work|/sessions|/agents",
  "arguments": {}
}
```

**Critical: /start-work command**
```json
POST /session/:id/command
Body: {
  "command": "/start-work",
  "agent": "Prometheus"
}
```
- Required after Prometheus creates/updates a plan
- Triggers handoff to Sisyphus for implementation
- Context preserved automatically

### Model ID Format
```
providerID:modelID
Example: openai:gpt-4, anthropic:claude-3-opus
```

### Agent Selection (oh-my-opencode)
- **Prometheus**: Planning, analysis, architecture (requires /start-work to execute)
- **Sisyphus**: Implementation, coding, general purpose (default agent)
- **Hephaestus**: Automation tasks, workflows
- **Atlas & others**: USER-ONLY - do not use for automation
- **default**: Sisyphus (first agent from GET /agent, or omit agent field)
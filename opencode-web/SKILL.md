---
name: opencode-web
description: "Control and operate oh-my-opencode via web API interface. This skill works with the oh-my-opencode plugin agents: Sisyphus (default), Hephaestus (automation), and Prometheus (planning). Use this skill to manage sessions, select models, switch agents, and coordinate codixwng through OpenCode's HTTP server."
metadata: {"version": "1.0.0", "author": "Kha Nguyen", "license": "MIT", "github_url": "https://github.com/disillusioners/opencode-web-skill"}
---


# OpenCode Web Controller (oh-my-opencode)

## Core rule

This skill controls OpenCode's oh-my-opencode plugin through its web API (HTTP interface), not through the TUI or CLI. All communication happens via REST API calls to the OpenCode server.

## Oh-my-opencode Agents

This skill uses the following agents from the oh-my-opencode plugin:

| Agent | Purpose | Usage |
|-------|---------|-------|
| **Sisyphus** | General purpose coding and analysis | Default agent - use when agent not specified |
| **Hephaestus** | Automation tasks and workflows | Use for automation-specific tasks |
| **Prometheus** | Planning and architecture | Use for creating detailed plans |

**Note**: Atlas and other agents in oh-my-opencode are designed for user interaction only. Do NOT use them for automated agent control.

## Pre-flight

- Verify OpenCode server is running: `opencode serve` (default: http://127.0.0.1:4096)
- Check authentication requirements (if `OPENCODE_SERVER_PASSWORD` is set)
- Confirm the server base URL with the user if different from default

## Server connection

- Default server URL: `http://127.0.0.1:4096`
- API documentation available at: `/doc` (e.g., http://127.0.0.1:4096/doc)
- Authentication (if configured):
  - Username: default `opencode`, override with `OPENCODE_SERVER_USERNAME`
  - Password: from `OPENCODE_SERVER_PASSWORD` environment variable
- Always verify server health first: `GET /global/health`

## Project management

### Get current project
```
GET /project/current
Response: Project (id, name, path, etc.)
```

### List projects
```
GET /project
Response: Project[]
```
- Verify you are working on the correct project before creating sessions.

## Session management

### List sessions
```
GET /session
Response: Session[] (sessionID, title, createdAt, etc.)
```

### Create new session
```
POST /session
Body: { parentID?, title? }
Response: Session
```

### Get session details
```
GET /session/:id
Response: Session (full details)
```

### Session selection
- List all sessions to find existing ones for the project
- Check if a session for the current project already exists:
  - Compare session title or project path
- If existing session found: reuse it (preserves context)
- If no existing session: create a new one (ask user first)
- Never create a new session without user approval unless explicitly requested

## Agent (mode) control

### List available agents
```
GET /agent
Response: Agent[] (id, name, description, etc.)
```

### Oh-my-opencode agents (for automation use only)

**Sisyphus** (default agent)
- General purpose coding and analysis
- Use when user doesn't specify a specific agent
- Handles most coding tasks
- Can switch between planning and execution as needed

**Hephaestus** (automation)
- Specialized for automation tasks
- Use for workflow automation
- Handles repetitive or scripted tasks

**Prometheus** (planning)
- Dedicated planning and architecture agent
- Creates detailed step-by-step plans
- **Requires special workflow**: Send `/start-work` command after plan approval

### Forbidden agents

**Atlas** and other agents are USER-ONLY:
- Designed for human interaction via TUI
- Do NOT use these in automated workflows
- They will not respond properly to programmatic control

### Agent selection
- Default agent: **Sisyphus** when user doesn't specify
- Specify agent via `agent` field in message body
- You can switch agents between messages without changing session

## Model selection

### List models
```
GET /config/providers
Response: { providers: Provider[], default: { ... } }
```
Extract models from the `providers` array. Each provider has a `models` object.
Construct model IDs as `provider_id/model_id`.

### Model selection workflow
- Ask user which AI model to use
- Select model by its ID (`provider_id/model_id`) in message requests
- If user doesn't specify: use the default model from the `default` field in the response

## Message handling

### Send message (synchronous)
```
POST /session/:id/message
Body: {
  messageID?,    // optional: reply to specific message
  agent?,        // optional: agent ID (default if not specified)
  model?,        // optional: provider_id/model_id format
  noReply?,      // optional: true to just add message without response
  system?,       // optional: system prompt override
  tools?,        // optional: tool restrictions
  parts: [       // required: message parts
    {
      role: "user" | "system" | "assistant",
      content: { type: "text", text: "your message" }
    }
  ]
}
Response: { info: Message, parts: Part[] }
```

### Send message (asynchronous)
```
POST /session/:id/prompt_async
Body: same as /session/:id/message
Response: 204 No Content (no waiting)
```

### Execute slash command
```
POST /session/:id/command
Body: {
  messageID?,   // optional: context message ID
  agent?,       // optional: agent ID
  model?,       // optional: model ID
  command: string,      // required: slash command
  arguments: any        // optional: command arguments
}
Response: { info: Message, parts: Part[] }
```

### Get messages
```
GET /session/:id/message?limit=N
Response: { info: Message, parts: Part[] }[]
```

## Prometheus agent behavior (Planning)

- Select or create a session for the project
- Send a planning request using the Prometheus agent:
  ```json
  {
    "agent": "Prometheus",
    "model": "provider_id/model_id", // optional
    "parts": [{
      "role": "user",
      "content": { "type": "text", "text": "Analyze the task and propose a step-by-step plan. Ask clarification questions if needed." }
    }]
  }
  ```
- Wait for the response (synchronous call)
- Review the plan carefully
- If the plan is incorrect or incomplete:
  - Send a follow-up message asking for revision
- **CRITICAL**: When plan is approved and ready for execution:
  - Send the `/start-work` slash command to begin implementation:
    ```json
    {
      "command": "/start-work",
      "agent": "Prometheus"
    }
    ```
- This triggers Prometheus to hand off the plan to Sisyphus for execution

## Implementation workflow (Sisyphus)

### After Prometheus plan approval

1. **Send /start-work command**:
   ```json
   POST /session/:id/command
   Body: {
     "command": "/start-work",
     "agent": "Prometheus"
   }
   ```
   This hands off execution to Sisyphus automatically.

2. **Monitor Sisyphus execution**:
   ```json
   GET /session/:id/message?limit=20
   ```
   Watch for Sisyphus messages implementing the plan.

3. **Handle questions from Sisyphus**:
   If Sisyphus asks clarification questions:
   - Respond directly in the same message thread
   - Sisyphus can handle both planning and execution context
   - No need to switch agents - Sisyphus is capable

### Direct Sisyphus usage

For straightforward tasks without explicit planning:
- Use Sisyphus directly (default agent)
- Sisyphus will plan and execute as needed
- Agent ID can be omitted to use session default (Sisyphus)

## Completion and looping

### Prometheus → Sisyphus workflow

1. **Planning phase** (Prometheus agent)
   - Use Prometheus for complex tasks requiring detailed planning
   - Review and revise plan as needed
   - Get user approval on plan

2. **Trigger execution** (/start-work)
   - Send `/start-work` command when plan is approved
   - Prometheus hands off to Sisyphus for implementation

3. **Execution phase** (Sisyphus agent)
   - Sisyphus implements the plan
   - Monitor progress via messages endpoint
   - Answer any questions Sisyphus has directly

4. **Review and iterate**
   - Check completed work against requirements
   - If issues arise: use Prometheus to replan, then /start-work again
   - Or let Sisyphus handle minor adjustments

### Direct Sisyphus workflow

- For simple tasks, skip explicit Prometheus planning
- Let Sisyphus handle everything (it can plan as needed)
- Monitor progress and provide feedback directly

### Tracking progress
- Use `GET /session/:id/message` to review conversation
- Use `GET /session/:id/todo` to check todo list
- Use `GET /session/status` to see current activity
- Use `GET /session/:id/diff` to review file changes

## Error handling

### Common errors and responses

| Error | Action |
|-------|--------|
| Server not reachable | Ask user to start `opencode serve` |
| Session not found | List sessions, create new if needed |
| Agent not found | List agents, correct the ID |
| Model not found | List providers/models, correct the ID |
| Timeout on message | Use `prompt_async` for long-running tasks |

### Session management errors

- If a session is stuck: use `POST /session/:id/abort` to abort
- If a message fails: check response for error details
- Always verify session status before sending new messages

## Utility endpoints



### Get files and directories
```
GET /file?path=/relative/path
Response: FileNode[] (directory listing)
```

### Read file content
```
GET /file/content?path=/relative/path
Response: FileContent
```

### Get todo list
```
GET /session/:id/todo
Response: Todo[]
```

### Get session diff
```
GET /session/:id/diff?messageID=
Response: FileDiff[]
```

## Best practices

1. **Always verify server health** before making API calls
2. **Reuse sessions** for the same project to preserve context
3. **Use synchronous messages** for critical operations (plan review, confirmation)
4. **Use asynchronous messages** for long-running implementations
5. **Switch agents** between messages as needed (don't assume same agent for entire session)
6. **Check default models** before forcing a specific one
7. **Review the API docs** at `/doc` for the most up-to-date endpoint information
8. **Use error responses** to guide user to the right action
9. **Document the API calls** clearly in your responses for transparency

## Example workflow (Prometheus → Sisyphus)

1. **Verify server**: `GET http://127.0.0.1:4096/global/health`
2. **List sessions**: `GET /session`
3. **Find/create session** for project
4. **Get models**: `GET /config/providers`
5. **Start planning** with Prometheus:
   ```json
   {
     "agent": "Prometheus",
     "model": "provider_id/model_id", // optional: uses default or previously selected model
     "parts": [{
       "role": "user",
       "content": { "type": "text", "text": "Plan the implementation of [feature]" }
     }]
   }
   ```
6. **Review and revise** plan with Prometheus as needed
7. **Trigger execution** with `/start-work` command:
   ```json
   {
     "command": "/start-work",
     "agent": "Prometheus"
   }
   ```
8. **Monitor Sisyphus** implementing the plan via `GET /session/:id/message`
9. **Answer any questions** Sisyphus has directly
10. **Complete task** and verify results

## Example workflow (Direct Sisyphus)

1. **Verify server**: `GET http://127.0.0.1:4096/global/health`
2. **List sessions**: `GET /session`
3. **Find/create session** for project
4. **Send task** to Sisyphus (default agent, no need to specify):
   ```json
   {
     "model": "provider_id/model_id", // optional
     "parts": [{
       "role": "user",
       "content": { "type": "text", "text": "Add [feature] to the application" }
     }]
   }
   ```
5. **Monitor progress** via messages endpoint
6. **Provide feedback** directly to Sisyphus as needed
7. **Complete task** and verify results

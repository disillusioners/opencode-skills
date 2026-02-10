---
name: opencode-web
description: "Control and operate oh-my-opencode via web API interface. This skill works with the oh-my-opencode plugin agents: Sisyphus (default) and Prometheus (planning). Use this skill to manage sessions, select models, switch agents, and coordinate coding through OpenCode's HTTP server."
metadata: {"version": "1.0.0", "author": "Kha Nguyen", "license": "MIT", "github_url": "https://github.com/disillusioners/opencode-web-skill"}
---


# OpenCode Web Controller (with oh-my-opencode plugin)

## Core rule

This skill controls OpenCode's oh-my-opencode plugin through its web API (HTTP interface), not through the TUI or CLI. All communication happens via REST API calls to the OpenCode server.

## Oh-my-opencode Agents

This skill uses the following agents from the oh-my-opencode plugin:

| Agent | Purpose | Usage |
|-------|---------|-------|
| **Sisyphus** | General purpose coding and analysis | Default agent - use when agent not specified |
| **Prometheus** | Planning and architecture | Use for creating detailed plans |

## Pre-flight

- Endpoint: `http://127.0.0.1:4096` (default)
- Use health check to verify OpenCode server is running: `GET /global/health`
- If not running, ask user to run `opencode serve`

## API Headers

Most API requests (especially those starting/managing sessions) require the following headers:
- `Content-Type: application/json`
- `Accept: application/json`
- `x-opencode-directory: <absolute_path_to_project_directory>`
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

If you currently have a session, use it. Otherwise, create a new session.

### Create new session
```
POST /session
Headers:
  Content-Type: application/json
  x-opencode-directory: /absolute/path/to/project
Body: { parentID?, title? }
Response: Session
```
- **CRITICAL**: You MUST remember the session `id` from the response to use in all subsequent `/session/:id/...` calls.


### Get session details
```
GET /session/:id
Response: Session (full details)
```

## Agent (mode) control

### Oh-my-opencode agents

**Sisyphus** (default agent)
- General purpose coding and analysis
- Use when user doesn't specify a specific agent
- Handles most coding tasks
- Can switch between planning and execution as needed

**Prometheus** (planning)
- Dedicated planning and architecture agent
- Creates detailed step-by-step plans
- **Requires special workflow**: Send `start-work` command after plan approval

### Agent selection
- Default agent: **Sisyphus** when user doesn't specify
- Specify agent via `agent` field in message body
- You can switch agents between messages without changing session

## Provider-Model selection

### List models
```
GET /config/providers
Response: { providers: Provider[], default: { ... } }
```
Extract models from the `providers` array. Each provider has a `models` object.
Construct model identifiers as an object `{ providerID, modelID }`.

### Model selection workflow
- Ask user which AI model to use
- Use the model identifier in `providerID/modelID` string format (e.g., `zai-coding-plan/glm-4.7`)
- If user doesn't specify: use the default model from the `default` field in the response

## Message handling

### Send message (synchronous)
```
POST /session/:id/message
Headers:
  Content-Type: application/json
  x-opencode-directory: /absolute/path/to/project
Body: {
  agent,        // required: sisyphus/prometheus
  model: "zai-coding-plan/glm-4.7", // required: model string
  parts: [       // required: message parts
    {
      type: "text", 
      text: "your message"
    }
  ]
}
Response: { info: Message, parts: Part[] }
```

IMPORTANT: API is synchronous, so you need to wait for the response. Some response may be very long (10 minutes), so you need to wait for the response. Send message or command one by one, do not send multiple messages at once. Must wait for the response before sending the next message or command.

### Execute slash command
Command also synchronous, it extend base on message.

```
POST /session/:id/command
Headers:
  Content-Type: application/json
  x-opencode-directory: /absolute/path/to/project
Body: {
  agent,       // required: sisyphus/prometheus
  model: "zai-coding-plan/glm-4.7", // required: model string
  command: string,      // required: slash command (e.g., "start-work")
  arguments: string,    // optional: command arguments
  parts: []             // required: usually empty for commands
}
Response: { info: Message, parts: Part[] }
```

### Get messages
```
GET /session/:id/message?limit=N
Response: { info: Message, parts: Part[] }[]
```

## Planning Workflow: Plan (with prometheus) -> Implement (with atlas)

- Select or create a session for the project
- Send a planning request using the Prometheus agent:
  ```json
  {
    "agent": "prometheus",
    "model": "zai-coding-plan/glm-4.7",
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
  - Send the `start-work` command to begin implementation:
    ```json
    {
      "command": "start-work",
      "agent": "atlas",
      "model": "zai-coding-plan/glm-4.7",
      "arguments": "",
      "parts": []
    }
    ```
- This triggers Prometheus to hand off the plan to Sisyphus for execution

## Direct Q&A, simple tasks workflow (Sisyphus)

For straightforward tasks or Q&A:
- Use Sisyphus directly (default agent)
- 

## Completion and looping

### Prometheus → Sisyphus workflow

1. **Planning phase** (Prometheus agent)
   - Use Prometheus for complex tasks requiring detailed planning
   - Review and revise plan as needed
   - Get user approval on plan

2. **Trigger execution** (start-work)
   - Send `start-work` command when plan is approved
   - Prometheus hands off to Sisyphus for implementation

3. **Execution phase** (Sisyphus agent)
   - Sisyphus implements the plan
   - Monitor progress via messages endpoint
   - Answer any questions Sisyphus has directly

4. **Review and iterate**
   - Check completed work against requirements
   - If issues arise: use Prometheus to replan, then `start-work` again
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
2. **List sessions**: `GET /session?directory=/path/to/project`
3. **Find/create session** for project (remember the session `id`)
4. **Get models**: `GET /config/providers`
5. **Start planning** with Prometheus:
   ```json
   {
     "agent": "Prometheus",
     "model": "zai-coding-plan/glm-4.7",
     "parts": [{
       "role": "user",
       "content": { "type": "text", "text": "Plan the implementation of [feature]" }
     }]
   }
   ```
6. **Review and revise** plan with Prometheus as needed
7. **Trigger execution** with `start-work` command:
   ```json
   {
     "command": "start-work",
     "agent": "sisyphus",
     "model": "zai-coding-plan/glm-4.7",
     "arguments": "",
     "parts": []
   }
   ```
8. **Monitor Sisyphus** implementing the plan via `GET /session/:id/message`
9. **Answer any questions** Sisyphus has directly
10. **Complete task** and verify results

## Example workflow (Direct Sisyphus)

1. **Verify server**: `GET http://127.0.0.1:4096/global/health`
2. **List sessions**: `GET /session?directory=/path/to/project`
3. **Find/create session** for project (remember the session `id`)
4. **Send task** to Sisyphus (default agent, no need to specify):
   ```json
   {
     "model": "zai-coding-plan/glm-4.7",
     "parts": [{
       "role": "user",
       "content": { "type": "text", "text": "Add [feature] to the application" }
     }]
   }
   ```
5. **Monitor progress** via messages endpoint
6. **Provide feedback** directly to Sisyphus as needed
7. **Complete task** and verify results

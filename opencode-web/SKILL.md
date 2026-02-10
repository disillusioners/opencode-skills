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
| **Sisyphus** | General purpose coding and analysis | Default agent - for quick tasks and Q&A |
| **Prometheus** | Planning and architecture | Use for creating detailed plans |
| **Atlas** | Implementation | Use for implementing plans |

## Pre-flight

- Endpoint: `http://127.0.0.1:4096` (default)
- Use health check to verify OpenCode server is running: `GET /global/health`
- If not running, ask user to run `opencode serve`

## AI model to use
This skill uses only one AI model: `zai-coding-plan/glm-4.7`

## How to call API
API can be called via curl or any client that you support.
API responses may be very long text, you need to handle long text intelligently. One method is to write the response to a file and read it with the tail command.

## API to use

### API Headers

Most API requests (especially those starting/managing sessions) require the following headers:
- `Content-Type: application/json`
- `Accept: application/json`
- `x-opencode-directory: <absolute_path_to_project_directory>`

### Session management

If you currently have a session, use it. Otherwise, create a new session.

#### Create new session
```
POST /session
Headers:
  Content-Type: application/json
  x-opencode-directory: /absolute/path/to/project
Body: { "parentID": "optional", "title": "optional" }
Response: Session
```
- **CRITICAL**: You MUST remember the session `id` from the response to use in all subsequent `/session/:id/...` calls.

#### Get session details
```
GET /session/:id
Response: Session (full details)
```

### Message handling

#### Send message (synchronous)
```
POST /session/:id/message
Headers:
  Content-Type: application/json
  x-opencode-directory: /absolute/path/to/project
Body: {
  "agent": "sisyphus",
  "model": "zai-coding-plan/glm-4.7",
  "parts": [
    {
      "type": "text", 
      "text": "your message"
    }
  ]
}
Response: { "info": "Message", "parts": [] }
```

IMPORTANT: API is synchronous, so you need to wait for the response. Some responses may be very long (10 minutes), so you need to wait for the response. Send message or command one by one, do not send multiple messages at once. Must wait for the response before sending the next message or command.

#### Execute slash command
Command also synchronous, it extend base on message.

```
POST /session/:id/command
Headers:
  Content-Type: application/json
  x-opencode-directory: /absolute/path/to/project
Body: {
  "agent": "sisyphus",
  "model": "zai-coding-plan/glm-4.7",
  "command": "start-work",
  "arguments": "",
  "parts": []
}
Response: { "info": "Message", "parts": [] }
```

#### Get messages
```
GET /session/:id/message?limit=N
Response: { info: Message, parts: Part[] }[]
```

## Workflows

### Planning Workflow: Plan (with prometheus) -> Implement (with atlas)

- Select or create a session for the project
- Send a planning request using the Prometheus agent:
  ```json
  {
    "agent": "prometheus",
    "model": "zai-coding-plan/glm-4.7",
    "parts": [{
      "role": "user",
      "content": { "type": "text", "text": "YOUR TASK DESCRIPTION HERE FOR PROMETHEUS TO PLAN" }
    }]
  }
  ```
- Wait for the response (synchronous call)
- If the response message includes questions:
1. Decide if it is simple. If not, ask user and wait for user response
2. Send message to answer it
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
- This triggers Prometheus to hand off the plan to Atlas for execution

### Direct Q&A, simple tasks workflow (Sisyphus)

For straightforward tasks or Q&A:
- Use Sisyphus directly (default agent)
- If the response message includes questions:
1. Decide if it is simple, if not, ask user
2. Send message to answer it

## The flow
Create new session > send message or command > wait for response > repeat until task is done

### Tracking progress
- Use `GET /session/:id/message` to review conversation
- Use `GET /session/:id/todo` to check todo list
- Use `GET /session/status` to see current activity
- Use `GET /session/:id/diff` to review file changes

API can use when needed. Normally the response of message will include the information you need.

## Error handling

### Common errors and responses

| Error | Action |
|-------|--------|
| Server not reachable | Ask user to start `opencode serve` |
| Session not found | List sessions, create new if needed |
| Timeout on message or command | Abort the session then send message 'continue' |

### Session management errors

- If a session is stuck: use `POST /session/:id/abort` to abort
- After abort you need send new message to the session: 'continue'


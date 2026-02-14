---
name: opencode-web
description: "Control and operate oh-my-opencode via web API interface using the Go-based opencode_skill."
metadata: {"version": "2.0.0", "author": "Kha Nguyen", "license": "MIT", "github_url": "https://github.com/disillusioners/opencode-web-skill"}
---

# OpenCode Web Controller (Go)

This skill controls OpenCode's agents (Sisyphus, Prometheus, Atlas) via the web API using a robust **Daemon-Client architecture** implemented in Go.

## Prerequisites

1.  **Binary**: Use `opencode_skill` for all interactions. Ensure it is in your PATH (e.g., `~/bin/opencode_skill`).
2.  **Session Initialization**: You **MUST** initialize a session with a target working directory before sending commands. The session remembers this directory, so you do not need to be in the project root when running subsequent commands.

## Usage

### 1. Initialize a Session
**Syntax:**
```bash
opencode_skill init-session <PROJECT> <SESSION_NAME> <WORKING_DIR>
```
- `<PROJECT>`: Project identifier (e.g., `myapp`, `website`, `api`).
- `<SESSION_NAME>`: Task or feature name (e.g., `planning`, `task-1`, `bugfix`).
- `<WORKING_DIR>`: Absolute path to the project root directory where the agent should work.

The full session name will be created as `PROJECT:SESSION_NAME` (e.g., `myapp:task-1`). This reduces naming conflicts and helps organize sessions by project.

**Example:**
```bash
opencode_skill init-session myapp feature-login /Users/me/projects/my-app
# Creates session: myapp:feature-login
```

**Re-initializing a Session:**
If you run `init-session` with the same PROJECT and SESSION_NAME, the old OpenCode session will be automatically aborted and a new one created with updated settings. No confirmation is required (designed for agent use).

### 2. Send Commands
**Syntax:**
```bash
opencode_skill <PROJECT> <SESSION_NAME> <MESSAGE> [options]
```

- `<PROJECT>`: The project identifier used when initializing the session.
- `<SESSION_NAME>`: The session name used when initializing the session.
- `<MESSAGE>`: Text to send, or a command starting with `/`.
- `[options]`:
    - `-agent <NAME>`: Switch agent (Default: `sisyphus`, Options: `prometheus`, `atlas`).
    - `--help`: Show all available options.

### Patience on timeout
Commands will timeout after **10 minutes** on the client side, but the **Daemon keeps working**.

If you see a timeout message:
```text
[TIMEOUT] Message is taking longer than 10 minutes.
Daemon is still running in background.
Run: `opencode_skill <PROJECT> <SESSION_NAME> /wait` to check again.
```
High complexity tasks may take longer than 10 minutes to complete. Use `/wait` to check the status of the daemon. (The `/wait` command also have 10 minutes timeout and run synchronously)
When you using other terminal/console tool to call the wrapper script, please modify the timeout param of those tool call to more than 10 minutes to wait correctly.

**To Reconnect:**
```bash
opencode_skill <PROJECT> <SESSION_NAME> /wait
```

### Available Commands

**Basic Flow:**
```bash
# Send a message or prompt
opencode_skill <PROJECT> <SESSION_NAME> "Your request here"

# Check status (non-blocking)
opencode_skill <PROJECT> <SESSION_NAME> /status

# Wait for result (blocking, up to 10 min)
opencode_skill <PROJECT> <SESSION_NAME> /wait
```
### Interactive Questions
If the agent asks a question (e.g., requires clarification), the wrapper will prompt you:
```text
[?] Request ID: ...
    Which linter should I use?
    Options available.
```

**CRITICAL INSTRUCTION**: When a question is received:
1.  **Suggest** the best answer to the user based on context.
2.  **Ask** the user for confirmation.
3.  **DO NOT** automatically execute the `/answer` command unless the user explicitly tells you to "auto-answer" or "decide for me".

**To Answer:**
```bash
# Answer with text or option label
opencode_skill <PROJECT> <SESSION_NAME> /answer "ESLint"

# If multiple questions are asked:
opencode_skill <PROJECT> <SESSION_NAME> /answer "ESLint" "Jest"
```

## Workflows
> **Reminder**: Ensure you have initialized the session using `init-session` before running these commands.

**Simple Workflow (For simple tasks)**
1.  **Initialize**: `opencode_skill init-session myapp feature-A /path/to/project`
2.  **Request**: `opencode_skill "myapp:feature-A" "Your request here" -agent sisyphus`
3.  **Wait**: `opencode_skill "myapp:feature-A" /wait` (when needed)
4.  **Answer**: `opencode_skill "myapp:feature-A" /answer "Option 1" "Option 2"` (for multiple questions)


**Plan & Execute (For high complexity tasks that require planning)**
1.  **Initialize**: `opencode_skill init-session myapp feature-A /path/to/project`
2.  **Plan**: `opencode_skill "myapp:feature-A" "Make a plan..." -agent prometheus`
3.  **Refine**: `opencode_skill "myapp:feature-A" "Feedback..." -agent prometheus`
4.  **Implement**: `opencode_skill "myapp:feature-A" "/start-work" -agent atlas`
5.  **Wait (if long)**: `opencode_skill "myapp:feature-A" /wait`
6.  **Answer**: `opencode_skill "myapp:feature-A" /answer "Option 1" "Option 2"` (for multiple questions)

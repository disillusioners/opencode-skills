---
name: opencode-web
description: "Control and operate oh-my-opencode via web API interface using the Go-based opencode_skill."
metadata: {"version": "2.0.0", "author": "Kha Nguyen", "license": "MIT", "github_url": "https://github.com/disillusioners/opencode-web-skill"}
---

# OpenCode Web Controller (Go)

This skill controls OpenCode's agents (Sisyphus, Prometheus, Atlas) via the web API using a robust **Daemon-Client architecture** implemented in Go.

## Prerequisites

1.  **Binary**: Use `opencode_skill` for all interactions. Ensure it is in your PATH (e.g., `~/bin/opencode_skill`).
2.  **Working Directory**: You **MUST** change your current working directory to the project root before running the command. The tool detects the project root from the CWD.

## Usage

**Syntax:**
```bash
cd <PROJECT_ROOT>
opencode_skill <SESSION_NAME> <MESSAGE> [options]
```

- `<SESSION_NAME>`: Unique name for your session (e.g., `planning`, `task-1`). **If a session with this name does not exist, a new one is automatically created.**
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
Run: `opencode_skill <session> /wait` to check again.
```
High complexity tasks may take longer than 10 minutes to complete. Use `/wait` to check the status of the daemon. (The `/wait` command also have 10 minutes timeout and run synchronously)
When you using other terminal/console tool to call the wrapper script, please modify the timeout param of those tool call to more than 10 minutes to wait correctly.

**To Reconnect:**
```bash
opencode_skill <SESSION_NAME> /wait
```

### Available Commands

**Basic Flow:**
```bash
# Send a message or prompt
opencode_skill <SESSION_NAME> "Your request here"

# Check status (non-blocking)
opencode_skill <SESSION_NAME> /status

# Wait for result (blocking, up to 10 min)
opencode_skill <SESSION_NAME> /wait
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
opencode_skill <SESSION_NAME> /answer "ESLint"

# If multiple questions are asked:
opencode_skill <SESSION_NAME> /answer "ESLint" "Jest"
```

## Workflows
> **Reminder**: Ensure you are in the project root directory before running these commands (`cd /path/to/project`).

**Simple Workflow (For simple tasks)**
1.  **Request**: `opencode_skill ... "feature-A" "Your request here" -agent sisyphus`
2.  **Wait**: `opencode_skill ... "feature-A" /wait` (when needed)
3.  **Answer**: `opencode_skill ... "feature-A" /answer "Option 1" "Option 2"` (for multiple questions)


**Plan & Execute (For high complexity tasks that require planning)**
1.  **Plan**: `opencode_skill ... "feature-A" "Make a plan..." -agent prometheus`
2.  **Refine**: `opencode_skill ... "feature-A" "Feedback..." -agent prometheus`
3.  **Implement**: `opencode_skill ... "feature-A" "/start-work" -agent atlas`
4.  **Wait (if long)**: `opencode_skill ... "feature-A" /wait`
5.  **Answer**: `opencode_skill ... "feature-A" /answer "Option 1" "Option 2"` (for multiple questions)

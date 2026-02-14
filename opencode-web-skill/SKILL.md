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

### Non-Blocking Message Submission
All message submissions (PROMPT, COMMAND, ANSWER) return **immediately** with a confirmation:

```text
[SUBMITTED] Run: opencode_skill <PROJECT> <SESSION_NAME> /wait
```

The daemon continues processing in the background. Use `/wait` to retrieve results when ready.

### MUST Retrieving Results with `/wait`
The `/wait` command is the primary way to get results from the daemon:
- **Blocking**: Waits up to 10 minutes for the daemon to complete its work
- **Non-blocking alternative**: Use `/status` to check if results are ready without waiting

**To check for results:**
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

**Simple Workflow (For simple tasks without planning)**
1.  **Initialize**: `opencode_skill init-session myapp feature-A /path/to/project`
2.  **Request**: `opencode_skill myapp feature-A "Your request here" -agent sisyphus`
3.  **Answer**: `opencode_skill myapp feature-A /answer "Option 1" "Option 2"` 
4.  **Wait until completion**: `opencode_skill myapp feature-A /wait`

**Plan & Execute (For high complexity tasks that require planning)**
1.  **Initialize**: `opencode_skill init-session myapp feature-A /path/to/project`
2.  **Plan**: `opencode_skill myapp feature-A "Make a plan..." -agent prometheus`
3.  **Answer multiple questions**: `opencode_skill myapp feature-A /answer "Option 1" "Option 2"`
4. **Answer a special question/choice: Deep review or Start work**: This answer based on your decision, normally high complexity tasks require deep review, low complexity tasks prefer start work.
5.  **When response message have explicitly guide to run command /start-work**: `opencode_skill myapp feature-A "/start-work" -agent atlas` (Note:from this point always use atlas agent on this session)
6.  **Wait until completion**: `opencode_skill myapp feature-A /wait`
7. **Ask for clean up finished plan and boulder.json**: `opencode_skill myapp feature-A "Clean up finished plan and boulder.json"` -agent atlas

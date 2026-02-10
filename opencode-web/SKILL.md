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
- Use health check to verify OpenCode server is running: `python3 opencode-web/opencode_wrapper.py --check-health` (or `curl -s http://127.0.0.1:4096/global/health`)
- If not running, ask user to run `opencode serve`

## AI model to use
This skill uses only one AI model: `zai-coding-plan/glm-4.7`

## How to use

We provide a Python wrapper script `opencode-web/opencode_wrapper.py` to simplify interaction with the OpenCode API. 
Use this script instead of raw `curl` requests.

**Syntax:**
```bash
python3 opencode-web/opencode_wrapper.py <SESSION_NAME> <MESSAGE> [options]
```

- `<SESSION_NAME>`: A unique name for your session (e.g., `planning`, `task-1`). The script will automatically map this to the correct Session ID. If the name is new, a new session is created.
- `<MESSAGE>`: The text to send, or a command starting with `/`.

### 1. Send Message (Sisyphus/General)

```bash
python3 opencode-web/opencode_wrapper.py "task-1" "Your message here"
```

### 2. Send Message with Specific Agent

```bash
python3 opencode-web/opencode_wrapper.py "planning-1" "Plan the implementation of X" --agent prometheus
```

### 3. Send Slash Command

```bash
python3 opencode-web/opencode_wrapper.py "task-1" "/start-work arguments" --agent atlas
```

### 4. View Conversation History

To view the recent message history (tail) of a session, use the `/log` command:

```bash
python3 opencode-web/opencode_wrapper.py <SESSION_NAME> "/log [N]"
```

- `N`: Number of recent messages to show (default: 10).

Example:
```bash
python3 opencode-web/opencode_wrapper.py "task-1" "/log 5"
```

## Workflows

### Planning Workflow: Plan (with prometheus) -> Implement (with atlas)

1.  **Plan**: Ask Prometheus to create a plan.
    ```bash
    python3 opencode-web/opencode_wrapper.py "feature-plan" "Create a plan for [task description]" --agent prometheus
    ```
    Wait for the response. If Prometheus asks questions, reply using the *same session name*:
    ```bash
    python3 opencode-web/opencode_wrapper.py "feature-plan" "Answer to question" --agent prometheus
    ```

2.  **Execute**: When the plan is approved, start work with Atlas (using the *same session name*).
    ```bash
    python3 opencode-web/opencode_wrapper.py "feature-plan" "/start-work" --agent atlas
    ```

### Direct Q&A (Sisyphus)

For simple questions or coding tasks:

```bash
python3 opencode-web/opencode_wrapper.py "quick-fix" "How do I fix this bug?"
```

## Error Handling

| Error | Action |
|-------|--------|
| Server not reachable | Ask user to start `opencode serve` |
| Script error | Check if `python3` is installed and `opencode_wrapper.py` is executable |



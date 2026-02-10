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
Use this script instead of raw `curl` requests. The script handles session management automatically (persisting session ID in `.opencode_session`).

### 1. Send Message (Sisyphus/General)

To send a message to the default agent (Sisyphus):

```bash
python3 opencode-web/opencode_wrapper.py "Your message here"
```

### 2. Send Message with Specific Agent

To use a different agent (e.g., `prometheus` for planning):

```bash
python3 opencode-web/opencode_wrapper.py "Plan the implementation of X" --agent prometheus
```

### 3. Send Slash Command

To send a command (e.g., `start-work`), start the message with `/`:

```bash
python3 opencode-web/opencode_wrapper.py "/start-work arguments" --agent atlas
```

### 4. Start New Session

To force a new session (e.g., for a new unrelated task):

```bash
python3 opencode-web/opencode_wrapper.py "New task started" --reset
```

## Workflows

### Planning Workflow: Plan (with prometheus) -> Implement (with atlas)

1.  **Plan**: Ask Prometheus to create a plan.
    ```bash
    python3 opencode-web/opencode_wrapper.py "Create a plan for [task description]" --agent prometheus
    ```
    Wait for the response. If Prometheus asks questions, reply using the standard message command:
    ```bash
    python3 opencode-web/opencode_wrapper.py "Answer to question" --agent prometheus
    ```

2.  **Execute**: When the plan is approved, start work with Atlas.
    ```bash
    python3 opencode-web/opencode_wrapper.py "/start-work" --agent atlas
    ```

### Direct Q&A (Sisyphus)

For simple questions or coding tasks, just use the default agent:

```bash
python3 opencode-web/opencode_wrapper.py "How do I fix this bug?"
```

## Error Handling

| Error | Action |
|-------|--------|
| Server not reachable | Ask user to start `opencode serve` |
| Script error | Check if `python3` is installed and `opencode_wrapper.py` is executable |



---
name: opencode-web
description: "Control and operate oh-my-opencode via web API interface using the Python wrapper script."
metadata: {"version": "1.0.1", "author": "Kha Nguyen", "license": "MIT", "github_url": "https://github.com/disillusioners/opencode-web-skill"}
---

# OpenCode Web Controller

This skill controls OpenCode's agents (Sisyphus, Prometheus, Atlas) via the web API using a Python wrapper.

## Prerequisites

1.  **Server Must Be Running**: Ensure `opencode serve` is running on `http://127.0.0.1:4096`.
2.  **Wrapper Script**: Use `opencode-web/opencode_wrapper.py` for all interactions.

## Usage

**Syntax:**
```bash
python3 opencode-web/opencode_wrapper.py <SESSION_NAME> <MESSAGE> [options]
```

> [!IMPORTANT]
> **Timeout Rule**: Every command or message sent to OpenCode may take time to process. You **MUST wait at least 5 minutes** for a response before considering the server stuck or the request failed. Only after this timeout should you attempt to use `/fix` or retry.

- `<SESSION_NAME>`: Unique name for your session (e.g., `planning`, `task-1`). **If a session with this name does not exist, a new one is automatically created.**
- `<MESSAGE>`: Text to send, or a command starting with `/`.
- `[options]`:
    - `--agent <NAME>`: Switch agent (Default: `sisyphus`, Options: `prometheus`, `atlas`).
    - `--help`: Show all available options (avoid using this unless necessary).

### Examples

**1. General Coding (Sisyphus)**
```bash
python3 opencode-web/opencode_wrapper.py "task-1" "Refactor utils.py"
```

**2. Planning (Prometheus)**
```bash
python3 opencode-web/opencode_wrapper.py "plan-1" "Create a plan for auth" --agent prometheus
```

**3. Execution (Atlas) - Start Work**
```bash
python3 opencode-web/opencode_wrapper.py "plan-1" "/start-work" --agent atlas
```

**4. View History**
View the last N messages of a session:
```bash
python3 opencode-web/opencode_wrapper.py "task-1" "/log 5"
```

**5. Fix Stuck Sessions**
If no response for > 5 minutes, abort and resume:
```bash
python3 opencode-web/opencode_wrapper.py "task-1" "/fix"
```

## Workflows

**Plan & Execute**
1.  **Plan**: `python3 ... "feature-A" "Make a plan..." --agent prometheus`
2.  **Refine**: `python3 ... "feature-A" "Feedback..." --agent prometheus`
3.  **Implement**: `python3 ... "feature-A" "/start-work" --agent atlas`

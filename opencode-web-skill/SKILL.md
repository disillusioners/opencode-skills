---
name: opencode-web
description: "Control and operate oh-my-opencode via web API interface using the Python wrapper script."
metadata: {"version": "2.0.0", "author": "Kha Nguyen", "license": "MIT", "github_url": "https://github.com/disillusioners/opencode-web-skill"}
---

# OpenCode Web Controller

This skill controls OpenCode's agents (Sisyphus, Prometheus, Atlas) via the web API using a robust **Daemon-Client architecture**.

## Prerequisites

1.  **Server Must Be Running**: Ensure `opencode serve` is running on `http://127.0.0.1:4096`.
2.  **Wrapper Script**: Use `~/opencode-web/opencode_wrapper.py` for all interactions.
3.  **Working Directory**: You **MUST** change your current working directory to the project root before running the wrapper script. The script detects the project root from the CWD.

## Daemon Architecture

This version uses a background daemon to manage sessions.
-   **Auto-Start**: The daemon starts automatically when you run the wrapper.
-   **Persistence**: Sessions stay alive even if the client disconnects.
-   **Interactive**: Supports agent questions and answers.

## Usage

**Syntax:**
```bash
cd <PROJECT_ROOT>
python3 ~/opencode-web/opencode_wrapper.py <SESSION_NAME> <MESSAGE> [options]
```

- `<SESSION_NAME>`: Unique name for your session (e.g., `planning`, `task-1`). **If a session with this name does not exist, a new one is automatically created.**
- `<MESSAGE>`: Text to send, or a command starting with `/`.
- `[options]`:
    - `--agent <NAME>`: Switch agent (Default: `sisyphus`, Options: `prometheus`, `atlas`).
    - `--help`: Show all available options.

### Timeout & Reconnection
Commands will timeout after **5 minutes** on the client side, but the **Daemon keeps working**.

If you see a timeout message:
```text
[TIMEOUT] Message is taking longer than 5 minutes.
Daemon is still running in background.
Run: `python -m opencode_wrapper <session> /wait` to check again.
```

**To Reconnect:**
```bash
python3 ~/opencode-web/opencode_wrapper.py <SESSION_NAME> /wait
```

### Interactive Questions
If the agent asks a question (e.g., requires clarification), the wrapper will prompt you:
```text
[?] Request ID: ...
    Which linter should I use?
    Options available.
```

**To Answer:**
```bash
# Answer with text or option label
python3 ~/opencode-web/opencode_wrapper.py <SESSION_NAME> /answer "ESLint"
```

## Workflows
> **Reminder**: Ensure you are in the project root directory before running these commands (`cd /path/to/project`).

**Plan & Execute**
1.  **Plan**: `python3 ... "feature-A" "Make a plan..." --agent prometheus`
2.  **Refine**: `python3 ... "feature-A" "Feedback..." --agent prometheus`
3.  **Implement**: `python3 ... "feature-A" "/start-work" --agent atlas`
4.  **Wait (if long)**: `python3 ... "feature-A" /wait`

**Fixing Stuck Sessions**
If the daemon is stuck, you can restart it:
```bash
pkill -f opencode_wrapper
```
Then run your command again to restart the daemon.

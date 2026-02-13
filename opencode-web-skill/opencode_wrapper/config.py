import os
import sys
from pathlib import Path

# OpenCode Configuration
OPENCODE_URL = "http://127.0.0.1:4096"
DEFAULT_AGENT = "sisyphus"
DEFAULT_MODEL = "zai-coding-plan/glm-5"

# Daemon Configuration
DAEMON_HOST = "127.0.0.1"
DAEMON_PORT = 44111

# Directory Paths
def get_project_root():
    """Finds the git root or uses current directory."""
    current = Path.cwd()
    while current != current.parent:
        if (current / ".git").exists():
            return current
        current = current.parent
    return Path.cwd()

PROJECT_ROOT = get_project_root()
WRAPPER_DIR = Path.home() / ".opencode_wrapper"
WRAPPER_DIR.mkdir(exist_ok=True)

PID_FILE = WRAPPER_DIR / "daemon.pid"
SESSION_MAP_FILE = WRAPPER_DIR / "sessions.json"

# Timing
POLL_INTERVAL = 2.0
CLIENT_TIMEOUT = 300  # 5 minutes
AUTO_FIX_TIMEOUT = 600 # 10 minutes
SHOW_LOGS = False

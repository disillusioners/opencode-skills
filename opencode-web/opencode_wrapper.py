#!/usr/bin/env python3
import os
import sys
import json
import argparse
import urllib.request
import urllib.error
from pathlib import Path

# Configuration
OPENCODE_URL = "http://127.0.0.1:4096"
SESSION_FILE = Path(".opencode_session")
DEFAULT_AGENT = "sisyphus"
DEFAULT_MODEL = "zai-coding-plan/glm-4.7"

def get_project_root():
    """Finds the git root or uses current directory."""
    current = Path.cwd()
    while current != current.parent:
        if (current / ".git").exists():
            return current
        current = current.parent
    return Path.cwd()

PROJECT_ROOT = get_project_root()

def load_session():
    if SESSION_FILE.exists():
        try:
            return SESSION_FILE.read_text().strip()
        except Exception:
            return None
    return None

def save_session(session_id):
    SESSION_FILE.write_text(session_id)

def api_request(method, endpoint, data=None):
    url = f"{OPENCODE_URL}{endpoint}"
    headers = {
        "Content-Type": "application/json",
        "Accept": "application/json",
        "x-opencode-directory": str(PROJECT_ROOT)
    }
    
    if data:
        body = json.dumps(data).encode('utf-8')
    else:
        body = None

    req = urllib.request.Request(url, data=body, headers=headers, method=method)
    
    try:
        with urllib.request.urlopen(req) as response:
            return json.loads(response.read().decode('utf-8'))
    except urllib.error.URLError as e:
        print(f"Error connecting to OpenCode: {e}")
        if hasattr(e, 'read'):
             print(e.read().decode('utf-8'))
        sys.exit(1)

def check_health():
    try:
        api_request("GET", "/global/health")
        return True
    except:
        return False

def create_session():
    print("Creating new session...")
    data = {"title": "OpenClaw Wrapper Session"}
    response = api_request("POST", "/session", data)
    session_id = response.get("id")
    if session_id:
        save_session(session_id)
        return session_id
    else:
        print("Failed to create session.")
        sys.exit(1)

def parse_model_string(model_str):
    if "/" in model_str:
        provider_id, model_id = model_str.split("/", 1)
        return {"providerID": provider_id, "modelID": model_id}
    # Fallback if no slash, though likely correct format is required
    return {"providerID": "zai-coding-plan", "modelID": model_str}

def send_message(session_id, message, agent, model):
    data = {
        "agent": agent,
        "model": parse_model_string(model),
        "parts": [{"type": "text", "text": message}]
    }
    print(f"Sending message to session {session_id}...")
    response = api_request("POST", f"/session/{session_id}/message", data)
    return response

def send_command(session_id, command_str, agent, model):
    # Parses command like "/start-work arg1 arg2"
    parts = command_str.strip().split()
    cmd = parts[0][1:] # remove leading /
    args = " ".join(parts[1:])
    
    data = {
        "agent": agent,
        "model": model, # API expects string here, unlike message endpoint
        "command": cmd,
        "arguments": args,
        "parts": []
    }
    print(f"Sending command '{cmd}' to session {session_id}...")
    response = api_request("POST", f"/session/{session_id}/command", data)
    return response

def main():
    parser = argparse.ArgumentParser(description="OpenCode Wrapper for OpenClaw")
    parser.add_argument("message", help="Message to send or command starting with /")
    parser.add_argument("--agent", default=DEFAULT_AGENT, help=f"Agent to use (default: {DEFAULT_AGENT})")
    parser.add_argument("--model", default=DEFAULT_MODEL, help=f"Model to use (default: {DEFAULT_MODEL})")
    parser.add_argument("--reset", action="store_true", help="Force create a new session")
    parser.add_argument("--check-health", action="store_true", help="Check server health and exit")
    
    args = parser.parse_args()
    
    if args.check_health:
        if check_health():
            print("OpenCode server is running and healthy.")
            sys.exit(0)
        else:
            print("OpenCode server is NOT reachable.")
            sys.exit(1)
            
    if not check_health():
        print("OpenCode server is not running at http://127.0.0.1:4096")
        print("Please run 'opencode serve' first.")
        sys.exit(1)

    session_id = load_session()
    if args.reset or not session_id:
        session_id = create_session()

    # Check if message is a command
    if args.message.startswith("/"):
        response = send_command(session_id, args.message, args.agent, args.model)
    else:
        response = send_message(session_id, args.message, args.agent, args.model)

    # Output likely contains parts with text
    if response and "parts" in response:
        for part in response["parts"]:
            if part.get("type") == "text":
                print(part.get("text"))
            elif "content" in part and part["content"].get("type") == "text":
                 print(part["content"].get("text"))
    else:
        print(json.dumps(response, indent=2))

if __name__ == "__main__":
    main()

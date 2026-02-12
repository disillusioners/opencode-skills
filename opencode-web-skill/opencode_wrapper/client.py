import time
import socket
import json
import os
import sys
import subprocess
import argparse
import requests
from pathlib import Path
from .config import DAEMON_HOST, DAEMON_PORT, PID_FILE, CLIENT_TIMEOUT, PROJECT_ROOT, OPENCODE_URL, SESSION_MAP_FILE

def parse_model_string(model_str):
    if "/" in model_str:
        provider_id, model_id = model_str.split("/", 1)
        return {"providerID": provider_id, "modelID": model_id}
    return {"providerID": "zai-coding-plan", "modelID": model_str}

def resolve_session(name):
    sessions = {}
    if SESSION_MAP_FILE.exists():
        try:
            sessions = json.loads(SESSION_MAP_FILE.read_text().strip())
        except Exception:
            pass
            
    if name in sessions:
        return sessions[name]

    # Create new session
    try:
        url = f"{OPENCODE_URL}/session"
        headers = {"x-opencode-directory": str(PROJECT_ROOT)}
        resp = requests.post(url, json={"title": name}, headers=headers)
        resp.raise_for_status()
        session_id = resp.json().get("id")
        
        sessions[name] = session_id
        SESSION_MAP_FILE.parent.mkdir(parents=True, exist_ok=True)
        SESSION_MAP_FILE.write_text(json.dumps(sessions, indent=2))
        return session_id
    except Exception as e:
        print(f"Error creating session: {e}")
        sys.exit(1)

class Client:
    def __init__(self, session_id):
        self.session_id = session_id
        self.sock = None

    def connect(self):
        try:
            self.sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            self.sock.connect((DAEMON_HOST, DAEMON_PORT))
            return True
        except ConnectionRefusedError:
            return False

    def send_request(self, action, payload=None):
        if not self.sock:
            if not self.connect():
                # Try to spawn daemon
                if not self._check_daemon_running():
                    print("Starting daemon...")
                    subprocess.Popen([sys.executable, "-m", "opencode_wrapper", "--daemon"], 
                                     cwd=str(PROJECT_ROOT),
                                     stdout=subprocess.DEVNULL, 
                                     stderr=subprocess.DEVNULL)
                    time.sleep(2) # Wait for start
                    if not self.connect():
                        print("Failed to connect to daemon.")
                        sys.exit(1)
        
        req = {
            "action": action,
            "session_id": self.session_id,
            "payload": payload
        }
        self.sock.sendall(json.dumps(req).encode('utf-8'))
        
        # Receive response
        try:
            resp_data = self.sock.recv(4096).decode('utf-8')
            return json.loads(resp_data)
        except Exception as e:
            print(f"Error reading response: {e}")
            return None
        finally:
            if self.sock:
                self.sock.close()
                self.sock = None

    def _check_daemon_running(self):
        return False

    def wait_for_result(self):
        start_time = time.time()
        print(f"Waiting for result (Timeout: {CLIENT_TIMEOUT}s)...")
        
        while time.time() - start_time < CLIENT_TIMEOUT:
            resp = self.send_request("GET_STATUS")
            if resp and resp.get("status") == "ok":
                data = resp["data"]
                state = data["state"]
                questions = data["questions"]
                result = data["latest_response"]
                
                if questions:
                    print("\n" + "="*40)
                    print("  ACTION REQUIRED")
                    print("="*40)
                    for q in questions:
                        print(f"[?] Request ID: {q['id']}")
                        for sub_q in q.get('questions', []):
                            print(f"    {sub_q.get('question')}")
                            if sub_q.get('options'):
                                print("    Options available.")
                    print("\nRun: `python -m opencode_wrapper <session> /answer '...'`")
                    return
                
                if state == "IDLE" and result:
                    if result.get("error"):
                         print(f"Error: {result['error']}")
                    else:
                         print("Response received:")
                         print(json.dumps(result.get("result"), indent=2))
                    return
                
                time.sleep(2)
            else:
                print("Error checking status via daemon.")
                time.sleep(2)
        
        print("\n[TIMEOUT] Message is taking longer than 5 minutes.")
        print("Daemon is still running in background.")
        print("Run: `python -m opencode_wrapper <session> /wait` to check again.")

def run_client(args):
    # Resolve Name -> ID
    session_id = resolve_session(args.session_name)
    client = Client(session_id)
    
    # Ensure session is managed
    client.send_request("START_SESSION")
    
    if args.message == "/wait":
        client.wait_for_result()
        
    elif args.message and args.message.startswith("/answer"):
        parts = args.message.split(" ", 1)
        if len(parts) < 2:
            print("Usage: /answer <answer_text_or_json>")
            return
        
        status = client.send_request("GET_STATUS")
        questions = status.get("data", {}).get("questions", [])
        if not questions:
            print("No pending questions.")
            return

        request_id = questions[0]['id']
        answer_text = parts[1]
        
        # Simple wrap for now: [[answer_text]]
        payload = {
            "requestID": request_id, 
            "answers": [[answer_text]]
        }
        
        resp = client.send_request("ANSWER", payload)
        print(f"Answer status: {resp.get('message')}")
        client.wait_for_result() # Wait for continued execution
        
    elif args.message and args.message.startswith("/"):
        # Command
        cmd_parts = args.message.split(maxsplit=1)
        command = cmd_parts[0][1:]
        arguments = cmd_parts[1] if len(cmd_parts) > 1 else ""
        
        payload = {
            "agent": args.agent,
            "model": parse_model_string(args.model),
            "command": command,
            "arguments": arguments,
            "parts": []
        }
        resp = client.send_request("COMMAND", payload)
        print(f"Command sent: {resp.get('message')}")
        client.wait_for_result()
        
    elif args.message:
        # Prompt
        payload = {
            "agent": args.agent,
            "model": parse_model_string(args.model),
            "parts": [{"type": "text", "text": args.message}]
        }
        resp = client.send_request("PROMPT", payload)
        print(f"Prompt sent: {resp.get('message')}")
        client.wait_for_result()
    else:
        print("No command provided.")

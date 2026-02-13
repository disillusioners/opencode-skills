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
    except requests.exceptions.ConnectionError:
        print(f"Error: Could not connect to OpenCode server at {OPENCODE_URL}.")
        print("Please ensure 'opencode serve' is running.")
        sys.exit(1)
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
                    try:
                        with open('/tmp/opencode_daemon_startup.err', 'w') as err_file:
                            subprocess.Popen([sys.executable, "-m", "opencode_wrapper", "--daemon"], 
                                             cwd=str(PROJECT_ROOT),
                                             stdout=err_file, 
                                             stderr=err_file,
                                             start_new_session=True)
                    except Exception as e:
                        print(f"Failed to start daemon: {e}")
                        sys.exit(1)

                    # Retry connecting for up to 5 seconds
                    for _ in range(10):
                        time.sleep(0.5)
                        if self.connect():
                            break
                    else:
                        print("Failed to connect to daemon after start attempt.")
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
                                print("    Options:")
                                for opt in sub_q.get('options'):
                                    label = opt.get('label', '')
                                    desc = opt.get('description', '')
                                    if desc:
                                        print(f"      - {label}: {desc}")
                                    else:
                                        print(f"      - {label}")
                    print("\nRun: `python3 path_to/opencode_wrapper.py <session> /answer '...'`")
                    return
                
                if state == "IDLE" and result:
                    if result.get("error"):
                         print(f"Error: {result['error']}")
                    else:
                         print("Response received:")
                         print(json.dumps(result.get("result"), indent=2))
                    return
                
                time.sleep(3)
            else:
                print("Error checking status via daemon.")
                time.sleep(3)
        
        print("\n[TIMEOUT] Message is taking longer than 5 minutes.")
        print("Daemon is still running in background.")
        print("Run: `python3 path_to/opencode_wrapper.py <session> /wait` to check again.")

def run_client(args):
    # Resolve Name -> ID
    session_id = resolve_session(args.session_name)
    client = Client(session_id)
    
    # Ensure session is managed (but not for /status, which is read-only)
    if args.message and args.message[0] != "/status":
        client.send_request("START_SESSION")
    
    # args.message is now a list strings (shell split)
    if not args.message:
        print("No command provided.")
        return

    cmd = args.message[0]
    
    if cmd == "/wait":
        client.wait_for_result()
    
    elif cmd == "/status":
        # Display current session status without waiting
        resp = client.send_request("GET_STATUS")
        if resp and resp.get("status") == "ok":
            data = resp["data"]
            state = data.get("state", "UNKNOWN")
            questions = data.get("questions", [])
            result = data.get("latest_response")
            
            print("\n" + "="*40)
            print(f"  SESSION STATUS: {state}")
            print("="*40)
            
            if questions:
                print("\n[QUESTIONS PENDING]")
                for q in questions:
                    print(f"Request ID: {q['id']}")
                    for sub_q in q.get('questions', []):
                        print(f"  - {sub_q.get('question')}")
                        if sub_q.get('options'):
                            print("    Options:")
                            for opt in sub_q.get('options'):
                                label = opt.get('label', '')
                                desc = opt.get('description', '')
                                if desc:
                                    print(f"      - {label}: {desc}")
                                else:
                                    print(f"      - {label}")
            
            if result:
                print("\n[LATEST RESPONSE]")
                if result.get("error"):
                    print(f"Error: {result['error']}")
                elif result.get("result"):
                    print(json.dumps(result.get("result"), indent=2))
                else:
                    print("No response data")
            
            if state == "IDLE" and not questions and not result:
                print("\nSession is idle with no pending work.")
            elif state == "BUSY":
                print("\nSession is currently processing...")
                print("Run `/wait` to monitor for completion.")
        else:
            error_msg = resp.get('message', 'Unknown error')
            if 'not found' in error_msg.lower():
                print(f"Error: Session '{args.session_name}' does not exist or is not active.")
                print("\nTo start a new session, send a prompt:")
                print(f"  python3 path_to/opencode_wrapper.py {args.session_name} \"Your prompt\"")
            else:
                print(f"Failed to get session status: {error_msg}")
        
    elif cmd == "/answer":
        # Usage: /answer "Ans1" "Ans2" ...
        answer_parts = args.message[1:]
        if not answer_parts:
            print("Usage: /answer <answer_text> [answer_text...]")
            return
        
        status = client.send_request("GET_STATUS")
        questions = status.get("data", {}).get("questions", [])
        if not questions:
            print("No pending questions.")
            return

        # Matches first request ID
        request_id = questions[0]['id']
        
        # Format payload: answers = [["Ans1"], ["Ans2"]]
        # This maps 1 CLI arg -> 1 Question answer (single selection)
        formatted_answers = [[ans] for ans in answer_parts]

        payload = {
            "requestID": request_id, 
            "answers": formatted_answers
        }
        
        resp = client.send_request("ANSWER", payload)
        print(f"Answer status: {resp.get('message')}")
        time.sleep(3) # Wait for backend to process answer and clear question
        client.wait_for_result() # Wait for continued execution

    elif cmd.startswith("/"):
        # Command: /cmd arg1 arg2
        command = cmd[1:]
        # Join arguments by space to emulate string behavior
        # But if arguments were quoted, we lose that distinction if we join?
        # Standard command payload usually expects 'arguments' as a single string?
        # Let's join with space.
        arguments = " ".join(args.message[1:])
        
        payload = {
            "agent": args.agent,
            "model": args.model,  # Command endpoint expects string format
            "command": command,
            "arguments": arguments,
            "parts": []
        }
        resp = client.send_request("COMMAND", payload)
        print(f"Command sent: {resp.get('message')}")
        client.wait_for_result()
        
    else:
        # Prompt: "Hello world"
        # Join all parts with space
        full_message = " ".join(args.message)
        payload = {
            "agent": args.agent,
            "model": parse_model_string(args.model),
            "parts": [{"type": "text", "text": full_message}]
        }
        resp = client.send_request("PROMPT", payload)
        print(f"Prompt sent: {resp.get('message')}")
        client.wait_for_result()

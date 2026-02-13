import socket
import json
import os
import signal
import sys
import logging
import threading
from .config import DAEMON_HOST, DAEMON_PORT, PID_FILE, SESSION_MAP_FILE
from .manager import SessionManager

# Setup Logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    filename='/tmp/opencode_daemon.log',
    filemode='a'
)
logger = logging.getLogger("OpencodeDaemon")

class DaemonServer:
    def __init__(self):
        self.sessions = {} # session_id -> SessionManager
        self.server_socket = None
        self.running = False

    def start(self):
        self.running = True
        self._write_pid()
        self._load_sessions()
        
        self.server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.server_socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        self.server_socket.bind((DAEMON_HOST, DAEMON_PORT))
        self.server_socket.listen(5)
        
        logger.info(f"Daemon listening on {DAEMON_HOST}:{DAEMON_PORT}")
        
        signal.signal(signal.SIGTERM, self.stop)
        signal.signal(signal.SIGINT, self.stop)

        try:
            while self.running:
                client_sock, addr = self.server_socket.accept()
                threading.Thread(target=self.handle_client, args=(client_sock,)).start()
        except Exception as e:
            logger.error(f"Server loop error: {e}")
        finally:
            self.cleanup()

    def handle_client(self, client_sock):
        try:
            data = client_sock.recv(4096).decode('utf-8')
            if not data:
                return
            
            req = json.loads(data)
            action = req.get("action")
            session_id = req.get("session_id")
            
            response = {"status": "error", "message": "Unknown action"}
            
            if action == "PING":
                response = {"status": "ok", "message": "PONG"}
                
            elif action == "START_SESSION":
                # Ensure Manager exists
                if session_id not in self.sessions:
                    manager = SessionManager(session_id)
                    manager.start()
                    self.sessions[session_id] = manager
                    self._save_sessions()
                    logger.info(f"Started manager for session {session_id}")
                response = {"status": "ok", "message": "Session managed"}
                
            elif action == "GET_STATUS":
                manager = self.sessions.get(session_id)
                if manager:
                    snapshot = manager.get_snapshot()
                    response = {"status": "ok", "data": snapshot}
                else:
                    response = {"status": "error", "message": "Session not found"}
            
            elif action in ["PROMPT", "COMMAND", "ANSWER", "FIX"]:
                manager = self.sessions.get(session_id)
                if manager:
                    # Only block regular PROMPT when busy (not COMMAND or special prompts)
                    if action == "PROMPT":
                        snapshot = manager.get_snapshot()
                        payload = req.get("payload", {})
                        
                        # Check if it's a special prompt like "start-work"
                        is_special_prompt = False
                        parts = payload.get("parts", [])
                        if parts and len(parts) > 0:
                            text = parts[0].get("text", "").strip().lower()
                            # Allow special prompts like "start-work", "continue", etc.
                            if text in ["start-work", "continue", "abort", "retry"]:
                                is_special_prompt = True
                        
                        if snapshot["state"] == "BUSY" and not is_special_prompt:
                            response = {"status": "error", "message": "Session is busy processing another request. Please wait."}
                        else:
                            internal_req = {"type": action, "payload": payload}
                            logger.info(f"Submitting {action} to manager {session_id}")
                            manager.submit_request(internal_req)
                            response = {"status": "ok", "message": "Request submitted"}
                    else:
                        # COMMAND, ANSWER, and FIX can always be submitted
                        internal_req = {"type": action, "payload": req.get("payload")}
                        logger.info(f"Submitting {action} to manager {session_id}")
                        manager.submit_request(internal_req)
                        response = {"status": "ok", "message": "Request submitted"}
                else:
                    response = {"status": "error", "message": "Session not found"}

            client_sock.sendall(json.dumps(response).encode('utf-8'))
            
        except Exception as e:
            logger.error(f"Client handler error: {e}")
            err_resp = {"status": "error", "message": str(e)}
            client_sock.sendall(json.dumps(err_resp).encode('utf-8'))
        finally:
            client_sock.close()

    def stop(self, signum=None, frame=None):
        logger.info("Stopping daemon...")
        self.running = False
        if self.server_socket:
            self.server_socket.close()
        for manager in self.sessions.values():
            manager.stop()
        sys.exit(0)

    def cleanup(self):
        if PID_FILE.exists():
            PID_FILE.unlink()

    def _write_pid(self):
        PID_FILE.write_text(str(os.getpid()))

    def _load_sessions(self):
        # Could load from file to resume, but threads are transient
        # For now, start fresh or rely on client to re-init
        pass

    def _save_sessions(self):
        # Persist list of active sessions if needed
        pass

if __name__ == "__main__":
    # Double-fork logic usually goes here for true daemon
    # For now, just running the loop is enough if spawned via subprocess
    DaemonServer().start()

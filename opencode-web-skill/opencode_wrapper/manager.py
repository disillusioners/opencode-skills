import time
import requests
import queue
import logging
from threading import Thread, Event
from .config import OPENCODE_URL, PROJECT_ROOT
from .worker import Worker

logger = logging.getLogger("OpencodeManager")

class SessionManager(Thread):
    def __init__(self, session_id):
        super().__init__()
        self.session_id = session_id
        self.input_queue = queue.Queue()
        self.worker = None
        self.state = "IDLE" 
        self.latest_response = None
        self.questions = []
        self._stop_event = Event()
        self.last_activity = 0
        self.daemon = True # Run as daemon thread
        
    def run(self):
        while not self._stop_event.is_set():
            try:
                # 1. Process Input
                req = self.input_queue.get(timeout=1.0)
                self._handle_request(req)
            except queue.Empty:
                pass
            
            # 2. Check Worker Status
            if self.worker and not self.worker.is_alive():
                # Worker finished
                self.worker = None
                if self.state == "BUSY":
                    self.state = "IDLE"

            # 3. Poll Questions (Throttle to 2s)
            if time.time() - self.last_activity > 2.0:
                 self._poll_questions()
                 self.last_activity = time.time()

    def submit_request(self, req):
        self.input_queue.put(req)

    def get_snapshot(self):
        return {
            "state": self.state,
            "session_id": self.session_id,
            "latest_response": self.latest_response,
            "questions": self.questions
        }

    def _handle_request(self, req):
        r_type = req.get("type")
        payload = req.get("payload")
        
        if r_type in ["PROMPT", "COMMAND"]:
            if self.worker and self.worker.is_alive():
                # We ignore new prompts if busy (Client should check state first)
                logger.warning(f"Session {self.session_id} is busy. Ignoring {r_type}.")
                return
            
            self.state = "BUSY"
            self.latest_response = None # Clear previous
            endpoint = "command" if r_type == "COMMAND" else "message"
            self.worker = Worker(self.session_id, payload, self.on_worker_done, endpoint=endpoint)
            self.worker.start()
            
        elif r_type == "ANSWER":
            # Handle Answer immediately (unblocks agent)
            try:
                url = f"{OPENCODE_URL}/question/{payload['requestID']}/reply"
                headers = {
                    "Content-Type": "application/json",
                    "x-opencode-directory": str(PROJECT_ROOT)
                }
                # Wrap answer in list if needed or assume Client formatted it
                # Wrapper protocol: Client sends formatted payload { "answers": [...] }
                # The payload here is exactly what we send to API? 
                # Let's assume payload has "answers" key.
                requests.post(url, json={"answers": payload["answers"]}, headers=headers)
                
                # Optimistically remove question
                self.questions = [q for q in self.questions if q["id"] != payload["requestID"]]
                
                # Update State
                if not self.questions:
                    if self.worker and self.worker.is_alive():
                        self.state = "BUSY"
                    else:
                        self.state = "IDLE"
            except Exception as e:
                logger.error(f"Answer failed: {e}")

        elif r_type == "FIX":
            try:
                # 1. Abort
                logger.info(f"Aborting session {self.session_id}...")
                requests.post(f"{OPENCODE_URL}/session/{self.session_id}/abort", 
                              json={}, 
                              headers={"x-opencode-directory": str(PROJECT_ROOT)})
                
                # 2. Reset Worker (if possible, start new)
                # We can't easily kill the old thread, but we can ignore it.
                # However, sending 'continue' message might fail if old request is still pending?
                # Abort should make the old request return.
                # Let's hope the old worker finishes quickly.
                
                # 3. Send Continue
                logger.info(f"Sending continue to session {self.session_id}...")
                self.latest_response = None
                self.state = "BUSY"
                
                payload = {
                    "agent": "sisyphus", # Default fallback
                    "model": {"providerID": "zai-coding-plan", "modelID": "glm-4.7"}, # Default fallback
                    "parts": [{"type": "text", "text": "continue"}]
                }
                # Use default model/agent if payload missing? Client should provide them ideally.
                # Assuming generic continue is fine with defaults.
                
                # We overwrite self.worker. The old thread becomes orphaned and eventually dies.
                self.worker = Worker(self.session_id, payload, self.on_worker_done, endpoint="message")
                self.worker.start()
                
            except Exception as e:
                logger.error(f"Fix failed: {e}")
                self.state = "IDLE"
                self.latest_response = {"result": None, "error": f"Fix failed: {e}"}

    def on_worker_done(self, result, error):
        self.latest_response = {"result": result, "error": error}
        logger.info(f"Worker finished for {self.session_id}")

    def _poll_questions(self):
        try:
            url = f"{OPENCODE_URL}/question"
            headers = {"x-opencode-directory": str(PROJECT_ROOT)}
            resp = requests.get(url, headers=headers, timeout=5)
            if resp.status_code == 200:
                resp_json = resp.json()
                data = resp_json if isinstance(resp_json, list) else resp_json.get('data', [])
                
                # Filter for this session
                self.questions = [q for q in data if q.get('sessionID') == self.session_id]
                
                if self.questions:
                    self.state = "WAITING_FOR_INPUT"
                elif self.state == "WAITING_FOR_INPUT":
                    # If invalid/gone, and worker is running -> BUSY
                    if self.worker and self.worker.is_alive():
                        self.state = "BUSY"
                    else:
                        self.state = "IDLE"
        except Exception as e:
            logger.error(f"Poll Questions Error: {e}")

    def stop(self):
        self._stop_event.set()

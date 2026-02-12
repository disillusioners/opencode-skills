import requests
import logging
from threading import Thread, Event
from .config import OPENCODE_URL, PROJECT_ROOT

logger = logging.getLogger("OpencodeWorker")

class Worker(Thread):
    def __init__(self, session_id, payload, on_complete, endpoint="message"):
        super().__init__()
        self.session_id = session_id
        self.payload = payload
        self.on_complete = on_complete
        self.endpoint = endpoint
        self._stop_event = Event()
        self.daemon = True

    def run(self):
        try:
            url = f"{OPENCODE_URL}/session/{self.session_id}/{self.endpoint}"
            headers = {
                "Content-Type": "application/json",
                "x-opencode-directory": str(PROJECT_ROOT)
            }
            logger.info(f"Worker sending {self.endpoint} for session {self.session_id}")
            
            response = requests.post(url, json=self.payload, headers=headers)
            response.raise_for_status()
            
            result = response.json()
            logger.info(f"Worker received response for session {self.session_id}")
            self.on_complete(result, error=None)
            
        except Exception as e:
            logger.error(f"Worker error: {e}")
            self.on_complete(None, error=str(e))

    def stop(self):
        self._stop_event.set()

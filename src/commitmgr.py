import json
import os
import sys
import time
import threading
import socket
from src.server import SupabaseAPI

if getattr(sys, 'frozen', False):  # Running as compiled EXE
    BASE_PATH = os.path.dirname(sys.executable)
else:  # Running from source
    BASE_PATH = os.path.dirname(os.path.abspath(__file__))

commit_file = os.path.join(BASE_PATH, "pending_commits.json")

class JsonCommitQueue:
    def __init__(self, api: SupabaseAPI, file_path=commit_file, check_interval=5):
        self.api = api
        self.file_path = file_path
        self.check_interval = check_interval
        self._stop_event = threading.Event()
        self._thread = threading.Thread(target=self._worker, daemon=True)
        self._thread.start()

    def _internet_available(self) -> bool:
        try:
            host = self.api.supabase.rest_url.replace("https://", "").split("/")[0]
            socket.create_connection((host, 443), timeout=2)
            return True
        except OSError:
            return False

    def _load_queue(self):
        if not os.path.exists(self.file_path):
            return []
        with open(self.file_path, "r", encoding="utf-8") as f:
            try:
                return json.load(f)
            except json.JSONDecodeError:
                return []

    def _save_queue(self, queue_data):
        with open(self.file_path, "w", encoding="utf-8") as f:
            json.dump(queue_data, f, indent=2)

    def submit_commit(self, device_id: str, location: str, delta: int, item_id: int):
        """Add a commit to the local JSON queue."""
        commit = {
            "device_id": device_id,
            "location": location,
            "delta": delta,
            "item_id": item_id,
        }
        queue_data = self._load_queue()
        queue_data.append(commit)
        self._save_queue(queue_data)
        print(f"Commit queued locally: {commit}")

    def _worker(self):
        while not self._stop_event.is_set():
            if self._internet_available():
                queue_data = self._load_queue()
                if queue_data:
                    print(f"Found {len(queue_data)} pending commits, trying to send...")
                new_queue = []
                for commit in queue_data:
                    try:
                        result = self.api.send_commit(**commit)
                        print("Committed:", result)
                    except Exception as e:
                        print("Commit failed, keeping in queue:", e)
                        new_queue.append(commit)
                self._save_queue(new_queue)
            time.sleep(self.check_interval)

    def stop(self):
        self._stop_event.set()
        self._thread.join()

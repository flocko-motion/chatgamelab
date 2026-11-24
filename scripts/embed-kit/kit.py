#!/usr/bin/env python3
"""The embed test kit: a blank website on another origin to try an embedded
game on, and the callback a verified page registers. Standard library only.

    python3 scripts/embed-kit/kit.py [--port 8000] [--platform http://localhost:7100]

Open http://localhost:8000, paste the embed snippet, press Run. For a
verified page, register http://localhost:8000/ as the page and
http://localhost:8000/callback as the callback (plain http to the own
machine is accepted in development only). The page chooses what the callback
answers and shows what it was asked. See README.md.
"""
import argparse
import json
import threading
import time
from datetime import datetime
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

HERE = Path(__file__).resolve().parent
SILENCE = 8  # seconds; the platform gives up after 5

state = {"answer": "yes", "log": []}
lock = threading.Lock()


class Kit(BaseHTTPRequestHandler):
    platform = "http://localhost:7100"

    def do_GET(self):
        if self.path in ("/", "/index.html"):
            page = (HERE / "index.html").read_text(encoding="utf-8")
            self.reply(200, "text/html; charset=utf-8", page.replace("{{PLATFORM}}", self.platform).encode())
        elif self.path == "/state":
            with lock:
                self.reply(200, "application/json", json.dumps(state).encode())
        else:
            self.reply(404, "text/plain", b"not here")

    def do_POST(self):
        body = self.rfile.read(int(self.headers.get("Content-Length") or 0))
        if self.path == "/answer":
            answer = json.loads(body or b"{}").get("answer")
            if answer in ("yes", "no", "silent"):
                with lock:
                    state["answer"] = answer
            self.reply(204, "text/plain", b"")
        elif self.path == "/callback":
            self.callback(body)
        else:
            self.reply(404, "text/plain", b"not here")

    def callback(self, body):
        with lock:
            answer = state["answer"]
            try:
                asked = json.loads(body)
            except ValueError:
                asked = body.decode(errors="replace")
            state["log"].insert(0, {"at": datetime.now().strftime("%H:%M:%S"), "asked": asked, "answer": answer})
            del state["log"][50:]
        print(f"callback asked {asked} -> {answer}", flush=True)
        if answer == "silent":
            time.sleep(SILENCE)
        self.reply(200, "application/json", json.dumps({"ok": answer != "no"}).encode())

    def reply(self, status, content_type, body):
        self.send_response(status)
        self.send_header("Content-Type", content_type)
        self.send_header("Cache-Control", "no-store")
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *_):
        pass


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--port", type=int, default=8000)
    parser.add_argument("--platform", default="http://localhost:7100", help="the ChatGameLab the snippets come from")
    args = parser.parse_args()
    Kit.platform = args.platform.rstrip("/")
    print(f"embed kit on http://localhost:{args.port}  (callback: http://localhost:{args.port}/callback)", flush=True)
    ThreadingHTTPServer(("127.0.0.1", args.port), Kit).serve_forever()


if __name__ == "__main__":
    main()

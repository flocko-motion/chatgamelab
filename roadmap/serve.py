#!/usr/bin/env python3
"""Serves the roadmap page on this machine (make roadmap-local) and reloads
its data on request: POST /api/reload reads every issue fresh (about a
minute), POST /api/reload?issue=N only what concerns issue N (seconds). GitHub's
answers are kept in DIR/cache between reloads. The page finds this server
through GET /api/local and then shows its reload buttons.

Usage: roadmap/serve.py DIR [PORT] [--fresh]
"""
import http.server
import json
import os
import subprocess
import sys
import threading
from urllib.parse import parse_qs, urlparse

args = [a for a in sys.argv[1:] if not a.startswith("--")]
DIR = args[0]
PORT = int(args[1]) if len(args) > 1 else 8765
HERE = os.path.dirname(os.path.abspath(__file__))
CACHE = os.path.join(DIR, "cache")
JSON = os.path.join(DIR, "roadmap.json")
busy = threading.Lock()


def build(extra):
    """Runs roadmap.py over the cache into roadmap.json; a failed run keeps the old one."""
    tmp = JSON + ".tmp"
    with open(tmp, "w") as out:
        run = subprocess.run([sys.executable, os.path.join(HERE, "roadmap.py"), "--cache", CACHE, *extra],
                             stdout=out, stderr=subprocess.PIPE, text=True)
    if run.returncode:
        os.remove(tmp)
        lines = run.stderr.strip().splitlines()
        raise RuntimeError(lines[-1] if lines else "roadmap.py failed")
    os.replace(tmp, JSON)


class Handler(http.server.SimpleHTTPRequestHandler):
    def __init__(self, *a, **k):
        super().__init__(*a, directory=DIR, **k)

    def reply(self, code, body):
        data = json.dumps(body).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def do_GET(self):
        if self.path == "/api/local":
            return self.reply(200, {"local": True})
        super().do_GET()

    def do_POST(self):
        url = urlparse(self.path)
        if url.path != "/api/reload":
            return self.reply(404, {"error": "not found"})
        issue = parse_qs(url.query).get("issue", [""])[0]
        if issue and not issue.isdigit():
            return self.reply(400, {"error": "issue must be a number"})
        if not busy.acquire(blocking=False):
            return self.reply(409, {"error": "a reload is running already"})
        try:
            build(["--drop", issue] if issue else ["--fresh"])
            self.reply(200, {"ok": True})
        except Exception as e:  # reported to the page, which shows it
            self.reply(500, {"error": str(e)})
        finally:
            busy.release()


if "--fresh" in sys.argv or not os.path.exists(JSON) or not os.path.getsize(JSON):
    print("reading the issues…", flush=True)
    build(["--fresh"])
print(f"→ http://localhost:{PORT}", flush=True)
http.server.ThreadingHTTPServer(("127.0.0.1", PORT), Handler).serve_forever()

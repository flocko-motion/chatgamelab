#!/usr/bin/env python3
"""
KI-Benchmark für ChatGameLab – kleine lokale Webseite zum Vergleichen von
OpenAI-Modellen (zurzeit Bilder; Texte und Audio folgen).

Start:  python3 scripts/ai-benchmark/server.py   (oder Doppelklick auf start.command)
Dann öffnet sich http://127.0.0.1:8777 im Browser.

Braucht nur Python 3, keine Zusatzpakete. Der OpenAI-Schlüssel wird auf der
Seite eingegeben und nur im Arbeitsspeicher gehalten – auf Wunsch zusätzlich
im Mac-Schlüsselbund. Läufe landen in results/<Zeitstempel>/ (nicht in Git).
"""

import argparse
import datetime as dt
import json
import os
import queue
import re
import statistics
import subprocess
import sys
import threading
import webbrowser
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

import bench_image as img

HERE = Path(__file__).resolve().parent
RESULTS = HERE / "results"
STATIC = HERE / "static"
KEYCHAIN_SERVICE = "chatgamelab-ai-benchmark"
BASE_URL = os.environ.get("OPENAI_BASE_URL", "https://api.openai.com/v1")

lock = threading.Lock()
jobs: "queue.Queue[dict]" = queue.Queue()
state = {
    "api_key": os.environ.get("OPENAI_API_KEY") or None,
    "current": None,   # {"run_id", "done", "total", "label", "job_nr", "job_total"}
    "queued": 0,
    "stop": False,
}


# ---------------------------------------------------------------------------
# Schlüssel im Mac-Schlüsselbund
# ---------------------------------------------------------------------------
def keychain_read():
    if sys.platform != "darwin":
        return None
    try:
        out = subprocess.run(["security", "find-generic-password", "-s", KEYCHAIN_SERVICE, "-w"],
                             capture_output=True, text=True, timeout=10)
        if out.returncode != 0:
            return None
        return out.stdout.strip() or None
    except Exception:
        return None


def keychain_write(key):
    if sys.platform != "darwin":
        return False
    out = subprocess.run(["security", "add-generic-password", "-U", "-s", KEYCHAIN_SERVICE,
                          "-a", "openai", "-w", key], capture_output=True, timeout=10)
    return out.returncode == 0


def keychain_delete():
    if sys.platform == "darwin":
        subprocess.run(["security", "delete-generic-password", "-s", KEYCHAIN_SERVICE],
                       capture_output=True, timeout=10)


# ---------------------------------------------------------------------------
# Läufe speichern / laden
# ---------------------------------------------------------------------------
RUN_ID = re.compile(r"^[0-9A-Za-z_\-]+$")
FILE_NAME = re.compile(r"^[0-9A-Za-z_.\-]+\.(png|jpg|jpeg|webp)$")


def save_run(run):
    d = RESULTS / run["id"]
    d.mkdir(parents=True, exist_ok=True)
    tmp = d / "run.json.tmp"
    tmp.write_text(json.dumps(run, indent=2, ensure_ascii=False), encoding="utf-8")
    tmp.replace(d / "run.json")


def rating_key(model, quality, run_nr):
    return f"{model}|{quality}|{run_nr}"


def load_ratings(run_id):
    """Bewertungen (0–10) liegen getrennt in ratings.json, damit ein laufender Test sie nicht überschreibt."""
    f = RESULTS / run_id / "ratings.json"
    try:
        return json.loads(f.read_text(encoding="utf-8"))
    except Exception:
        return {}


def load_runs():
    runs = []
    if RESULTS.exists():
        for f in sorted(RESULTS.glob("*/run.json")):
            try:
                run = json.loads(f.read_text(encoding="utf-8"))
            except Exception:
                continue
            ratings = load_ratings(run["id"])
            for v in run.get("results", []):
                for rec in v.get("runs", []):
                    rec["rating"] = ratings.get(rating_key(v["model"], v["quality"], rec.get("run", 1)))
                    if rec.get("usage"):  # immer mit der aktuellen Preisliste rechnen
                        rec["cost_usd"] = img.cost_from_usage(v["model"], rec["usage"])
            runs.append(run)
    return runs


def historical_medians(size):
    """Median-Dauer je (Modell, Qualität) aus allen bisherigen Läufen dieser Bildgröße."""
    times = {}
    for run in load_runs():
        if run.get("size") != size:
            continue
        for v in run.get("results", []):
            for rec in v.get("runs", []):
                if rec.get("final_s") is not None:
                    times.setdefault((v["model"], v["quality"]), []).append(rec["final_s"])
    return {k: statistics.median(v) for k, v in times.items()}


def new_run_id():
    base = dt.datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
    rid, n = base, 2
    while (RESULTS / rid).exists():
        rid, n = f"{base}-{n}", n + 1
    return rid


# ---------------------------------------------------------------------------
# Hintergrund-Arbeiter: arbeitet die Läufe nacheinander ab
# ---------------------------------------------------------------------------
def worker():
    while True:
        job = jobs.get()
        with lock:
            state["queued"] = max(0, state["queued"] - 1)
            if state["stop"]:
                if state["queued"] == 0:
                    state["stop"] = False
                continue
        run_image_job(job)


def run_image_job(job):
    run = {
        "id": new_run_id(),
        "kind": "image",
        "status": "running",
        "started": dt.datetime.now().isoformat(timespec="seconds"),
        "finished": None,
        "label": job.get("label") or "",
        "prompt": job["prompt"],
        "size": job["size"],
        "prices_per_million": img.PRICES_PER_MILLION,
        "results": [],
    }
    limit = job["max_seconds"]
    run["max_seconds"] = limit
    qualities = [q for q in img.QUALITIES if q in job["qualities"]]  # immer low → medium → high
    variants = [(m, q) for m in job["models"] for q in qualities]
    hist = historical_medians(job["size"])
    with lock:
        state["current"] = {"run_id": run["id"], "done": 0, "total": len(variants), "label": "",
                            "job_nr": job["nr"], "job_total": job["of"]}
    save_run(run)
    try:
        prev = {}  # Modell -> Ergebnis der vorigen Qualitätsstufe in diesem Lauf
        for i, (model, quality) in enumerate(variants):
            with lock:
                if state["stop"]:
                    run["status"] = "stopped"
                    break
                state["current"].update(done=i, label=f"{model} · {quality}")
            skip = None
            p = prev.get(model)
            if hist.get((model, quality), 0) > limit:
                skip = f"übersprungen – braucht bisher im Mittel {hist[(model, quality)]:.0f} s (Limit {limit} s)"
            elif p and p.get("timeout"):
                skip = f"übersprungen – schon die Stufe darunter lag über {limit} s"
            elif p and p.get("skipped"):
                skip = "übersprungen – die Stufe darunter wurde schon übersprungen"
            elif p and p.get("final_s") and p["final_s"] * img.NEXT_QUALITY_FACTOR > limit:
                skip = (f"übersprungen – die Stufe darunter brauchte {p['final_s']:.0f} s, "
                        f"diese bräuchte absehbar über {limit} s")
            if skip:
                rec = {"run": 1, "error": None, "skipped": skip, "final_s": None, "usage": None, "cost_usd": None}
            else:
                r = img.generate(state["api_key"], BASE_URL, model, quality, job["size"], job["prompt"], timeout=limit)
                rec = {"run": 1, "error": r["error"], "timeout": r.get("timeout", False),
                       "request_id": r["request_id"], "final_s": r["final_s"], "usage": r["usage"],
                       "cost_usd": img.cost_from_usage(model, r["usage"])}
                if r["image"]:
                    fname = f"{model}-{quality}.png"
                    (RESULTS / run["id"] / fname).write_bytes(r["image"])
                    rec["file"] = fname
            prev[model] = rec
            run["results"].append({"model": model, "quality": quality, "runs": [rec]})
            save_run(run)
        else:
            run["status"] = "done"
    except Exception as e:  # unerwarteter Fehler – Lauf trotzdem sauber abschließen
        run["status"] = "error"
        run["error"] = f"{type(e).__name__}: {e}"
    run["finished"] = dt.datetime.now().isoformat(timespec="seconds")
    save_run(run)
    with lock:
        state["current"] = None
        if state["queued"] == 0:
            state["stop"] = False


# ---------------------------------------------------------------------------
# HTTP
# ---------------------------------------------------------------------------
class Handler(BaseHTTPRequestHandler):
    def log_message(self, *a):
        pass

    def _json(self, obj, code=200):
        data = json.dumps(obj, ensure_ascii=False).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Cache-Control", "no-store")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def _file(self, path: Path, ctype):
        if not path.is_file():
            return self._json({"error": "nicht gefunden"}, 404)
        data = path.read_bytes()
        self.send_response(200)
        self.send_header("Content-Type", ctype)
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def _body(self):
        n = int(self.headers.get("Content-Length") or 0)
        return json.loads(self.rfile.read(n) or b"{}")

    def do_GET(self):
        p = self.path.split("?")[0]
        if p in ("/", "/index.html"):
            return self._file(STATIC / "index.html", "text/html; charset=utf-8")
        if p == "/api/state":
            with lock:
                key = state["api_key"]
                return self._json({
                    "has_key": bool(key),
                    "key_hint": ("…" + key[-4:]) if key else None,
                    "busy": state["current"] is not None or state["queued"] > 0,
                    "current": state["current"],
                    "queued": state["queued"],
                    "stopping": state["stop"],
                    "image": {
                        "default_models": img.DEFAULT_MODELS,
                        "model_order": img.MODEL_ORDER,
                        "deprecated": sorted(img.DEPRECATED),
                        "qualities": img.QUALITIES,
                        "sizes": img.SIZES,
                        "default_prompt": img.DEFAULT_PROMPT,
                        "prices_per_million": img.PRICES_PER_MILLION,
                        "provisional_prices": sorted(img.PROVISIONAL_PRICES),
                        "fallback_estimates": img.fallback_estimates(),
                        "default_max_seconds": img.DEFAULT_MAX_SECONDS,
                        "slow_seconds": 30,
                    },
                })
        if p == "/api/runs":
            return self._json(load_runs())
        if p == "/api/models":
            if not state["api_key"]:
                return self._json({"error": "Kein API-Schlüssel"}, 400)
            try:
                return self._json({"models": img.list_image_models(state["api_key"], BASE_URL)})
            except Exception as e:
                return self._json({"error": str(e)}, 502)
        m = re.match(r"^/results/([^/]+)/([^/]+)$", p)
        if m and RUN_ID.match(m.group(1)) and FILE_NAME.match(m.group(2)):
            ext = m.group(2).rsplit(".", 1)[1]
            return self._file(RESULTS / m.group(1) / m.group(2),
                              "image/png" if ext == "png" else f"image/{ext}")
        self._json({"error": "nicht gefunden"}, 404)

    def do_POST(self):
        p = self.path.split("?")[0]
        try:
            body = self._body()
        except Exception:
            return self._json({"error": "ungültige Anfrage"}, 400)

        if p == "/api/key":
            key = (body.get("key") or "").strip()
            if not key:
                return self._json({"error": "Bitte einen Schlüssel eingeben."}, 400)
            try:
                models = img.list_image_models(key, BASE_URL)
            except Exception as e:
                return self._json({"error": f"Schlüssel funktioniert nicht: {e}"}, 400)
            with lock:
                state["api_key"] = key
            saved = keychain_write(key) if body.get("remember") else False
            return self._json({"ok": True, "saved": saved, "models": models})

        if p == "/api/key/forget":
            keychain_delete()
            with lock:
                state["api_key"] = None
            return self._json({"ok": True})

        if p == "/api/runs":
            if not state["api_key"]:
                return self._json({"error": "Kein API-Schlüssel"}, 400)
            models = [m for m in body.get("models", []) if isinstance(m, str) and re.match(r"^[\w.\-]+$", m)]
            qualities = [q for q in body.get("qualities", []) if q in img.QUALITIES]
            size = body.get("size") if body.get("size") in img.SIZES else img.SIZES[0]
            prompt = (body.get("prompt") or "").strip()
            count = max(1, min(10, int(body.get("count") or 1)))
            max_seconds = max(5, min(600, int(body.get("max_seconds") or img.DEFAULT_MAX_SECONDS)))
            if not models or not qualities or not prompt:
                return self._json({"error": "Bitte Modelle, Qualitäten und Prompt angeben."}, 400)
            with lock:
                state["queued"] += count
                state["stop"] = False
            for n in range(count):
                jobs.put({"models": models, "qualities": qualities, "size": size, "prompt": prompt,
                          "max_seconds": max_seconds,
                          "label": (body.get("label") or "").strip()[:80], "nr": n + 1, "of": count})
            return self._json({"ok": True, "queued": count})

        if p == "/api/rate":
            run_id = str(body.get("run_id") or "")
            if not RUN_ID.match(run_id) or not (RESULTS / run_id / "run.json").exists():
                return self._json({"error": "Lauf nicht gefunden"}, 404)
            rating = body.get("rating")
            if rating is not None and (not isinstance(rating, int) or not 0 <= rating <= 10):
                return self._json({"error": "Bewertung muss 0–10 sein"}, 400)
            key = rating_key(str(body.get("model")), str(body.get("quality")), int(body.get("run") or 1))
            with lock:
                ratings = load_ratings(run_id)
                if rating is None:
                    ratings.pop(key, None)
                else:
                    ratings[key] = rating
                (RESULTS / run_id / "ratings.json").write_text(
                    json.dumps(ratings, indent=2, ensure_ascii=False), encoding="utf-8")
            return self._json({"ok": True})

        if p == "/api/stop":
            with lock:
                state["stop"] = True
            return self._json({"ok": True})

        self._json({"error": "nicht gefunden"}, 404)


def main():
    ap = argparse.ArgumentParser(description="KI-Benchmark für ChatGameLab")
    ap.add_argument("--port", type=int, default=8777)
    ap.add_argument("--no-open", action="store_true", help="Browser nicht automatisch öffnen")
    args = ap.parse_args()

    RESULTS.mkdir(exist_ok=True)
    if not state["api_key"]:
        state["api_key"] = keychain_read()
    # Läufe, die beim letzten Beenden noch liefen, als abgebrochen markieren
    for run in load_runs():
        if run.get("status") == "running":
            run["status"] = "stopped"
            save_run(run)

    threading.Thread(target=worker, daemon=True).start()
    server = ThreadingHTTPServer(("127.0.0.1", args.port), Handler)
    url = f"http://127.0.0.1:{args.port}"
    print(f"KI-Benchmark läuft: {url}\nBeenden mit Ctrl+C (oder dieses Fenster schließen).")
    if not args.no_open:
        webbrowser.open(url)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nBeendet.")


if __name__ == "__main__":
    main()

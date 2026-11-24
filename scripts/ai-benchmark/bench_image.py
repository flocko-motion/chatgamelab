"""
Bild-Teil des KI-Benchmarks: ruft die OpenAI-Bild-API auf (Streaming, ohne
Vorschaubilder), misst die Zeit und berechnet die Kosten aus den Tokens.
"""

import base64
import json
import socket
import time
import urllib.error
import urllib.request

QUALITIES = ["low", "medium", "high"]
SIZES = ["1024x1024", "1536x1024", "1024x1536"]

# Modelle, die standardmäßig angehakt sind. Weitere findet die Seite über
# „Weitere Modelle bei OpenAI suchen“.
DEFAULT_MODELS = ["gpt-image-1-mini", "gpt-image-1.5", "gpt-image-2"]

# Anzeige-Reihenfolge: vom ältesten (vermutlich schwächsten) zum neuesten Modell.
# Unbekannte Modelle kommen danach, alphabetisch.
MODEL_ORDER = [
    "gpt-image-1-mini",     # Okt. 2025
    "gpt-image-1.5",        # Dez. 2025
    "gpt-image-2",          # Apr. 2026
    "gpt-image-2.5-flare",  # Sep. 2026
    "gpt-image-2.6-flare",  # nach Sep. 2026
]

# Preise in US-Dollar pro 1 Million Tokens. Stand 24.11.2026, mit der echten
# OpenAI-Abrechnung abgeglichen (Abweichung unter 1 Cent pro Modell).
# Quelle zum Nachprüfen: https://platform.openai.com/docs/pricing
# text_out: gpt-image-1.5 erzeugt neben dem Bild auch Text-Tokens (internes
# „Nachdenken“), die zum Textpreis abgerechnet werden.
PRICES_PER_MILLION = {
    "gpt-image-2":      {"text_in": 5.00, "text_out": 10.00, "image_out": 30.00},
    "gpt-image-1.5":    {"text_in": 5.00, "text_out": 10.00, "image_out": 32.00},
    "gpt-image-1-mini": {"text_in": 2.00, "text_out": 10.00, "image_out": 8.00},
    # VORLÄUFIG (nur aus Drittquellen, noch nicht mit der Abrechnung abgeglichen):
    "gpt-image-2.5-flare": {"text_in": 5.00, "text_out": 10.00, "image_out": 30.00},
    "gpt-image-2.6-flare": {"text_in": 5.00, "text_out": 10.00, "image_out": 30.00},
}
PROVISIONAL_PRICES = {"gpt-image-2.5-flare", "gpt-image-2.6-flare"}

# Standard-Zeitlimit pro Bild. Was länger braucht, wird abgebrochen; höhere
# Qualitätsstufen werden übersprungen, wenn sie das Limit absehbar reißen.
DEFAULT_MAX_SECONDS = 45
# Erfahrungswert: die nächste Qualitätsstufe braucht bis zu ~3× so lange.
NEXT_QUALITY_FACTOR = 3

# Gemessene Tokens aus dem ersten Lauf (1024x1024) – nur für die Kostenschätzung,
# solange es für eine Variante noch keine eigenen Messwerte gibt.
EXPECTED_TOKENS = {
    "gpt-image-2":      {"low": (196, 0), "medium": (1756, 0), "high": (7024, 0)},
    "gpt-image-1.5":    {"low": (272, 240), "medium": (1056, 450), "high": (4160, 410)},
    "gpt-image-1-mini": {"low": (272, 0), "medium": (1056, 0), "high": (4160, 0)},
    # geraten (wie gpt-image-2), bis es Messwerte gibt:
    "gpt-image-2.5-flare": {"low": (196, 0), "medium": (1756, 0), "high": (7024, 0)},
    "gpt-image-2.6-flare": {"low": (196, 0), "medium": (1756, 0), "high": (7024, 0)},
}

DEFAULT_PROMPT = """Photorealistic photograph, shot on a full-frame camera with a 35 mm lens at f/2.8, natural late-afternoon sunlight filtering through old chestnut trees.

Scene: a traditional Bavarian beer garden on a crowded summer evening. In the foreground, a middle-aged innkeeper with a grey moustache, rolled-up white shirt sleeves and a worn brown leather apron holds an empty one-litre glass beer stein at an angle under the brass tap of a wooden beer barrel. A single last drop of beer is falling from the tap. His expression: shocked disbelief.

Behind him, at a long wooden table with a blue-and-white checked tablecloth, six guests of different ages in everyday summer clothes hold up their empty steins. Their faces show frustration, outrage and comic despair.

Details: dry glass with traces of foam, weathered wooden benches, gravel ground, strings of light bulbs between the trees, a chalkboard without readable text.

Realistic skin texture, anatomically correct hands and fingers, natural colours, shallow depth of field with the guests slightly out of focus. No text, no logos, no watermark."""


def cost_from_usage(model, usage):
    """Berechnete Kosten in USD aus den von der API gemeldeten Tokens (None = Preis unbekannt)."""
    if not usage or model not in PRICES_PER_MILLION:
        return None
    p = PRICES_PER_MILLION[model]
    ind = usage.get("input_tokens_details") or {}
    outd = usage.get("output_tokens_details") or {}
    text_in = ind.get("text_tokens", usage.get("input_tokens", 0)) or 0
    image_in = ind.get("image_tokens", 0) or 0
    image_out = outd.get("image_tokens", usage.get("output_tokens", 0)) or 0
    text_out = outd.get("text_tokens", 0) or 0
    return (text_in * p["text_in"] + image_in * p["text_in"]
            + text_out * p["text_out"] + image_out * p["image_out"]) / 1_000_000


def fallback_estimates():
    """Geschätzte Kosten pro Bild (USD) je Modell/Qualität für 1024x1024."""
    out = {}
    for model, by_q in EXPECTED_TOKENS.items():
        p = PRICES_PER_MILLION.get(model)
        if not p:
            continue
        out[model] = {q: (200 * p["text_in"] + t_out * p["text_out"] + img * p["image_out"]) / 1_000_000
                      for q, (img, t_out) in by_q.items()}
    return out


def _request(api_key, base_url, path, body=None, timeout=60):
    req = urllib.request.Request(
        base_url.rstrip("/") + path,
        data=None if body is None else json.dumps(body).encode(),
        headers={"Content-Type": "application/json", "Authorization": f"Bearer {api_key}"},
        method="GET" if body is None else "POST",
    )
    return urllib.request.urlopen(req, timeout=timeout)


def _http_error_text(e):
    msg = e.read().decode("utf-8", "replace")
    try:
        msg = json.loads(msg)["error"]["message"]
    except Exception:
        pass
    return f"HTTP {e.code}: {msg[:400]}"


def list_image_models(api_key, base_url):
    """Alle Modelle des Kontos, die nach Bildmodell aussehen."""
    try:
        with _request(api_key, base_url, "/models", timeout=20) as resp:
            data = json.loads(resp.read())
    except urllib.error.HTTPError as e:
        raise RuntimeError(_http_error_text(e))
    ids = sorted(m["id"] for m in data.get("data", []))
    return [i for i in ids if "image" in i or "dall-e" in i]


def generate(api_key, base_url, model, quality, size, prompt, timeout=300):
    """Erzeugt ein Bild per Streaming. Gibt dict mit image (bytes), final_s, usage, error zurück."""
    body = {
        "model": model, "prompt": prompt, "n": 1, "size": size, "quality": quality,
        "output_format": "png", "stream": True, "partial_images": 0,
    }
    result = {"final_s": None, "usage": None, "image": None, "request_id": None, "error": None}
    start = time.perf_counter()
    try:
        with _request(api_key, base_url, "/images/generations", body, timeout) as resp:
            result["request_id"] = resp.headers.get("x-request-id")
            for raw in resp:
                line = raw.decode("utf-8", "replace").strip()
                if not line.startswith("data: "):
                    continue
                data = line[6:]
                if data == "[DONE]":
                    break
                try:
                    event = json.loads(data)
                except json.JSONDecodeError:
                    continue
                etype = event.get("type", "")
                if etype == "error" or etype.endswith(".failed"):
                    result["error"] = json.dumps(event.get("error", event), ensure_ascii=False)[:400]
                    break
                if etype == "image_generation.completed" and event.get("b64_json"):
                    result["final_s"] = round(time.perf_counter() - start, 2)
                    result["image"] = base64.b64decode(event["b64_json"])
                    result["usage"] = event.get("usage")
                    break
    except urllib.error.HTTPError as e:
        result["error"] = _http_error_text(e)
    except (TimeoutError, socket.timeout) as e:
        result["error"] = f"zu langsam – nach {timeout} s abgebrochen"
        result["timeout"] = True
    except urllib.error.URLError as e:
        if isinstance(e.reason, (TimeoutError, socket.timeout)):
            result["error"] = f"zu langsam – nach {timeout} s abgebrochen"
            result["timeout"] = True
        else:
            result["error"] = f"Netzwerkfehler: {e.reason}"
    except Exception as e:
        result["error"] = f"{type(e).__name__}: {e}"
    if result["image"] is None and result["error"] is None:
        result["error"] = "Stream endete ohne fertiges Bild"
    return result

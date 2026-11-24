# KI-Benchmark

Kleine lokale Webseite, um OpenAI-Modelle für ChatGameLab nach **Qualität,
Geschwindigkeit und Kosten** zu vergleichen. Zurzeit für Bilder; Texte und
Audio sind als Reiter vorbereitet.

Braucht nur Python 3 – keine Zusatzpakete.

## Starten

Doppelklick auf `start.command` (Mac) oder im Terminal:

```bash
python3 scripts/ai-benchmark/server.py
```

Dann öffnet sich http://127.0.0.1:8777. Beenden mit Ctrl+C im Terminal.

Beim ersten Start fragt die Seite nach dem OpenAI-API-Schlüssel. Er wird nur
lokal benutzt und auf Wunsch im Mac-Schlüsselbund gespeichert
(`chatgamelab-ai-benchmark`). Alternativ: Umgebungsvariable `OPENAI_API_KEY`.

## Was die Seite kann

- **Neuer Test:** Modelle × Qualitätsstufen (low/medium/high) mit demselben
  Prompt, Bildgröße, Anzahl Läufe, Zeitlimit (Standard 45 s). Vorher gibt es
  eine Kostenschätzung aus bisherigen Läufen.
- **Zeitlimit:** Was länger braucht, wird abgebrochen. Varianten, die bisher im
  Mittel über dem Limit lagen, werden gar nicht erst versucht; die nächsthöhere
  Stufe wird übersprungen, wenn die Stufe darunter schon über ⅓ des Limits lag.
- **Läufe:** jeder Lauf ist ein eigener Eintrag (← → blättert). Pro Bild:
  Dauer, Tokens, Kosten pro Bild in Cent und pro 1000 Bilder in Dollar.
- **Eindruck:** jedes Bild lässt sich von 0 (unterirdisch) bis 10
  (überragend) bewerten; in der Großansicht mit den Tasten 0–9.
- **Alle Läufe zusammen:** Tabelle mit Median-Zeit, Ø Eindruck, Ø Kosten je
  Modell/Stufe, darunter alle Bilder einer Variante nebeneinander.
- **Weitere Modelle bei OpenAI suchen:** zeigt alle Bildmodelle des Kontos.

Modelle stehen vom ältesten zum neuesten (`MODEL_ORDER` in `bench_image.py`).

## Kosten

Die Kosten sind **berechnet** (von OpenAI gemeldete Tokens × Preisliste in
`bench_image.py`), keine echte Abrechnung. Die Preise für gpt-image-2, 1.5 und
1-mini sind mit der Abrechnung abgeglichen (24.11.2026); die Flare-Preise sind
vorläufig. Echte Beträge: OpenAI-Dashboard → Ausgaben → Gruppieren nach
„Einzelposten“ (nur pro Tag; neue Modelle lassen sich so eindeutig zuordnen).

## Dateien

| Datei | Inhalt |
|---|---|
| `server.py` | lokaler Webserver, Warteschlange für Läufe, Speichern, Bewertungen |
| `bench_image.py` | OpenAI-Bild-Aufruf (Streaming), Preise, Kostenberechnung, Standard-Prompt |
| `static/index.html` | die Seite (HTML, CSS, JavaScript in einer Datei) |
| `start.command` | Doppelklick-Start für den Mac |
| `results/` | Läufe mit Bildern, `run.json` und `ratings.json` – nicht in Git |

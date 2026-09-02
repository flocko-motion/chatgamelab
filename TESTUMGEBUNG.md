# Testumgebung: Einrichten, Starten, Ändern

Diese Anleitung ist für alle, die lokal an ChatGameLab testen oder Änderungen
vorbereiten, die anschließend über eine Pull Request von Flo auf dem Dev-Server
geprüft und für Prod freigegeben werden. Sie ersetzt keine vollständige
Doku — Details stehen in [README.md](README.md).

## 1. Testumgebung einrichten (einmalig)

Voraussetzung: **Docker Desktop** ist installiert und geöffnet.

```bash
cd /Users/ulrich1000/Documents/GitHub/chatgamelab
cp .env.example .env
```

Danach `.env` öffnen und mindestens folgende Werte eintragen (Rest kann meist
auf den Vorgaben bleiben):

- `AUTH0_DOMAIN`, `AUTH0_AUDIENCE` — bei Flo erfragen, falls nicht vorhanden
- OpenAI-API-Key, falls Spiele getestet werden sollen

## 2. Testumgebung starten

```bash
./run-dev.sh
```

Läuft alles komplett in Docker — nichts muss lokal installiert werden.

Im Browser aufrufen:

```
http://localhost
```

Beenden mit `Ctrl+C` im Terminal oder `docker compose down`.

*(Für tieferes Debugging von Frontend/Backend einzeln: `./run-dev.sh frontend`
bzw. `./run-dev.sh backend` — für reines Testen reicht die einfache Variante
oben.)*

## 3. Änderungen für Flo bereitstellen

1. **Nie direkt auf `main`** arbeiten — das ist die Live-Seite.
2. Von `development` einen eigenen Branch abzweigen:
   ```bash
   git checkout development
   git pull
   git checkout -b fix/mein-thema
   ```
3. Änderungen machen, committen, pushen:
   ```bash
   git add -A
   git commit -m "kurze Beschreibung"
   git push -u origin fix/mein-thema
   ```
4. **Pull Request gegen `development`** erstellen (auf GitHub oder mit
   `gh pr create --base development`).
5. Flo bekommt die PR-Benachrichtigung, reviewt und merged sie. **Erst nach
   dem Merge** erscheint die Änderung auf `cgldev.fmnoel.de` (spiegelt den
   `development`-Branch).
6. Freigabe für Prod = separater Merge von `development` nach `main`
   (macht in der Regel Flo).

**Wichtig:** Solange eine PR offen ist, ist sie nur auf GitHub sichtbar — nicht
auf `cgldev.fmnoel.de`.

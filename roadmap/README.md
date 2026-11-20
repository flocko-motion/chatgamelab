# Roadmap

The project's roadmap, generated from the GitHub issues and never edited by hand:
**https://flocko-motion.github.io/chatgamelab/** (zoom, drag, filter by epic, search).

To change the roadmap, change the issues: put them under an epic (sub-issues), give them a milestone (the phases, ordered by due date), set *blocked by* relations, priority labels `P1`–`P3`, assignees.

## How it runs

- `roadmap.py` reads the issues through `gh` and prints the roadmap as JSON.
- `index.html` is the page: static, Cytoscape.js with the ELK layout, renders `roadmap.json` next to it.
- `.github/workflows/roadmap.yml` builds both on issue changes, daily and by hand, and deploys them to Pages. It runs from the default branch, where GitHub sends issue events.

## Commands

- `make roadmap`: rebuild it now, e.g. after a new *blocked by*, which sends no event. The page's *Update* button opens the same workflow.
- `make roadmap-local`: the page on this machine, from the live issues (your `gh` login), on http://localhost:8765.

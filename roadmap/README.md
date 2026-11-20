# Roadmap

The project's roadmap, generated from the GitHub issues and never edited by hand:

- [interactive page](https://flocko-motion.github.io/chatgamelab/) on GitHub Pages: zoom, drag, filter by epic, search;
- [wiki page](https://github.com/flocko-motion/chatgamelab/wiki/Roadmap): the same as text with Mermaid diagrams.

To change the roadmap, change the issues: put them under an epic (sub-issues), set *blocked by* relations, priority labels `P1`–`P3`, assignees.

## How it runs

- `roadmap.py` reads the issues through `gh` and prints the wiki page; with `ROADMAP_JSON=<path>` it also writes the data for the interactive page.
- `index.html` is the interactive page: static, Cytoscape.js with the ELK layout, renders `roadmap.json` next to it.
- `.github/workflows/roadmap.yml` runs both on issue changes, daily and by hand, pushes the wiki page and deploys the interactive page to Pages. It runs from the default branch, where GitHub sends issue events.

## Commands

- `make roadmap`: rebuild both now, e.g. after a new *blocked by*, which sends no event.
- `make roadmap-local`: the interactive page on this machine, from the live issues (your `gh` login), on http://localhost:8765.

# Roadmap

The project's roadmap, generated from the GitHub issues and never edited by hand:
**https://flocko-motion.github.io/chatgamelab/roadmap/** (zoom, drag, show only one person's issues or the unassigned ones (linkable as `?who=login`, `?who=-`), search the titles (several hits open a list), click for details (in the side pane, until closed); *Epics only* for the overview: the same layout with the issues hidden, each epic box labelled, linkable as `#epics`).

An issue's card is filled in its assignees' colours (one per person, a gradient for several, white for none); a red outline means blocked. The page is always light, in GitHub's style.

Open issues under no epic show too, in their milestone, in a *Without epic* box right of the epics. The side pane opens only on demand: for what is clicked, or the legend under *?*. When epics are still open whose every issue is closed, a button in the top bar (*N to close*) lists them.

To change the roadmap, change the issues: an epic is an issue titled `Epic: …`, shown even before it has issues (then in its own milestone); put issues under an epic (sub-issues), give them a milestone (the phases, ordered by due date), set *blocked by* relations, priority labels `P1`–`P3`, assignees.

## How it runs

- `roadmap.py` reads the issues through `gh` and prints the roadmap as JSON.
- `serve.py` serves the page on this machine (`make roadmap-local`) and reloads its data from the page: *Update* in the top bar reads every issue fresh, *Update* in the side pane only what concerns that issue, in seconds. GitHub's answers are cached between reloads (`roadmap.py --cache`).
- `index.html` is the page: static, Cytoscape.js with the ELK layout, renders `roadmap.json` next to it.
- `.github/workflows/roadmap.yml` builds both on issue changes, daily, by hand and on pushes that change them, and deploys them to Pages under `/roadmap/`, beside the homepage (`homepage/`) at the root. It runs from the default branch, where GitHub sends issue events.

## Commands

- `make roadmap`: rebuild it now, e.g. after a new *blocked by*, which sends no event. The page's *Update* button opens the same workflow.
- `make roadmap-local`: the page on this machine, on http://localhost:8765, from the live issues (your `gh` login), with the reload buttons. When issues from an earlier run are there, it asks whether to read them all fresh (a few seconds); Enter keeps them.

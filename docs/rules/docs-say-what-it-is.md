---
id: docs-say-what-it-is
applies_to:
  - "docs/**/*.typ"
  - "docs/**/*.md"
  - "**/ARCHITECTURE.md"
  - "AGENTS.md"
  - "CONTRIBUTING.md"
  - "README.md"
trigger: "writes or changes prose that describes the product, the code or how to work on it"
---

# Docs say what things are

A doc describes what something is, in the present tense. It is no diary: what
used to be, what was decided against and what changed live in git and the
issues.

```text
Bad:  There is no shared pipeline anymore; each genre now has its own.
Bad:  An earlier draft had guests leave workshops; this was dropped.
Good: Each genre has its own pipeline.
Good: A guest's place is fixed: a guest enters and leaves no workshop.
```

- **Positive statements**: "each genre has its own pipeline", not "there is
  no shared pipeline".
- **No history**: no "used to", "now", "no longer", "earlier", "since #123".
- **The spec stays conceptual** (`docs/chatgamelab.typ`): no file paths,
  endpoint lists, make targets or prices; those live in code and its docs.
  It describes the target state, built or not; `#todo[…]` marks only open
  decisions.

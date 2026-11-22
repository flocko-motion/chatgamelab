---
id: phase-refs
applies_to:
  - "**/*.md"
  - "**/*.typ"
  - "roadmap/**"
trigger: "names a numbered phase or step of a plan, such as 6e"
---

# A phase is named with its issue

A numbered phase is always named by its number and its GitHub issue together:
the number shows the order, the issue is what gets tracked and handed on.

```text
Bad:  6e moves the web app into schemaf.
Bad:  #349 moves the web app into schemaf.
Good: 6e (#349) moves the web app into schemaf.
```

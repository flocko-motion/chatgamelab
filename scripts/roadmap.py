#!/usr/bin/env python3
"""Renders the repo's roadmap as Markdown with Mermaid, for the wiki page Roadmap.

Phases are the open epics (issues with sub-issues); the order comes from
"blocked by" relations, priority from the labels P1/P2/P3. Everything is read
live through gh; the page is never edited by hand.

Usage: scripts/roadmap.py [owner/repo] > Roadmap.md
"""
import json
import subprocess
import sys
from collections import defaultdict

REPO = sys.argv[1] if len(sys.argv) > 1 else "flocko-motion/chatgamelab"

# Where the roadmap workflow deploys the interactive page.
PAGES_URL = f"https://{REPO.split('/')[0]}.github.io/{REPO.split('/')[1]}/"

# Done cards per row at the top of an epic; Mermaid does not wrap a row.
DONE_PER_ROW = 4


def gh(path):
    out = subprocess.run(["gh", "api", "--paginate", f"repos/{REPO}/{path}"],
                         check=True, capture_output=True, text=True).stdout
    # --paginate concatenates JSON arrays; normalise.
    return json.loads(out.replace("][", ","))


def priority(issue):
    names = {l["name"] for l in issue.get("labels", [])}
    for p in ("P1", "P2", "P3"):
        if p in names:
            return int(p[1])
    return 4


# Every open issue; the epics among them are those with sub-issues, oldest first.
OPEN = [i for i in gh("issues?state=open&per_page=100") if "pull_request" not in i]
EPICS = sorted(i["number"] for i in OPEN if (i.get("sub_issues_summary") or {}).get("total", 0) > 0)

_deps = {}


def deps(n):
    """The issues n is blocked by, cached."""
    if n not in _deps:
        _deps[n] = gh(f"issues/{n}/dependencies/blocked_by")
    return _deps[n]


def epic_blockers(e, seen=None):
    """The open epics epic e waits for. A closed issue in between drops out
    but passes its own blockers on, as for issues."""
    seen = set() if seen is None else seen
    found = set()
    for b in deps(e):
        n = b["number"]
        if n in seen or not b.get("repository_url", "").endswith(REPO):
            continue
        seen.add(n)
        if n in EPICS:
            found.add(n)
        elif b["state"] != "open":
            found |= epic_blockers(n, seen)
    return found


# Epics in dependency order: a blocked epic after the epics it waits for.
waits_on = {e: epic_blockers(e) for e in EPICS}
ordered, left = [], {e: set(bs) for e, bs in waits_on.items()}
while left:
    free = sorted(e for e, bs in left.items() if not bs) or sorted(left)
    e = free[0]
    ordered.append(e)
    del left[e]
    for bs in left.values():
        bs.discard(e)
EPICS = ordered

issues = {}      # number -> issue
phase_of = {}    # number -> epic number
phases = []      # (epic, [numbers])
for e in EPICS:
    epic = gh(f"issues/{e}")
    subs = gh(f"issues/{e}/sub_issues")
    nums = []
    for s in subs:
        if s.get("repository_url", "").endswith(REPO):
            issues[s["number"]] = s
            phase_of[s["number"]] = e
            nums.append(s["number"])
    phases.append((epic, nums))

blockers = defaultdict(set)  # number -> open-or-closed blockers within the set
for n in issues:
    for b in deps(n):
        if b["number"] in issues:
            blockers[n].add(b["number"])


def is_open(n):
    return issues[n]["state"] == "open"


def open_blockers(n, seen=None):
    """The open issues n waits for. Done issues drop out but pass their own
    blockers on: with A -> B (done) -> C, C waits for A."""
    seen = set() if seen is None else seen
    found = set()
    for b in blockers[n]:
        if b in seen:
            continue
        seen.add(b)
        if is_open(b):
            found.add(b)
        else:
            found |= open_blockers(b, seen)
    return found


def epic_waits(n):
    """The open issues of the epics n's epic waits for: none of n starts
    before those epics are done."""
    found = set()
    for e in waits_on.get(phase_of.get(n), ()):
        found |= {m for m in issues if phase_of.get(m) == e and is_open(m)}
    return found


def waits(n):
    return open_blockers(n) | epic_waits(n)


def state(n):
    if not is_open(n):
        return "done"
    if waits(n):
        return "blocked"
    return "ready"


phase_index = {e["number"]: i for i, (e, _) in enumerate(phases)}


def tiebreak(n):
    return (priority(issues[n]), phase_index[phase_of[n]], n)


# Order: Kahn over open issues; a blocked issue always after its blockers.
open_nums = [n for n in issues if is_open(n)]
pending = {n: set(waits(n)) for n in open_nums}
order = []
while pending:
    free = sorted((n for n, bs in pending.items() if not bs), key=tiebreak)
    if not free:  # a cycle: list the rest as they come
        order += sorted(pending, key=tiebreak)
        break
    n = free[0]
    order.append(n)
    del pending[n]
    for bs in pending.values():
        bs.discard(n)

url = lambda n: issues[n]["html_url"]
people = lambda n: [u["login"] for u in issues[n].get("assignees") or []]
tags = lambda n: [l["name"] for l in issues[n].get("labels") or []]


def meta(n, sep=" · "):
    parts = []
    if people(n):
        parts.append("👤 " + ", ".join(people(n)))
    if tags(n):
        parts.append("🏷 " + ", ".join(tags(n)))
    return sep.join(parts)

title = lambda n: issues[n]["title"]
P = lambda n: f"P{priority(issues[n])}" if priority(issues[n]) < 4 else ""

lines = [
    "<!-- GENERATED: do not edit. Written by .github/workflows/roadmap.yml from the issues; any edit here is overwritten. -->",
    "",
    "> [!WARNING]",
    "> **Generated automatically — do not edit this page.** It is rebuilt from the issues after every change, and edits here are overwritten.",
    "> To change the roadmap, change the issues: their epic, priority label (`P1`–`P3`) and *blocked by* relations.",
    "",
    "# Roadmap",
    "",
    f"**[Interactive version]({PAGES_URL})**: zoom, drag, filter by epic, search.",
    "",
]

lines += ["## Next up", ""]
ready = sorted((n for n in open_nums if state(n) == "ready"), key=tiebreak)
for n in ready[:5]:
    lines.append(f"- [#{n}]({url(n)}) {title(n)}" + (f" — {meta(n)}" if meta(n) else ""))
lines.append("")

lines += ["## Phases", ""]
for epic, nums in phases:
    done = sum(1 for n in nums if not is_open(n))
    total = len(nums) or 1
    bar = "▓" * round(10 * done / total) + "░" * (10 - round(10 * done / total))
    lines.append(f"- `{bar} {done:>2}/{len(nums):<2}` **[#{epic['number']} {epic['title']}]({epic['html_url']})**")
lines.append("")

lines += ["## Order", "", "Top to bottom: what comes first. No scale, only order.", "", "```mermaid", "flowchart TB"]
for epic, nums in phases:
    done = sum(1 for n in nums if not is_open(n))
    etitle = epic["title"].replace('"', "'")
    elink = f"<a href='{epic['html_url']}' style='text-decoration:none;color:inherit'><b>#{epic['number']}</b></a>"
    lines.append(f'  subgraph E{epic["number"]}["{elink} {etitle} — {done}/{len(nums)}"]')
    # Done issues come first, one card each, in the order they closed.
    closed = sorted((n for n in nums if not is_open(n)), key=lambda n: issues[n].get("closed_at") or "")
    opened = [n for n in nums if is_open(n)]
    for n in closed + opened:
        label = title(n).replace('"', "'")
        extra = meta(n, "<br/>").replace('"', "'")
        link = f"<a href='{url(n)}' style='text-decoration:none;color:inherit'><b>#{n}</b></a>"
        text = f"{link} {label}" + (f"<br/><small>{extra}</small>" if extra else "")
        lines.append(f'    I{n}["{text}"]:::{state(n)}')
    # Invisible edges lay the done cards out as a grid, DONE_PER_ROW wide, each
    # card above the one in its column on the next row, and the last row above
    # the open issues that start the epic.
    roots = [n for n in opened if not any(phase_of.get(b) == epic["number"] for b in open_blockers(n))]
    for i, d in enumerate(closed):
        below = i + DONE_PER_ROW
        if below < len(closed):
            lines.append(f"    I{d} ~~~ I{closed[below]}")
        else:
            for n in roots:
                lines.append(f"    I{d} ~~~ I{n}")
    lines.append("  end")
for n in issues:
    if not is_open(n):
        continue
    for b in sorted(open_blockers(n)):
        lines.append(f"  I{b} --> I{n}")
for e in EPICS:
    for b in sorted(waits_on[e]):
        lines.append(f"  E{b} --> E{e}")
lines += [
    "  classDef done fill:#e6e6e6,color:#8a8a8a,stroke:#c8c8c8",
    "  classDef ready fill:#d8f5d8,color:#1f3d1f,stroke:#4caf50",
    "  classDef blocked fill:#fbe3e3,color:#5c2a2a,stroke:#d98c8c",
    "```",
    "",
    "### How to read it",
    "",
    "- **Top to bottom is the order:** what has to happen first is higher up. There is no time scale; nothing is said about how long a step takes.",
    "- **An arrow** runs from an issue to one that waits for it (*blocked by* on GitHub). An issue starts only once everything pointing at it is done. Done issues leave the graph but keep their chain: if A blocks B, B is done and B blocks C, the arrow runs from A to C.",
    "- **A box** groups the issues of one epic; its title shows how many are done. An arrow between boxes means the lower epic waits for the upper one (*blocked by* between the epics): its issues count as blocked until that epic is done.",
    "- **Each card** shows the issue's number and title, then who is assigned (👤) and its labels (🏷).",
    "",
    "| Colour | Means |",
    "|---|---|",
    "| 🟩 green | **ready**: open, and nothing it waits for is still open; can be started now |",
    "| 🟥 pale red | **blocked**: open, but waiting for at least one open issue |",
    "| ⬜ grey | **done**: closed; done issues sit in a row at the top of their epic, in the order they closed |",
    "",
]

lines += ["## Order as a list", ""]
for i, n in enumerate(order, 1):
    wait = sorted(open_blockers(n))
    ewait = sorted(waits_on.get(phase_of.get(n), ()))
    parts = [f"#{b}" for b in wait] + [f"epic #{e}" for e in ewait]
    after = f" — after {', '.join(parts)}" if parts else ""
    lines.append(f"{i}. [#{n}]({url(n)}) {title(n)}{after}" + (f" — {meta(n)}" if meta(n) else ""))
lines.append("")

# Backlog: open issues outside every open epic, the epics themselves aside.
backlog = [i for i in OPEN if i["number"] not in issues and i["number"] not in EPICS]
backlog.sort(key=lambda i: (priority(i), i["number"]))
lines += [f"## Backlog ({len(backlog)})", "",
          "Open issues in no open epic: to sort into an epic, prioritise or close.", ""]
for i in backlog:
    issues[i["number"]] = i
    n = i["number"]
    lines.append(f"- [#{n}]({url(n)}) {title(n)}" + (f" — {meta(n)}" if meta(n) else ""))
lines.append("")

# The same roadmap as data, for the interactive page (ROADMAP_JSON=path).
JSON_OUT = __import__("os").environ.get("ROADMAP_JSON")
if JSON_OUT:
    from datetime import datetime, timezone

    def item(n):
        i = issues[n]
        return {
            "number": n,
            "title": i["title"],
            "url": i["html_url"],
            "state": state(n) if n in phase_of else ("done" if i["state"] != "open" else "backlog"),
            "epic": phase_of.get(n),
            "assignees": people(n),
            "labels": tags(n),
            "priority": priority(i) if priority(i) < 4 else None,
            "closedAt": i.get("closed_at"),
            "blockedBy": sorted(open_blockers(n)) if n in phase_of and is_open(n) else [],
        }

    data = {
        "repo": REPO,
        "generated": datetime.now(timezone.utc).isoformat(timespec="seconds"),
        "epics": [{
            "number": epic["number"],
            "title": epic["title"],
            "url": epic["html_url"],
            "done": sum(1 for n in nums if not is_open(n)),
            "total": len(nums),
            "waitsOn": sorted(waits_on[epic["number"]]),
            "issues": nums,
        } for epic, nums in phases],
        "issues": [item(n) for n in phase_of],
        "order": order,
        "next": ready[:5],
        "backlog": [item(i["number"]) for i in backlog],
    }
    with open(JSON_OUT, "w") as f:
        json.dump(data, f, indent=1)

print("\n".join(lines))

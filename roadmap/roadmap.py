#!/usr/bin/env python3
"""Renders the repo's roadmap as Markdown with Mermaid, for the wiki page Roadmap.

Phases are the open epics (issues with sub-issues); the order comes from
"blocked by" relations, priority from the labels P1/P2/P3. Everything is read
live through gh; the page is never edited by hand.

Usage: roadmap/roadmap.py [owner/repo] > Roadmap.md
       ROADMAP_JSON=roadmap.json roadmap/roadmap.py > /dev/null
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


issues = {}      # number -> issue
phase_of = {}    # number -> epic number
subs_of = {}     # epic -> its sub-issues in this repo
for e in EPICS:
    subs_of[e] = [x for x in gh(f"issues/{e}/sub_issues") if x.get("repository_url", "").endswith(REPO)]
    for x in subs_of[e]:
        issues[x["number"]] = x
        phase_of[x["number"]] = e

_open_by_number = {i["number"]: i for i in OPEN}

# Milestones (GitHub's milestone field, set per issue) group the roadmap into
# phases, ordered by due date, then title. They order and group; nothing waits
# because of a milestone, only by blocked-by.
MILESTONES = sorted(gh("milestones?state=open&per_page=100"),
                    key=lambda m: (m.get("due_on") or "9999", m["title"]))
ms_order = {m["number"]: k for k, m in enumerate(MILESTONES)}
epic_ms = {}  # epic -> the earliest phase it has issues in, for ordering
for e in EPICS:
    ks = [ms_order.get((x.get("milestone") or {}).get("number"), len(MILESTONES)) for x in subs_of[e]]
    epic_ms[e] = min(ks) if ks else len(MILESTONES)


def milestone_of(n):
    m = issues[n].get("milestone")
    return m["number"] if m and m["number"] in ms_order else None


def epic_blockers(e, seen=None):
    """The open epics epic e waits for by blocked-by. A closed issue in
    between drops out but passes its own blockers on, as for issues."""
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


# Epics in order: by phase, and within a phase after the epics they wait for.
waits_on = {e: epic_blockers(e) for e in EPICS}
ordered, left = [], {e: set(bs) for e, bs in waits_on.items()}
while left:
    free = sorted((e for e, bs in left.items() if not bs), key=lambda e: (epic_ms[e], e)) \
        or sorted(left, key=lambda e: (epic_ms[e], e))
    e = free[0]
    ordered.append(e)
    del left[e]
    for bs in left.values():
        bs.discard(e)
EPICS = ordered
phases = [(gh(f"issues/{e}"), [x["number"] for x in subs_of[e]]) for e in EPICS]


def upstream(g, seen=None):
    """Every epic g waits for by blocked-by, directly or through others."""
    seen = set() if seen is None else seen
    for b in waits_on.get(g, set()):
        if b not in seen:
            seen.add(b)
            upstream(b, seen)
    return seen


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
    """The open issues of every epic n's epic waits for, directly or through
    epics and milestones: none of n starts before those are done."""
    return gate_work(phase_of.get(n))


def gate_work(g):
    """The open issues of every epic g waits for, directly or through others."""
    up = upstream(g)
    return {m for m in issues if phase_of.get(m) in up and is_open(m)}


def waits(n):
    return open_blockers(n) | epic_waits(n)


def state(n):
    if not is_open(n):
        return "done"
    if waits(n):
        return "blocked"
    return "ready"


def gate_state(g):
    """An epic is done when all its issues are, blocked while something it
    waits for is open, else ready."""
    if all(not is_open(m) for m in issues if phase_of.get(m) == g):
        return "done"
    return "blocked" if gate_work(g) else "ready"


phase_index = {e["number"]: i for i, (e, _) in enumerate(phases)}
issues_epic_url = {e["number"]: e["html_url"] for e, _ in phases}
gate_word = lambda g: "epic"


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

lines += ["## Phases", "", "The milestones, in order, with the epics that have issues in them. A milestone groups and orders; nothing waits because of it.", ""]


def bar(done, total):
    total = total or 1
    return "▓" * round(10 * done / total) + "░" * (10 - round(10 * done / total))


for k, m in enumerate(MILESTONES + [None]):
    key = m["number"] if m else None
    members = []
    for epic, nums in phases:
        mine = [n for n in nums if milestone_of(n) == key]
        if mine:
            members.append((epic, mine))
    if not members:
        continue
    if m:
        done, total = m["closed_issues"], m["open_issues"] + m["closed_issues"]
        due = f" · due {m['due_on'][:10]}" if m.get("due_on") else " · no due date"
        lines.append(f"**◆ [{m['title']}]({m['html_url']})** `{bar(done, total)} {done}/{total}`{due}")
    else:
        lines.append("**No milestone**")
    lines.append("")
    for epic, mine in members:
        done = sum(1 for n in mine if not is_open(n))
        lines.append(f"- `{bar(done, len(mine))} {done:>2}/{len(mine):<2}` **[#{epic['number']} {epic['title']}]({epic['html_url']})**")
    lines.append("")

LINK = "style='text-decoration:none;color:inherit'"
CLASSES = [
    "  classDef done fill:#e6e6e6,color:#8a8a8a,stroke:#c8c8c8",
    "  classDef ready fill:#d8f5d8,color:#1f3d1f,stroke:#4caf50",
    "  classDef blocked fill:#fbe3e3,color:#5c2a2a,stroke:#d98c8c",
    "  classDef ext fill:#ffffff,color:#666666,stroke:#999999,stroke-dasharray:4 3",
]


def card(n):
    label = title(n).replace('"', "'")
    extra = meta(n, "<br/>").replace('"', "'")
    link = f"<a href='{url(n)}' {LINK}><b>#{n}</b></a>"
    return f"{link} {label}" + (f"<br/><small>{extra}</small>" if extra else "")


# The overview: one card per epic, arrows between epics.
lines += ["## Order", "",
          "Top to bottom: what comes first. No scale, only order. First the epics, then each epic on its own.", "",
          "```mermaid", "flowchart TB"]
epic_by_number = {e["number"]: (e, nums) for e, nums in phases}
pieces = {}  # epic -> its piece ids, in milestone order
for k, m in enumerate(MILESTONES + [None]):
    key = m["number"] if m else None
    members = [(g, [n for n in epic_by_number[g][1] if milestone_of(n) == key]) for g in EPICS]
    members = [(g, mine) for g, mine in members if mine]
    if not members:
        continue
    label = f"◆ {m['title']}".replace('"', "'") if m else "No milestone"
    lines.append(f'  subgraph M{k}["{label}"]')
    for g, mine in members:
        epic = epic_by_number[g][0]
        done = sum(1 for n in mine if not is_open(n))
        state_ = "done" if done == len(mine) else ("blocked" if any(state(n) == "blocked" for n in mine if is_open(n)) and not any(state(n) == "ready" for n in mine) else "ready")
        etitle = epic["title"].replace('"', "'")
        pid = f"E{g}_{k}"
        pieces.setdefault(g, []).append(pid)
        lines.append(f'    {pid}["<a href=\'{epic["html_url"]}\' {LINK}><b>#{g}</b></a> {etitle}<br/><small>{done}/{len(mine)} done</small>"]:::{state_}')
    lines.append("  end")
for g, ps in pieces.items():
    for a, b in zip(ps, ps[1:]):
        lines.append(f"  {a} === {b}")
for g in EPICS:
    for b in sorted(waits_on[g]):
        if b in pieces and g in pieces:
            lines.append(f"  {pieces[b][-1]} --> {pieces[g][0]}")
lines += CLASSES + ["```", ""]

# Each epic on its own: done cards on top, then its open issues; blockers from
# other epics appear as dashed cards.
for epic, nums in phases:
    e = epic["number"]
    done = sum(1 for n in nums if not is_open(n))
    lines += [f"### [#{e}]({epic['html_url']}) {epic['title']} — {done}/{len(nums)}", ""]
    ewait = sorted(waits_on[e])
    if ewait:
        lines += ["Waits for " + ", ".join(f"epic [#{b}]({issues_epic_url[b]})" for b in ewait) + ".", ""]

    lines += ["```mermaid", "flowchart TB"]
    closed = sorted((n for n in nums if not is_open(n)), key=lambda n: issues[n].get("closed_at") or "")
    opened = [n for n in nums if is_open(n)]
    for n in closed + opened:
        lines.append(f'  I{n}["{card(n)}"]:::{state(n)}')
    external = sorted({b for n in opened for b in open_blockers(n) if phase_of.get(b) != e})
    for b in external:
        where = f" in #{phase_of[b]}" if b in phase_of else ""
        lines.append(f'  I{b}["<a href=\'{url(b)}\' {LINK}><b>#{b}</b></a>{where}"]:::ext')
    # Invisible edges lay the done cards out as a grid, DONE_PER_ROW wide,
    # above the open issues that start the epic.
    roots = [n for n in opened if not any(phase_of.get(b) == e for b in open_blockers(n))]
    for i, d in enumerate(closed):
        below = i + DONE_PER_ROW
        if below < len(closed):
            lines.append(f"  I{d} ~~~ I{closed[below]}")
        else:
            for n in roots:
                lines.append(f"  I{d} ~~~ I{n}")
    for n in opened:
        for b in sorted(open_blockers(n)):
            lines.append(f"  I{b} --> I{n}")
    lines += CLASSES + ["```", ""]

lines += [
    "### How to read it",
    "",
    "- **Top to bottom is the order:** what has to happen first is higher up. There is no time scale; nothing is said about how long a step takes.",
    "- **The overview** groups the epics by milestone, in order. An epic with issues in several milestones appears once per milestone, its pieces joined by a thick line. An arrow between epics means the lower one waits for the upper one (*blocked by*); its issues count as blocked until then.",
    "- **Each epic's diagram** shows its issues. An arrow runs from an issue to one that waits for it (*blocked by*). Done issues leave the graph but keep their chain: if A blocks B, B is done and B blocks C, the arrow runs from A to C. A dashed card is a blocker from another epic.",
    "- **Each card** shows the issue's number (a link) and title, then who is assigned (👤) and its labels (🏷).",
    "",
    "| Colour | Means |",
    "|---|---|",
    "| 🟩 green | **ready**: open, and nothing it waits for is still open; can be started now |",
    "| 🟥 pale red | **blocked**: open, but waiting for at least one open issue or epic |",
    "| ⬜ grey | **done**: closed; done issues sit at the top of their epic, in the order they closed |",
    "| ⬚ dashed | a blocker from another epic |",
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
            "milestone": milestone_of(n) if n in phase_of else None,
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
            "state": gate_state(epic["number"]),
            "issues": nums,
        } for epic, nums in phases],
        "issues": [item(n) for n in phase_of],
        "milestones": [{
            "number": m["number"],
            "title": m["title"],
            "url": m["html_url"],
            "due": m.get("due_on"),
            "open": m["open_issues"],
            "closed": m["closed_issues"],
        } for k, m in enumerate(MILESTONES)],
        "order": order,
        "next": ready[:5],
        "backlog": [item(i["number"]) for i in backlog],
    }
    with open(JSON_OUT, "w") as f:
        json.dump(data, f, indent=1)

print("\n".join(lines))

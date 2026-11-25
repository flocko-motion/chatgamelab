#!/usr/bin/env python3
"""Reads the repo's roadmap from the issues and prints it as JSON, for the
interactive page (roadmap/index.html).

Epics are issues with sub-issues; milestones (GitHub's field) group them into
phases; the order comes from "blocked by" relations, priority from the labels
P1/P2/P3. Everything is read live through gh.

Usage: roadmap/roadmap.py [owner/repo] [--cache DIR [--fresh | --drop N ...]] > roadmap.json

With --cache, every answer from GitHub is kept in DIR and read from there
the next time; --fresh empties it first, --drop N forgets what concerns
issue N (roadmap/serve.py uses these for its reload buttons).
"""
import argparse
import glob
import json
import os
import re
import shutil
import subprocess
import sys
import time
from collections import defaultdict
from concurrent.futures import ThreadPoolExecutor

_args = argparse.ArgumentParser()
_args.add_argument("repo", nargs="?", default="flocko-motion/chatgamelab")
_args.add_argument("--cache", help="keep GitHub's answers in this directory")
_args.add_argument("--fresh", action="store_true", help="empty the cache first")
_args.add_argument("--drop", type=int, action="append", default=[], help="forget what concerns this issue")
ARGS = _args.parse_args()
REPO = ARGS.repo
CACHE = ARGS.cache


def fetch(path, tries=4):
    """GitHub's answer for path. GitHub answers a burst of calls now and then
    with a 5xx or a rate limit; such a call is asked again, after 2, 4 and 8
    seconds. Any other refusal stops the run at once, with GitHub's words."""
    for attempt in range(tries):
        run = subprocess.run(["gh", "api", "--paginate", f"repos/{REPO}/{path}"],
                             capture_output=True, text=True)
        if run.returncode == 0:
            # --paginate concatenates JSON arrays; normalise.
            return json.loads(run.stdout.replace("][", ","))
        transient = re.search(r"HTTP (5\d\d|429)|rate limit|timeout|Something went wrong",
                              run.stderr, re.I)
        if not transient or attempt == tries - 1:
            sys.exit(f"gh api {path}: {run.stderr.strip()}")
        time.sleep(2 ** (attempt + 1))


def cached(path):
    return os.path.join(CACHE, re.sub(r"[^A-Za-z0-9]+", "_", path) + ".json")


_memo = {}


def gh(path):
    """GitHub's answer for path: from memory, else from the cache, else fetched."""
    if path in _memo:
        return _memo[path]
    f = cached(path) if CACHE else None
    if f and os.path.exists(f):
        with open(f) as fh:
            data = json.load(fh)
    else:
        data = fetch(path)
        if f:
            with open(f + ".tmp", "w") as fh:
                json.dump(data, fh)
            os.replace(f + ".tmp", f)
    _memo[path] = data
    return data


def prefetch(paths):
    """Reads many paths at once; each gh call is a process waiting on the
    network, so they run side by side. Later gh calls find them in memory."""
    with ThreadPoolExecutor(max_workers=12) as pool:
        list(pool.map(gh, [p for p in dict.fromkeys(paths) if p not in _memo]))


def forget(n):
    """Drops what concerns issue n from the cache, so the next read takes it
    fresh: the issue and its relations, the sub-issue lists of its epic before
    and now, the issue and milestone lists, and the blocked-by lists of the
    issues it blocks (as cached, and as GitHub has them now). For an epic the
    same holds for each of its sub-issues, those it has now and those it had:
    one that moved in is dropped from its old epic's list, one that moved out
    is picked up by its new epic's."""
    def cached_list(path):
        f = cached(path)
        if not os.path.exists(f):
            return []
        with open(f) as fh:
            return json.load(fh)

    old_subs = {x["number"] for x in cached_list(f"issues/{n}/sub_issues")}
    with ThreadPoolExecutor() as pool:
        issue, blocking, subs = pool.map(fetch, [f"issues/{n}", f"issues/{n}/dependencies/blocking",
                                                 f"issues/{n}/sub_issues"])
    new_subs = {x["number"] for x in subs if x.get("repository_url", "").endswith(REPO)}
    with ThreadPoolExecutor(max_workers=12) as pool:
        moved_out = list(pool.map(lambda m: fetch(f"issues/{m}"), sorted(old_subs - new_subs)))
    numbers = {n} | old_subs | new_subs
    paths = ["issues?state=open&per_page=100", "milestones?state=open&per_page=100", f"issues/{n}/sub_issues"]
    for k in numbers:
        paths += [f"issues/{k}", f"issues/{k}/dependencies/blocked_by"]
    for i in [issue] + moved_out:
        if i.get("parent_issue_url"):
            paths.append(f"issues/{i['parent_issue_url'].rsplit('/', 1)[1]}/sub_issues")
    paths += [f"issues/{b['number']}/dependencies/blocked_by" for b in blocking]
    files = {cached(p) for p in paths}
    # Every epic list holding one of them, and every blocked-by list naming n.
    for f in glob.glob(os.path.join(CACHE, "issues_*_sub_issues.json")):
        if any(x.get("number") in numbers for x in cached_list_file(f)):
            files.add(f)
    for f in glob.glob(os.path.join(CACHE, "issues_*_dependencies_blocked_by.json")):
        if any(x.get("number") == n for x in cached_list_file(f)):
            files.add(f)
    for f in files:
        if os.path.exists(f):
            os.remove(f)


def cached_list_file(f):
    with open(f) as fh:
        return json.load(fh)


if CACHE:
    if ARGS.fresh:
        shutil.rmtree(CACHE, ignore_errors=True)
    os.makedirs(CACHE, exist_ok=True)
    for n in ARGS.drop:
        forget(n)


def priority(issue):
    names = {l["name"] for l in issue.get("labels", [])}
    for p in ("P1", "P2", "P3"):
        if p in names:
            return int(p[1])
    return 4


# Every open issue; the epics among them are those with sub-issues, oldest first.
prefetch(["issues?state=open&per_page=100", "milestones?state=open&per_page=100"])
OPEN = [i for i in gh("issues?state=open&per_page=100") if "pull_request" not in i]
EPICS = sorted(i["number"] for i in OPEN if (i.get("sub_issues_summary") or {}).get("total", 0) > 0)

_deps = {}


def deps(n):
    """The issues n is blocked by, cached."""
    if n not in _deps:
        _deps[n] = gh(f"issues/{n}/dependencies/blocked_by")
    return _deps[n]


prefetch([p for e in EPICS for p in (f"issues/{e}/sub_issues", f"issues/{e}", f"issues/{e}/dependencies/blocked_by")])
issues = {}      # number -> issue
phase_of = {}    # number -> epic number
subs_of = {}     # epic -> its sub-issues in this repo
for e in EPICS:
    subs_of[e] = [x for x in gh(f"issues/{e}/sub_issues") if x.get("repository_url", "").endswith(REPO)]
    for x in subs_of[e]:
        issues[x["number"]] = x
        phase_of[x["number"]] = e

_open_by_number = {i["number"]: i for i in OPEN}
prefetch([f"issues/{n}/dependencies/blocked_by" for n in issues])

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

people = lambda n: [u["login"] for u in issues[n].get("assignees") or []]
tags = lambda n: [l["name"] for l in issues[n].get("labels") or []]

# Without epic: open issues outside every open epic, the epics themselves aside.
unfiled = [i for i in OPEN if i["number"] not in issues and i["number"] not in EPICS]
unfiled.sort(key=lambda i: (priority(i), i["number"]))
for i in unfiled:
    issues[i["number"]] = i
# The roadmap shows them too, in their milestone beside the epics, so their
# blocked-by counts like an epic issue's, in both directions.
prefetch([f"issues/{i['number']}/dependencies/blocked_by" for i in unfiled])
for n in issues:
    for b in deps(n):
        if b["number"] in issues:
            blockers[n].add(b["number"])


def unfiled_state(n):
    if not is_open(n):
        return "done"
    return "blocked" if open_blockers(n) else "ready"

# The roadmap as data, for the interactive page.
import re
from datetime import date, datetime, timezone


def summary(body, limit=600):
    """An epic's first prose paragraph, markdown stripped, for the page to show
    as many lines of as fit. Headings, tables, lists and code are skipped."""
    for para in re.split(r"\n\s*\n", body or ""):
        para = para.strip()
        if not para or para[0] in "#|-*>`<" or re.match(r"\d+\.\s", para):
            continue
        text = re.sub(r"\[([^\]]*)\]\([^)]*\)", r"\1", para)
        text = re.sub(r"[`*_]{1,2}([^`*_]+)[`*_]{1,2}", r"\1", text)
        text = " ".join(text.split())
        return text if len(text) <= limit else text[:limit].rsplit(" ", 1)[0] + "…"
    return ""


def item(n):
    i = issues[n]
    return {
        "number": n,
        "title": i["title"],
        "url": i["html_url"],
        "state": state(n) if n in phase_of else unfiled_state(n),
        "epic": phase_of.get(n),
        "assignees": people(n),
        "labels": tags(n),
        "priority": priority(i) if priority(i) < 4 else None,
        "closedAt": i.get("closed_at"),
        "blockedBy": sorted(open_blockers(n)) if is_open(n) else [],
        "milestone": milestone_of(n),
    }

data = {
    "repo": REPO,
    "generated": datetime.now(timezone.utc).isoformat(timespec="seconds"),
    "epics": [{
        "number": epic["number"],
        "title": epic["title"],
        "url": epic["html_url"],
        "description": summary(epic.get("body")),
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
    "unfiled": [item(i["number"]) for i in unfiled],
}
json.dump(data, sys.stdout, indent=1)
print()



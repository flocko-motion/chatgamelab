#!/usr/bin/env python3
"""Review a branch against the rules in docs/rules/, one rule at a time.

Each rule gets its own Claude agent that sees only that rule and only the
files it applies to. Every verdict must quote the lines it rests on; a verdict
whose quote is not in the material is rejected and asked again once. Each
violation then goes to a second agent that tries to disprove it. The report
lands in .reviews/ (gitignored), to read or to discuss with another agent.

    scripts/review.py                     # this branch against v2/development
    scripts/review.py feat/x              # another branch
    scripts/review.py --pr 488            # a pull request
    scripts/review.py --audit             # every file, not only the changes
    scripts/review.py --rule ui-text      # only some rules

The rules are read from the working tree, never from the reviewed branch: a
branch cannot weaken the rules it is checked against.
"""

from __future__ import annotations

import argparse
import concurrent.futures as futures
import datetime
import json
import os
import re
import subprocess
import sys
from dataclasses import dataclass, field
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
RULES_DIR = ROOT / "docs" / "rules"
OUT_DIR = ROOT / ".reviews"
DEFAULT_BASE = "origin/v2/development"

MAX_BATCH_CHARS = 80_000  # material per checking agent
CONTEXT_LINES = 12  # diff context around each change
FILE_WINDOW = 200  # lines of a file around a finding, for the disprover
CALL_TIMEOUT = 900  # seconds per agent


# ── rules ────────────────────────────────────────────────────────────────


@dataclass
class Rule:
    id: str
    applies_to: list[str]
    excludes: list[str]
    trigger: str
    context: list[str]  # files every agent of this rule reads, e.g. the spec
    text: str  # the whole file, frontmatter included

    def matches(self, path: str) -> bool:
        return any(glob_match(g, path) for g in self.applies_to) and not any(
            glob_match(g, path) for g in self.excludes
        )


def parse_frontmatter(text: str) -> dict:
    """The small YAML subset the rules use: scalars and lists of strings."""
    m = re.match(r"^---\n(.*?)\n---\n", text, re.S)
    if not m:
        return {}
    meta: dict = {}
    key = None
    for line in m.group(1).splitlines():
        if not line.strip():
            continue
        item = re.match(r"^\s+-\s+(.*)$", line)
        if item and key:
            meta.setdefault(key, []).append(unquote(item.group(1)))
            continue
        kv = re.match(r"^([\w-]+):\s*(.*)$", line)
        if kv:
            key, value = kv.group(1), kv.group(2).strip()
            if value.startswith("["):
                meta[key] = [unquote(v) for v in value.strip("[]").split(",") if v.strip()]
            elif value:
                meta[key] = unquote(value)
    return meta


def unquote(s: str) -> str:
    s = s.strip()
    if len(s) >= 2 and s[0] == s[-1] and s[0] in "\"'":
        return s[1:-1]
    return s


def load_rules(only: list[str] | None) -> list[Rule]:
    rules = []
    for path in sorted(RULES_DIR.glob("*.md")):
        text = path.read_text()
        meta = parse_frontmatter(text)
        rule = Rule(
            id=meta.get("id", path.stem),
            applies_to=as_list(meta.get("applies_to")),
            excludes=as_list(meta.get("excludes")),
            trigger=meta.get("trigger", ""),
            context=as_list(meta.get("context")),
            text=text,
        )
        if not rule.applies_to:
            warn(f"{path.name}: no applies_to, skipped")
            continue
        if only and rule.id not in only:
            continue
        rules.append(rule)
    if only:
        missing = set(only) - {r.id for r in rules}
        if missing:
            sys.exit(f"no such rule: {', '.join(sorted(missing))}")
    return rules


def as_list(v) -> list[str]:
    if v is None:
        return []
    return v if isinstance(v, list) else [v]


_glob_cache: dict[str, re.Pattern] = {}


def glob_match(glob: str, path: str) -> bool:
    """Globs as in .gitignore: ** spans directories, * stays within one."""
    if glob not in _glob_cache:
        out, i = "", 0
        while i < len(glob):
            if glob.startswith("**/", i):
                out += "(?:.*/)?"
                i += 3
            elif glob.startswith("**", i):
                out += ".*"
                i += 2
            elif glob[i] == "*":
                out += "[^/]*"
                i += 1
            elif glob[i] == "?":
                out += "[^/]"
                i += 1
            else:
                out += re.escape(glob[i])
                i += 1
        _glob_cache[glob] = re.compile(out + r"\Z")
    return bool(_glob_cache[glob].match(path))


# ── git ──────────────────────────────────────────────────────────────────


def git(*args: str) -> str:
    return subprocess.run(
        ["git", *args], cwd=ROOT, check=True, capture_output=True, text=True
    ).stdout


def resolve_pr(number: int) -> tuple[str, str]:
    """The pull request's head commit and the base commit GitHub compares it
    to, which holds for a merged pull request too."""
    info = json.loads(subprocess.run(
        ["gh", "pr", "view", str(number), "--json", "baseRefName,baseRefOid,headRefOid"],
        cwd=ROOT, check=True, capture_output=True, text=True,
    ).stdout)
    git("fetch", "-q", "origin", f"pull/{number}/head", info["baseRefName"])
    return info["headRefOid"], info["baseRefOid"]


@dataclass
class Material:
    """What one file shows the checking agent: its diff or its content,
    with line numbers of the reviewed version."""

    path: str
    text: str


def numbered_diff(base: str, ref: str, path: str) -> str:
    """The file's diff, each line marked with its line number in ref:
    '  42 + added', '  43   context'; removed lines carry no number."""
    raw = git("diff", f"-U{CONTEXT_LINES}", "--no-color", base, ref, "--", path)
    out, new = [], 0
    for line in raw.splitlines():
        h = re.match(r"^@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@", line)
        if h:
            new = int(h.group(1))
            out.append(line)
        elif line.startswith(("diff ", "index ", "--- ", "+++ ", "new file", "deleted file", "similarity", "rename ")):
            continue
        elif line.startswith("+"):
            out.append(f"{new:5d} + {line[1:]}")
            new += 1
        elif line.startswith("-"):
            out.append(f"      - {line[1:]}")
        elif line.startswith("\\"):
            continue
        else:
            out.append(f"{new:5d}   {line[1:]}")
            new += 1
    return "\n".join(out)


def numbered_file(ref: str, path: str, around: int | None = None) -> str:
    try:
        lines = git("show", f"{ref}:{path}").splitlines()
    except subprocess.CalledProcessError:
        return "(file not readable at this ref)"
    lo, hi = 0, len(lines)
    if around is not None and len(lines) > 2 * FILE_WINDOW:
        lo, hi = max(0, around - FILE_WINDOW), min(len(lines), around + FILE_WINDOW)
    return "\n".join(f"{i + 1:5d}   {lines[i]}" for i in range(lo, hi))


def batches(materials: list[Material]) -> list[list[Material]]:
    """Files grouped so that each agent gets at most MAX_BATCH_CHARS; a
    larger file goes alone."""
    out: list[list[Material]] = []
    cur: list[Material] = []
    size = 0
    for m in materials:
        if cur and size + len(m.text) > MAX_BATCH_CHARS:
            out.append(cur)
            cur, size = [], 0
        cur.append(m)
        size += len(m.text)
    if cur:
        out.append(cur)
    return out


# ── agents ───────────────────────────────────────────────────────────────

CHECK_SCHEMA = {
    "type": "object",
    "required": ["files"],
    "properties": {
        "files": {
            "type": "array",
            "items": {
                "type": "object",
                "required": ["path", "verdict", "evidence"],
                "properties": {
                    "path": {"type": "string"},
                    "verdict": {"enum": ["ok", "violation", "not_applicable"]},
                    "evidence": {
                        "type": "string",
                        "description": "A line quoted verbatim from this file's material that the verdict rests on, without the line number and marker.",
                    },
                    "findings": {
                        "type": "array",
                        "items": {
                            "type": "object",
                            "required": ["line", "quote", "why"],
                            "properties": {
                                "line": {"type": "integer"},
                                "quote": {"type": "string"},
                                "why": {"type": "string"},
                            },
                        },
                    },
                },
            },
        }
    },
}

DISPROVE_SCHEMA = {
    "type": "object",
    "required": ["verdict", "reason"],
    "properties": {
        "verdict": {"enum": ["confirmed", "refuted"]},
        "reason": {"type": "string"},
    },
}

KNOWN_SCHEMA = {
    "type": "object",
    "required": ["matches"],
    "properties": {
        "matches": {
            "type": "array",
            "items": {
                "type": "object",
                "required": ["finding", "issue", "reason"],
                "properties": {
                    "finding": {"type": "integer"},
                    "issue": {"type": ["integer", "null"], "description": "The open issue that covers the finding, or null."},
                    "reason": {"type": "string"},
                },
            },
        }
    },
}


def open_issues() -> list[dict]:
    out = subprocess.run(
        ["gh", "issue", "list", "--state", "open", "--limit", "1000", "--json", "number,title,body"],
        cwd=ROOT, check=True, capture_output=True, text=True,
    ).stdout
    return json.loads(out)


def known_prompt(findings: list[dict], issues: list[dict]) -> str:
    parts = [
        "These findings are debt in existing code: a branch moved or re-indented the code, it did "
        "not write it. For each finding, name the open issue that already covers it, or null. An "
        "issue covers a finding when fixing the issue as described would fix the finding: the same "
        "rule broken in the same place, or a broader issue that names this kind of debt in this "
        "part of the code. A loosely related issue does not cover it; when in doubt, null.",
        "\n## Findings\n",
    ]
    for i, f in enumerate(findings):
        parts.append(f"{i}. rule {f['rule'].id}, {f['path']}:{f['line']}: {f['why']}\n   `{f['quote'].strip()}`")
    parts.append("\n## Open issues\n")
    for issue in issues:
        parts.append(f"### #{issue['number']} {issue['title']}\n\n{(issue['body'] or '').strip()}\n")
    return "\n".join(parts)


# The agents run on the logged-in Claude session (the subscription); an API
# key in the environment would make claude bill the API instead.
AGENT_ENV = {k: v for k, v in os.environ.items() if k not in ("ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN")}


def ask(prompt: str, schema: dict) -> dict:
    """One agent, without tools: everything it needs is in the prompt."""
    cmd = [
        "claude", "-p", "--output-format", "json", "--no-session-persistence",
        "--tools", "", "--json-schema", json.dumps(schema),
    ]
    run = subprocess.run(
        cmd, input=prompt, cwd=ROOT, env=AGENT_ENV, capture_output=True, text=True,
        timeout=CALL_TIMEOUT,
    )
    try:
        reply = json.loads(run.stdout)
    except json.JSONDecodeError:
        raise RuntimeError(f"claude failed: {run.stderr.strip() or run.stdout.strip()[:500]}")
    if reply.get("is_error") or "structured_output" not in reply:
        raise RuntimeError(f"claude answered no verdict: {str(reply.get('result'))[:500]}")
    return reply["structured_output"]


def context_section(rule: Rule, ref: str) -> str:
    """The files the rule names as context, at the reviewed version, so a
    branch that updates the spec is judged against its update."""
    parts = []
    for path in rule.context:
        try:
            parts.append(f"\n### {path}\n\n```\n{git('show', f'{ref}:{path}')}\n```")
        except subprocess.CalledProcessError:
            parts.append(f"\n### {path}\n\n(not present at this version)")
    return "\n## Context the rule refers to\n" + "".join(parts) if parts else ""


def check_prompt(rule: Rule, batch: list[Material], mode: str, problems: list[str], ref: str) -> str:
    scope = (
        "You see the changes of a branch: lines marked + are new, lines marked - are removed, "
        "the rest is context. Judge only the new lines (+); context shows what they belong to."
        if mode == "review"
        else "You see whole files. Judge every line."
    )
    parts = [
        "You review code against exactly one rule of this repository. Check only this rule; "
        "other concerns are someone else's.",
        f"\n## The rule\n\n{rule.text}",
        context_section(rule, ref),
        f"\n## What you check\n\n{scope}\n"
        "Each line starts with its line number. Not every file need concern the rule: the "
        f"rule applies when the code {rule.trigger or 'falls under it'}.",
        "\n## Your answer\n\n"
        "For every file below, one verdict:\n"
        "- `violation`: the rule is broken; list each place as a finding with its line number, "
        "the line quoted verbatim, and why it breaks the rule;\n"
        "- `ok`: the rule applies and is kept;\n"
        "- `not_applicable`: the file does nothing the rule is about.\n"
        "Every verdict quotes, in `evidence`, one line verbatim from that file's material, "
        "without its line number and marker. A verdict or finding without a verbatim quote is "
        "rejected. Report a violation only where the rule, read as written, is broken; when in "
        "doubt, it is ok.",
    ]
    if problems:
        parts.append(
            "\n## Your previous answer was rejected\n\n" + "\n".join(f"- {p}" for p in problems)
        )
    for m in batch:
        parts.append(f"\n## File: {m.path}\n\n```\n{m.text}\n```")
    return "\n".join(parts)


def normalize(s: str) -> str:
    return re.sub(r"\s+", " ", s).strip()


def strip_markers(text: str) -> str:
    """The material without line numbers and +/- markers, to find quotes in."""
    return "\n".join(re.sub(r"^\s*\d*\s[+\- ]\s", "", line) for line in text.splitlines())


def validate(answer: dict, batch: list[Material]) -> list[str]:
    problems = []
    by_path = {m.path: normalize(strip_markers(m.text)) for m in batch}
    seen = set()
    for f in answer.get("files", []):
        path = f.get("path")
        if path not in by_path:
            problems.append(f"{path}: no such file in the material")
            continue
        seen.add(path)
        quote = normalize(f.get("evidence", ""))
        if not quote or quote not in by_path[path]:
            problems.append(f"{path}: the evidence is no verbatim quote from the file")
        if f.get("verdict") == "violation":
            if not f.get("findings"):
                problems.append(f"{path}: a violation needs at least one finding")
            for x in f.get("findings", []):
                q = normalize(x.get("quote", ""))
                if not q or q not in by_path[path]:
                    problems.append(f"{path}:{x.get('line')}: the finding's quote is not verbatim")
    for path in by_path.keys() - seen:
        problems.append(f"{path}: no verdict")
    return problems


def disprove_prompt(rule: Rule, finding: dict, ref: str, mode: str, diff: str) -> str:
    scope = (
        "This is a review of a branch's changes. Every line the branch adds or changes (marked + "
        "in its changes) is in scope, also when the branch only moved or re-indented code that "
        "existed before: touching code surfaces its debt. Only a finding on a context line the "
        "branch left as it was is out of scope and refuted."
        if mode == "review"
        else "This is an audit of the whole repository: every line is in scope."
    )
    parts = [
        "A reviewer says this code breaks a rule of this repository. Try to disprove it. "
        "Refute it when the rule does not cover this code (check its applies_to, its trigger and "
        "the exceptions it names), when the quote is misread or taken out of context, or when the "
        "code keeps the rule after all. Confirm it only when it holds against your best attempt.",
        f"\n{scope}",
        f"\n## The rule\n\n{rule.text}",
        context_section(rule, ref),
        "\n## The finding\n\n"
        f"- file: {finding['path']}\n- line: {finding['line']}\n"
        f"- quote: {finding['quote']}\n- why: {finding['why']}",
    ]
    if diff:
        parts.append(f"\n## The branch's changes to this file\n\n```\n{diff}\n```")
    parts.append(
        f"\n## The file at the reviewed version\n\n```\n"
        f"{numbered_file(ref, finding['path'], around=finding['line'])}\n```"
    )
    return "\n".join(parts)


# ── the run ──────────────────────────────────────────────────────────────


@dataclass
class Job:
    rule: Rule
    batch: list[Material]
    answer: dict | None = None
    problems: list[str] = field(default_factory=list)
    error: str | None = None


def run_check(job: Job, mode: str, ref: str) -> Job:
    problems: list[str] = []
    for _ in range(2):  # one retry on a rejected answer
        try:
            answer = ask(check_prompt(job.rule, job.batch, mode, problems, ref), CHECK_SCHEMA)
        except Exception as e:  # noqa: BLE001 - reported, not raised
            job.error = str(e)
            return job
        problems = validate(answer, job.batch)
        if not problems:
            job.answer = answer
            return job
    job.problems = problems
    return job


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("ref", nargs="?", default="HEAD", help="branch or commit to review (default HEAD)")
    ap.add_argument("--base", default=DEFAULT_BASE, help=f"what the branch is compared to (default {DEFAULT_BASE})")
    ap.add_argument("--pr", type=int, help="review a pull request instead of a branch")
    ap.add_argument("--audit", action="store_true", help="check every file, not only the changes")
    ap.add_argument("--rule", action="append", help="check only this rule (repeatable)")
    ap.add_argument("--jobs", type=int, default=6, help="agents at once (default 6)")
    args = ap.parse_args()

    ref, base, label = args.ref, args.base, args.ref
    if args.pr:
        ref, base = resolve_pr(args.pr)
        label = f"pr-{args.pr}"
    sha = git("rev-parse", ref).strip()
    mode = "audit" if args.audit else "review"
    rules = load_rules(args.rule)

    if mode == "review":
        merge_base = git("merge-base", base, sha).strip()
        numstat = git("diff", "--numstat", "--no-renames", "--diff-filter=AM", merge_base, sha)
    else:
        merge_base = ""
        numstat = git("diff", "--numstat", git("hash-object", "-t", "tree", "/dev/null").strip(), sha)
    # numstat marks binary files with "-": an agent cannot read them
    files = [line.split("\t")[2] for line in numstat.splitlines() if not line.startswith("-\t")]

    material_cache: dict[str, Material] = {}

    def material(path: str) -> Material:
        if path not in material_cache:
            text = numbered_diff(merge_base, sha, path) if mode == "review" else numbered_file(sha, path)
            material_cache[path] = Material(path, text)
        return material_cache[path]

    jobs: list[Job] = []
    skipped: list[str] = []
    for rule in rules:
        matched = [f for f in files if rule.matches(f)]
        if not matched:
            skipped.append(rule.id)
            continue
        for batch in batches([material(f) for f in matched]):
            jobs.append(Job(rule, batch))

    stamp = datetime.datetime.now().strftime("%Y%m%d-%H%M%S")
    out = OUT_DIR / f"{stamp}-{re.sub(r'[^A-Za-z0-9._-]+', '-', label)}"
    (out / "raw").mkdir(parents=True)

    print(f"{mode} of {label} ({sha[:8]}){' against ' + base if mode == 'review' else ''}: "
          f"{len(files)} files, {len(jobs)} checks for {len(rules) - len(skipped)} rules")

    # 1. check, one agent per rule and batch
    with futures.ThreadPoolExecutor(args.jobs) as pool:
        done = list(pool.map(lambda j: run_check(j, mode, sha), jobs))
    for i, job in enumerate(done):
        (out / "raw" / f"check-{job.rule.id}-{i}.json").write_text(json.dumps(
            {"rule": job.rule.id, "files": [m.path for m in job.batch], "answer": job.answer,
             "problems": job.problems, "error": job.error}, indent=2))
        print(f"  {job.rule.id}: {', '.join(m.path for m in job.batch)[:100]} "
              f"{'✗ ' + (job.error or 'rejected') if not job.answer else '✓'}")

    findings = []
    for job in done:
        for f in (job.answer or {}).get("files", []):
            if f["verdict"] == "violation":
                for x in f.get("findings", []):
                    findings.append({"rule": job.rule, "path": f["path"], **x})

    # 2. disprove, one agent per violation
    def disprove(finding: dict) -> dict:
        diff = material_cache[finding["path"]].text if mode == "review" else ""
        try:
            v = ask(disprove_prompt(finding["rule"], finding, sha, mode, diff), DISPROVE_SCHEMA)
        except Exception as e:  # noqa: BLE001
            v = {"verdict": "error", "reason": str(e)}
        return {**finding, "disproof": v}

    # a finding whose line the branch removed somewhere is carried (moved or
    # re-indented), not written: debt the branch surfaced, to file for later
    if mode == "review":
        removed = {
            normalize(line[1:])
            for line in git("diff", "--no-color", "-U0", merge_base, sha).splitlines()
            if line.startswith("-") and not line.startswith("---") and line[1:].strip()
        }
        for f in findings:
            f["origin"] = "carried" if normalize(f["quote"]) in removed else "new"

    if findings:
        print(f"disproving {len(findings)} findings")
    with futures.ThreadPoolExecutor(args.jobs) as pool:
        judged = list(pool.map(disprove, findings))

    # 3. carried debt an open issue already covers is known, not a finding;
    # debt the branch writes is always surfaced
    carried = [f for f in judged if f["disproof"]["verdict"] == "confirmed" and f.get("origin") == "carried"]
    if carried:
        print(f"checking {len(carried)} carried findings against the open issues")
        try:
            issues = open_issues()
            numbers = {i["number"] for i in issues}
            answer = ask(known_prompt(carried, issues), KNOWN_SCHEMA)
            for m in answer.get("matches", []):
                if 0 <= m["finding"] < len(carried) and m.get("issue") in numbers:
                    carried[m["finding"]]["known"] = {"issue": m["issue"], "reason": m["reason"]}
        except Exception as e:  # noqa: BLE001 - without issues, carried findings stay findings
            warn(f"issue check failed: {e}")

    write_report(out, label, sha, base, mode, rules, skipped, done, judged)
    confirmed = sum(1 for f in judged if f["disproof"]["verdict"] == "confirmed" and "known" not in f)
    known = sum(1 for f in judged if "known" in f)
    print(f"\n{confirmed} confirmed, {known} known from issues, "
          f"{len(judged) - confirmed - known} refuted or failed"
          f"\n→ {out.relative_to(ROOT)}/report.md")


def write_report(out: Path, label, sha, base, mode, rules, skipped, done, judged) -> None:
    (out / "findings.json").write_text(json.dumps(
        [{**{k: v for k, v in f.items() if k != "rule"}, "rule": f["rule"].id} for f in judged], indent=2))

    def section(title: str, items: list[dict]) -> list[str]:
        lines = [f"\n## {title} ({len(items)})"]
        for rule_id in sorted({f["rule"].id for f in items}):
            lines.append(f"\n### {rule_id}")
            for f in sorted((f for f in items if f["rule"].id == rule_id), key=lambda f: (f["path"], f["line"])):
                tag = f" ({f['origin']})" if f.get("origin") else ""
                lines += [
                    f"\n**{f['path']}:{f['line']}**{tag}: {f['why']}",
                    f"\n```\n{f['quote']}\n```",
                    f"\n{f['disproof']['verdict']}: {f['disproof']['reason']}",
                ]
                if "known" in f:
                    lines.append(f"\ncovered by #{f['known']['issue']}: {f['known']['reason']}")
        return lines

    confirmed = [f for f in judged if f["disproof"]["verdict"] == "confirmed" and "known" not in f]
    known = [f for f in judged if "known" in f]
    rest = [f for f in judged if f["disproof"]["verdict"] != "confirmed"]
    failed = [j for j in done if not j.answer]
    lines = [
        f"# Review of {label}",
        "",
        f"- commit: `{sha}`",
        f"- mode: {mode}" + (f", against `{base}`" if mode == "review" else ""),
        f"- rules: {', '.join(r.id for r in rules if r.id not in skipped) or 'none'}",
        f"- no files for: {', '.join(skipped) or 'none'}",
    ]
    lines += section("Confirmed", confirmed)
    lines += section("Known debt, covered by an open issue", known)
    lines += section("Refuted", rest)
    if failed:
        lines.append(f"\n## Checks without a valid answer ({len(failed)})")
        for j in failed:
            lines.append(f"\n- {j.rule.id}: {', '.join(m.path for m in j.batch)}")
            for p in j.problems or [j.error]:
                lines.append(f"  - {p}")
    (out / "report.md").write_text("\n".join(lines) + "\n")


def warn(msg: str) -> None:
    print(f"warning: {msg}", file=sys.stderr)


if __name__ == "__main__":
    main()

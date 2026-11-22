# Reviewing a branch against the rules

`scripts/review.py` checks a branch, a pull request or the whole repository
against the rules in `docs/rules/`. It needs `claude` (Claude Code), signed
in, and for pull requests `gh`. The agents run as `claude -p` on that signed-in
session; an API key in the environment is not passed on.

```sh
scripts/review.py                     # this branch against v2/development
scripts/review.py feat/x              # another branch
scripts/review.py --pr 488            # a pull request, open or merged
scripts/review.py --audit             # every file: the debt, not the change
scripts/review.py --rule ui-text      # only some rules (repeatable)
```

## What it does

1. Takes the changed files (review) or every file (audit), and per rule the
   files its `applies_to` and `excludes` select. A rule without files is
   skipped.
2. Gives each rule its own agent, with only that rule, the files it may read
   (`context`, e.g. the spec) and the changes to check. Every verdict quotes
   the lines it rests on; a quote not found in the material rejects the
   answer, which is asked again once.
3. Hands each violation to a second agent that tries to disprove it. Every
   line the branch adds or changes is in scope, also code it only moved or
   re-indented: touching code surfaces its debt. A finding is marked `new`
   when the branch wrote the line, `carried` when the same line was removed
   elsewhere in the branch (moved or re-indented); a very common line can be
   marked carried by coincidence.
4. Compares the carried findings with the open GitHub issues. Carried debt
   that an issue already covers is no finding; the report lists it under
   "Known debt" with the issue. New debt is never excused by an issue: what
   the branch introduces, it fixes.
5. Writes `.reviews/<time>-<branch>/report.md` (gitignored): the confirmed
   findings by rule with file and line (new ones to fix in the branch,
   carried ones to file as issues), the known debt, the refuted ones with
   the reason, and checks that gave no valid answer. `findings.json` and `raw/` hold
   the same for tools and for digging.

The rules come from the working tree, not from the reviewed branch, so a
branch cannot loosen the rules it is checked against.

## After the run

Read `report.md`, or hand it to an agent to discuss: "read
.reviews/<run>/report.md; which findings should be fixed, and is a rule
wrong?". A refuted finding that was right, or a confirmed one that was not,
points at a rule to sharpen. A mistake a review finds twice becomes a rule.

## Writing a rule

One file per rule in `docs/rules/`, frontmatter first:

```markdown
---
id: list-table
applies_to:
  - "frontend/src/**/*.tsx"
excludes:
  - "frontend/src/common/components/ListTable/**"
context:                       # optional: files every check reads
  - "docs/chatgamelab.typ"
trigger: "renders a list or table of records"
---
```

Then the rule, why it holds, and a bad and a good example: the examples
define the rule more than its prose. Rules that a tool can check (types,
imports, tests) belong to the tool, not here.

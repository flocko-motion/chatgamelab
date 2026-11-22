# Opening the seal

The secure core decides who may do what, and it is the only code that reaches
the database. The seal keeps it from changing in passing: a change to it is a
deliberate act of the owner, and it is visible.

## What is sealed

The paths in `.github/CODEOWNERS`: the core (`go/core/`), the database
(`go/db/`), the composition root (`go/app/app.go`), the tests that pin the
rules down (permission matrix, guest place, `scripts/test.sh`) and the seal
itself (CODEOWNERS, the workflows, `.claude/settings.json`).

## Three layers

1. **Tests**, for every pull request (`Tests`, required as `Run Tests`): the
   core's boundary test (only the core reaches the database), its gate
   coverage, the permission matrix and the guest-place constraints.
2. **The ruleset** `core-seal` on `v2/development`: a pull request touching a
   sealed path needs the code owner's approval. The workflow `core-seal`
   labels it `core`.
3. **Agents**: `.claude/settings.json` denies agents edits to the sealed paths
   and the commands that would open the seal (admin merges, ruleset changes).
   This layer is a guardrail; the ruleset is the seal.

## How to open it

Agents push with the owner's account, so the owner is the author of their pull
requests, and GitHub lets no one approve their own. The approval is therefore
given as a **bypass merge**: on the pull request, the owner ticks "Merge
without waiting for requirements to be met (bypass rules)" and merges. GitHub
logs every bypass (Settings → Rules → Insights). The bypass is allowed only
through a pull request, never as a direct push. It skips every ruleset the
owner may bypass, `Run Tests` (ruleset "Integration Tests") among them, so the
owner ticks it only once `Run Tests` is green.

Before ticking it, the owner reads the sealed part of the diff; the `core`
label and the job summary of `core-seal` list the sealed files touched.

A pull request by someone else (a second account, a contributor) is approved
the ordinary way: the owner reviews and approves it.

### When an agent works on the core

The deny rules hold in every worktree that has them, and an agent cannot lift
them: deny wins over every allow, and `.claude/settings.json` is itself
denied. For an issue that changes the core, the owner removes the deny rules
by hand from that worktree's `.claude/settings.json` and does not commit that
change; the worktree is removed after the merge.

## Setting it up

Once, by the owner:

```sh
gh api -X POST repos/flocko-motion/chatgamelab/rulesets --input - <<'EOF'
{
  "name": "core-seal",
  "target": "branch",
  "enforcement": "active",
  "conditions": { "ref_name": { "include": ["refs/heads/v2/development"], "exclude": [] } },
  "bypass_actors": [
    { "actor_id": 5, "actor_type": "RepositoryRole", "bypass_mode": "pull_request" }
  ],
  "rules": [
    {
      "type": "pull_request",
      "parameters": {
        "required_approving_review_count": 0,
        "require_code_owner_review": true,
        "dismiss_stale_reviews_on_push": true,
        "require_last_push_approval": false,
        "required_review_thread_resolution": false
      }
    }
  ]
}
EOF
```

Actor 5 is the repository's admin role; `bypass_mode: pull_request` allows
the bypass on a pull request's merge only.

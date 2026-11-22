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
2. **The ruleset** `core-seal` on `v2/development` requires the check
   `core-seal`. The workflow of that name fails for every pull request that
   touches a sealed path, labels it `core` and lists the sealed files in its
   job summary; for every other pull request it passes.
3. **Agents**: `.claude/settings.json` denies agents edits to the sealed paths
   and the commands that would open the seal (admin merges, ruleset changes).
   This layer is a guardrail; the ruleset is the seal.

## How to open it

A pull request touching the seal never turns green on its own. The owner's
approval is a **bypass merge**: on the pull request, the owner ticks "Merge
without waiting for requirements to be met (bypass rules)" and merges. GitHub
logs every bypass (Settings → Rules → Insights). The bypass is allowed only
through a pull request, never as a direct push. It skips every ruleset the
owner may bypass, `Run Tests` (ruleset "Integration Tests") among them, so the
owner ticks it only once `Run Tests` is green.

Before ticking it, the owner reads the sealed part of the diff; the `core`
label and the job summary of `core-seal` list the sealed files touched.

A review cannot be the approval: agents push with the owner's account, so the
owner is the author of their pull requests, and GitHub asks no author to
review files they own. CODEOWNERS still marks the sealed paths and asks the
owner to review pull requests by others; those, too, merge by bypass.

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
      "type": "required_status_checks",
      "parameters": {
        "strict_required_status_checks_policy": false,
        "required_status_checks": [{ "context": "core-seal", "integration_id": 15368 }]
      }
    }
  ]
}
EOF
```

Actor 5 is the repository's admin role; `bypass_mode: pull_request` allows
the bypass on a pull request's merge only. Integration 15368 is GitHub
Actions, which reports the check.

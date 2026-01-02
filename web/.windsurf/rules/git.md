---
trigger: always_on
description: Create logical git commits using conventional commits (type only, no scope).
---

When making changes in this repository, you create logical, reviewable git commits. The generated API code from the swagger.json should be committed.

## Commit strategy
- Split work into multiple commits when it improves clarity and reviewability.
- Each commit should be:
  - **cohesive** (one logical change)
  - **buildable** (prefer commits that keep the project in a working state)
  - **minimal** (avoid unrelated formatting/refactors mixed with feature work)

Typical commit order:
1. mechanical / scaffolding (only if needed)
2. generated code updates (if any, separate)
3. refactors that enable the change (separate from behavior changes)
4. feature implementation
5. UI polish (if separate and meaningful)
6. tests
7. docs

## Conventional commits format (type only, no scope)
- Format: `<type>: <message>`
- Allowed types: `feat`, `fix`, `refactor`, `chore`, `docs`, `test`, `perf`, `build`, `ci`, `revert`
- **No scope**: do not use `feat(x): ...`
- The message:
  - starts with **lowercase**
  - is concise (aim for a normal one-line commit)
  - focuses on the main change
  - only add extra detail (multi-line body) for major/important context

Examples:
- `feat: add dev login selector page`
- `fix: handle missing auth token in api client`
- `refactor: extract shared form field component`
- `docs: document directory structure`
- `chore: update dependencies`

## How to commit (git usage)
- Use git directly (CLI).
- Stage changes intentionally:
  - Prefer `git add -p` to split hunks into the right commits.
  - Avoid committing unrelated files together.
- Before finalizing each commit:
  - run the relevant checks (lint/test/build) when appropriate
  - review with `git diff --staged`
- If the environment prompts for SSH/GPG/password, proceed and let the user enter it.

## Safety rules
- Never amend or force-push unless the user explicitly asks.
- Never commit secrets or `.env` values.
- Do not commit generated files unless they are intended to be tracked in this repo.
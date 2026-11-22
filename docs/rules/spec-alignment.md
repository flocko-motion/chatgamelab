---
id: spec-alignment
applies_to:
  - "go/core/**/*.go"
  - "go/db/migrations/**"
  - "go/api/**/*.go"
  - "go/engine/**/*.go"
  - "frontend/src/**/*.tsx"
  - "frontend/src/**/*.ts"
  - "docs/chatgamelab.typ"
excludes:
  - "**/*_test.go"
  - "**/*.gen.go"
  - "frontend/src/api/generated/**"
  - "frontend/src/i18n/**"
context:
  - "docs/chatgamelab.typ"
trigger: "changes a concept: who may do what (access rules, grants), user roles, what a role means, or a user flow (signing up, inviting, joining, playing, sharing, paying)"
---

# Concepts follow the spec

The spec (`docs/chatgamelab.typ`) holds the concepts: what the product is and
does, who may do what, which roles there are and what they mean, and the flows
users go through. A change to a concept either implements what the spec says
or updates the spec in the same change.

```text
Bad:  A new grant lets staff delete games of other workshops; the spec's
      roles say staff manage their organisation's workshops only.
Bad:  Guests may now leave their workshop; the spec says a guest's place
      is fixed, and the spec is unchanged.
Good: The same grant, with the spec's Roles section saying so in the
      same change.
Good: A new endpoint lists a workshop's games for staff, as the spec's
      roles already allow.
```

- **Concepts**: access rules and grants, user roles and what they mean,
  user flows. These are in the spec, and code and spec agree on them.
- **No concepts**: design decisions, implementation details, technical
  details (tables, queries, endpoints, components, libraries, layout). The
  spec does not care how a concept is built; such a change needs no spec
  change.
- **When code and spec differ**, the finding says which way: the code drifts
  from the spec, or the spec misses what the change introduces.

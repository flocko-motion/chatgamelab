---
id: frontend-structure
applies_to:
  - "frontend/src/**/*.ts"
  - "frontend/src/**/*.tsx"
excludes:
  - "frontend/src/api/generated/**"
  - "frontend/src/routeTree.gen.ts"
trigger: "adds or moves a file, adds a component, hook or helper, or adds logic to a component"
---

# Frontend structure: headless logic, thin components

## Components hold no logic

Components render. The only state they may hold is UI state: whether a button
is pressed or loading, a menu or modal open, a tab selected, a field focused.
Everything else lives in hooks, and the component calls them: fetching and
caching data, filtering, sorting, paging, validation, permissions, business
rules, API calls, derived values.

```tsx
// Bad: the component filters, sorts and calls the API
function MyGames() {
  const { data } = useQuery(...);
  const [sort, setSort] = useState("title");
  const games = data?.filter(g => g.mine).sort(by(sort));
  const remove = (id) => api.games.delete(id).then(refetch);
  ...
}

// Good: the hook owns the logic, the component renders it
function MyGames() {
  const games = useMyGames();
  const [confirmOpen, setConfirmOpen] = useState(false);   // UI state: fine
  return <GamesTable list={games} onDelete={games.remove} />;
}
```

## Code sits where its scope is

- Used by one feature: `features/<feature>/` (`components/`, `hooks/`, …).
- Used by several features: `common/` (`components/`, `hooks/`, `lib/`,
  `types/`).
- Talking to the API: `api/` (`client/`, `hooks/`).
- Global configuration: `config/`; app-wide providers: `providers/`; texts:
  `i18n/`.
- Routes are thin: they compose features and hold no logic.

## No duplicates

A new component, hook or helper does not duplicate one that exists in
`common/`. A component a second feature starts to use moves to `common/`.

## Dependency direction

`common/` and `config/` do not import from `features/`; features do not
import from `routes/`.

## Generated files

`api/generated/` and `routeTree.gen.ts` are never edited by hand; change the
source and regenerate.

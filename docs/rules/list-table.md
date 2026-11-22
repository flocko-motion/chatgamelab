---
id: list-table
applies_to:
  - "frontend/src/**/*.tsx"
  - "frontend/src/**/*.ts"
excludes:
  - "frontend/src/common/components/ListTable/**"
  - "frontend/src/api/lists/**"
  - "frontend/src/api/generated/**"
trigger: "renders a list or table of records, or lets the user search, filter, sort or page one"
---

# Lists are a List<T> from the API layer, shown by ListTable

A list of records comes from its entity hook in `api/` (`useGames`,
`useUsers`, `useInvites`, `useOrganisations`, `useMembers`,
`useWorkshopList`, `useParticipants`, `useOrganisationApiKeys`, …) as a
`List<T>`: the page, how many match in all, the loading and error state, the
query, what the list offers (its preset), and the actions to search, filter,
toggle, sort and page. `ListTable` (`@components/ListTable`) renders it. The
list sends what the user wants and shows the page that comes back; the
server filters, searches, sorts and pages (`core.List`), and its filters and
orders reach the web app as the generated client's types. Toolbar, sortable
headers, cards on a phone and paging look and work the same in every list.

```tsx
// Bad: a list of its own, filtering and sorting on the client
const [sortValue, setSortValue] = useState("name-asc");
const shown = users.filter((u) => u.name.includes(search)).sort(byName);
<Table>
  <Table.Thead>…</Table.Thead>
  <Table.Tbody>{shown.map((u) => <Table.Tr key={u.id}>…</Table.Tr>)}</Table.Tbody>
</Table>

// Good
// features/admin/lib/userListPresets.ts: declared once, so it stays the same
export const USER_LIST: ListPreset<"noGuests" | "all", UserListSort> = {
  filters: [
    { value: "noGuests", label: "admin.users.filters.noGuests" },
    { value: "all", label: "admin.users.filters.all", apply: [] },
  ],
  sorts: [...bothWays("name", (dir) => `admin.users.sort.name-${dir}`, ["asc", "desc"])],
  defaultSort: { by: "name", dir: "asc" },
};

// the component
const users = useUsers(USER_LIST);
<ListTable
  list={users}
  columns={columns}
  getRowKey={(user) => user.id}
  renderCard={renderCard}
/>
```

- **The entity hook builds the list** (`useEntityList` in `api/lists/`),
  from `useListQuery` and `useServerList`; features take it from there.
- **A preset holds what a list offers**, each option with its label (an i18n
  key): filters, toggles and orders, declared once in the feature
  (`ListPreset`, `gameListPresets.ts`). `ListTable` shows them from the list.
  An entity's preconfigured table keeps its columns beside its preset
  (`GamesTable`).
- **Columns are `ListColumn`s**; a header sorts with `sortBy`, a column that
  needs width shows from a breakpoint with `visibleFrom`.
- **The list's state is its query**: search, filter, sort and page live in
  it, in the URL by default, so back and links keep them. A route
  that hosts such a list reads the keys with `validateListSearch`.
- **Embedded lists** (a list inside a row, a card or a settings page) keep
  their query local (`inUrl: false`) and use `ListTable`'s `compact` form: no
  toolbar, and paging only when the list holds more than a page.
- **A row that opens** shows its details below itself (`renderExpanded`),
  e.g. an organisation's members as an embedded list.
- **Orders use only fields the caller can see.**

A small fixed table that is no list of records (a key-value overview, a
legend) may use Mantine's `Table` directly.

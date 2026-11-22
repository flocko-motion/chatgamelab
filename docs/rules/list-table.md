---
id: list-table
applies_to:
  - "frontend/src/**/*.tsx"
excludes:
  - "frontend/src/common/components/ListTable/**"
  - "frontend/src/api/generated/**"
trigger: "renders a list or table of records, or lets the user search, filter, sort or page one"
---

# Lists build on ListTable

A list of records builds on `ListTable` (`@components/ListTable`), with its
query in `useListQuery` and its page in `useServerList`. The server filters,
sorts and pages; the list sends what the user wants and shows what comes back.
Toolbar, sortable headers, cards on a phone and paging then look and work the
same in every list.

```tsx
// Bad: a list of its own, filtering and sorting on the client
const [sortValue, setSortValue] = useState("name-asc");
const shown = users.filter((u) => u.name.includes(search)).sort(byName);
<Table>
  <Table.Thead>…</Table.Thead>
  <Table.Tbody>{shown.map((u) => <Table.Tr key={u.id}>…</Table.Tr>)}</Table.Tbody>
</Table>

// Good
const controls = useListQuery({ filters, only: toggles, sorts, defaultSort });
const { query } = controls;
const result = useUsers({
  search: query.search, sortBy: query.sortBy, sortDir: query.sortDir,
  limit: query.pageSize, offset: listOffset(query),
});
const list = useServerList<UserView>(controls, result);
<ListTable
  list={list}
  controls={controls}
  columns={columns}
  getRowKey={(user) => user.id}
  renderCard={renderCard}
  sorts={sorts}
/>
```

- **Columns are `ListColumn`s**; a header sorts with `sortBy`, a column that
  needs width shows from a breakpoint with `visibleFrom`.
- **No list state of its own**: search, filter, sort and page live in the
  query, so back and links keep them.

A small fixed table that is no list of records (a key-value overview, a
legend) may use Mantine's `Table` directly.

import { Fragment, useState, type CSSProperties, type ReactNode } from "react";
import {
  Alert,
  Card,
  Group,
  Pagination,
  Select,
  SimpleGrid,
  Skeleton,
  Stack,
  Table,
  UnstyledButton,
  type MantineBreakpoint,
  type MantineColor,
} from "@mantine/core";
import {
  IconAlertCircle,
  IconArrowDown,
  IconArrowUp,
  IconChevronDown,
  IconChevronUp,
  IconMoodEmpty,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useResponsiveDesign } from "@/common/hooks/useResponsiveDesign";
import type { List } from "@/api/lists";
import { PAGE_SIZES, listOffset, pageCount } from "@/common/lib/listQuery";
import { DimmedLoader } from "@components/LoadingAnimation";
import { HelperText } from "@components/typography";
import { TextButton, ToggleIconButton } from "@components/buttons";
import {
  FilterBar,
  FilterMultiSelect,
  FilterSegmentedControl,
  FilterToggles,
  SearchInput,
  SortSelector,
} from "@components/controls";
import classes from "./ListTable.module.css";

export interface ListColumn<T, S extends string = string> {
  key: string;
  header: ReactNode;
  render: (item: T) => ReactNode;
  width?: number | string;
  /**
   * The order this column's header sorts by, if it sorts. Several orders
   * (e.g. modified, created) are cycled through, each descending then
   * ascending.
   */
  sortBy?: S | readonly S[];
  /** Show the column only from this breakpoint up (the table is wide). */
  visibleFrom?: MantineBreakpoint;
  /** Clicks in this cell don't open or expand the row (buttons, toggles). */
  interactive?: boolean;
  /**
   * A toggle in the header that narrows the list to the column's subject,
   * e.g. a star for "only favorites". A shortcut only: the same toggle is
   * among the preset's `toggles`. On a phone (cards, no headers) it sits in the
   * toolbar.
   */
  headerToggle?: ListHeaderToggle;
}

export interface ListHeaderToggle {
  active: boolean;
  onToggle: () => void;
  icon: ReactNode;
  activeIcon: ReactNode;
  label: string;
}

export interface ListTableProps<T, F extends string, S extends string> {
  /** The list, from its entity hook: page, state, query and the ways to change it. */
  list: List<T, F, S>;
  columns: ListColumn<T, S>[];
  getRowKey: (item: T) => string;
  /** How an item looks as a card: on a phone, and in the grid layout. */
  renderCard: (item: T) => ReactNode;
  /**
   * "table" (default): rows with sortable headers, cards on a phone.
   * "grid": cards on every screen, as many side by side as fit; the
   * toolbar holds the order and the header shortcuts, as on a phone.
   */
  layout?: "table" | "grid";
  onRowClick?: (item: T) => void;
  /** A colour stripe on the row's left edge, e.g. for sponsored games. */
  getRowAccent?: (item: T) => MantineColor | undefined;
  /**
   * What a row shows below itself when opened (e.g. an organisation's
   * members); a click on the row opens and closes it. Null: nothing to open.
   * On a phone a button below the card opens it.
   */
  renderExpanded?: (item: T) => ReactNode | null;
  /** Actions shown at the right end of the toolbar, e.g. create and import. */
  actions?: ReactNode;
  empty: { title: string; description?: string; icon?: ReactNode };
  errorMessage: string;
  /**
   * For a list inside another view (a row, a card, a settings page): no
   * toolbar, a plain table without a card around it. Paging shows only when
   * the list holds more than a page.
   */
  compact?: boolean;
  /**
   * How a filter option looks in the grid's filter bar: an icon, and the
   * colour it takes while on (e.g. a genre's). Its label stays the preset's.
   */
  optionLook?: (value: string) => { icon?: ReactNode; color?: string } | undefined;
}

/**
 * A list the server filters, sorts and pages: toolbar (search, filter, sort),
 * a table with sortable headers (cards on a phone), and a footer with the
 * pages. It renders only; the list comes from its entity hook in `api/`.
 */
export function ListTable<T, F extends string, S extends string>({
  list,
  columns,
  getRowKey,
  renderCard,
  layout = "table",
  onRowClick,
  getRowAccent,
  renderExpanded,
  actions,
  empty,
  errorMessage,
  compact = false,
  optionLook,
}: ListTableProps<T, F, S>) {
  const { preset } = list;
  const { t } = useTranslation(["common", ...presetNamespaces(preset)]);
  const label = (option: { label?: string; text?: string }) =>
    option.text ?? (option.label ? t(option.label) : "");
  const filters = (preset.filters ?? []).map((f) => ({ value: f.value, label: label(f) }));
  const toggles = (preset.toggles ?? []).map((o) => ({ value: o.value, label: label(o) }));
  const multi = preset.multiFilter;
  const multiOptions = (multi?.options ?? []).map((o) => ({ value: o.value, label: label(o) }));
  // The bar shows the multi-filter by what it lets through: none picked lets
  // all through, so all are on. Switching off the last one that is on does
  // nothing (it would show no game); all on again is no pick.
  const allPicks = multiOptions.map((o) => o.value);
  const shownPicks: readonly string[] = list.query.picked.length > 0 ? list.query.picked : allPicks;
  const toggleShown = (v: string) => {
    const next = shownPicks.includes(v) ? shownPicks.filter((x) => x !== v) : [...shownPicks, v];
    if (next.length === 0) return;
    list.setPicks((next.length === allPicks.length ? [] : next) as never);
  };
  const sorts = preset.sorts
    .filter((s) => s.label)
    .map((s) => ({ by: s.by, dir: s.dir, label: label(s) }));
  const { isMobile } = useResponsiveDesign();
  const { query } = list;
  // Cards have no headers: their sorts and shortcuts go to the toolbar.
  const cards = isMobile || layout === "grid";

  // Always open: on a phone it takes what the icons leave, on a wider
  // screen up to a comfortable width.
  const search = (
    <div style={{ display: "flex", flex: 1, minWidth: 0, maxWidth: isMobile ? undefined : 320 }}>
      <SearchInput
        value={list.searchInput}
        onChange={list.setSearch}
        placeholder={t("search")}
        clearLabel={t("list.clearSearch")}
      />
    </div>
  );
  const sortSelector = sorts.length > 0 && (
    <SortSelector
      options={sorts.map((s) => ({ value: `${s.by}-${s.dir}`, label: s.label }))}
      value={`${query.sortBy}-${query.sortDir}`}
      onChange={(v) => {
        const match = sorts.find((s) => `${s.by}-${s.dir}` === v);
        if (match) list.setSort(match.by, match.dir);
      }}
      label={t("games.sort.label")}
    />
  );

  const body = (
    <ListBody
      list={list}
      columns={columns}
      getRowKey={getRowKey}
      renderCard={renderCard}
      layout={layout}
      onRowClick={onRowClick}
      getRowAccent={getRowAccent}
      renderExpanded={renderExpanded}
      empty={empty}
      errorMessage={errorMessage}
      isMobile={isMobile}
      compact={compact}
    />
  );

  const footer = !list.isInitialLoading && !list.error && list.total > 0 && (
    <ListFooter list={list} compact={compact} />
  );

  if (compact) {
    return (
      <div className={classes.root}>
        {body}
        {list.total > list.items.length && footer}
      </div>
    );
  }

  // The grid (cards everywhere): search, order and actions on top, then every
  // filter in one bar, each on or off on its own and combinable.
  if (layout === "grid") {
    const look = (o: { value: string; label: string }) => ({ ...o, ...optionLook?.(o.value) });
    return (
      <div className={classes.root}>
        <div className={classes.gridToolbar}>
          <div className={classes.gridTop}>
            <div className={classes.gridSearch}>
              <SearchInput
                value={list.searchInput}
                onChange={list.setSearch}
                placeholder={t("search")}
                clearLabel={t("list.clearSearch")}
              />
            </div>
            {sortSelector}
            {actions && <div className={classes.gridActions}>{actions}</div>}
          </div>
          {(toggles.length > 0 || multiOptions.length > 0) && (
            <FilterBar
              label={t("list.filter")}
              clearLabel={t("list.clearFilters")}
              changed={query.only.length > 0 || query.picked.length > 0}
              onClear={() => {
                query.only.forEach((o) => list.toggleOnly(o));
                list.clearPicks();
              }}
              groups={[
                { options: toggles.map(look), active: query.only, onToggle: (v) => list.toggleOnly(v as never) },
                // Shown as what is shown: none picked shows every option, so
                // every option is on; switching one off picks the others.
                {
                  options: multiOptions.map(look),
                  active: shownPicks,
                  onToggle: (v) => toggleShown(v),
                },
              ]}
            />
          )}
        </div>

        <div>{body}</div>

        {footer}
      </div>
    );
  }

  return (
    <div className={classes.root}>
      <Group justify="space-between" wrap={isMobile ? "nowrap" : "wrap"} gap="sm">
        {/* Left: what narrows and orders the list. On a phone (cards, no
            headers) also the header shortcuts, the order and the search, each
            an icon: star, filter, sort, search. There the group takes the
            free space, so an opened search grows only up to the actions. */}
        <Group gap="sm" wrap="nowrap" style={{ minWidth: 0, flex: isMobile ? 1 : undefined }}>
          {cards &&
            columns.map(
              (column) =>
                column.headerToggle && (
                  <ToggleIconButton key={column.key} {...column.headerToggle} />
                ),
            )}
          {toggles.length > 0 && (
            <FilterToggles
              active={query.only}
              onToggle={list.toggleOnly}
              options={toggles}
              menuLabel={t("list.filter")}
            />
          )}
          {multi && multiOptions.length > 0 && (
            <FilterMultiSelect
              picked={query.picked}
              onToggle={list.togglePick}
              onClear={list.clearPicks}
              options={multiOptions}
              label={t(multi.label)}
              clearLabel={t("list.clearPicks")}
            />
          )}
          {filters.length > 1 && (
            <FilterSegmentedControl<F>
              value={query.filter}
              onChange={list.setFilter}
              options={filters}
            />
          )}
          {cards && sortSelector}
          {isMobile && search}
        </Group>
        {/* Right: the search on a wider screen (its headers sort), then the
            list's actions (create, import). */}
        <Group
          gap="sm"
          wrap="nowrap"
          style={
            isMobile ? { flexShrink: 0 } : { flex: 1, justifyContent: "flex-end", minWidth: 0 }
          }
        >
          {!isMobile && search}
          {actions}
        </Group>
      </Group>

      <div>{body}</div>

      {footer}
    </div>
  );
}

function ListBody<T, F extends string, S extends string>({
  list,
  columns,
  getRowKey,
  renderCard,
  layout,
  onRowClick,
  getRowAccent,
  renderExpanded,
  empty,
  errorMessage,
  isMobile,
  compact,
}: Omit<ListTableProps<T, F, S>, "actions"> & {
  isMobile: boolean;
}) {
  const { t } = useTranslation("common");
  // Which rows are open: UI state, kept here.
  const [open, setOpen] = useState<ReadonlySet<string>>(() => new Set());
  const toggleOpen = (key: string) =>
    setOpen((prev) => {
      const next = new Set(prev);
      if (!next.delete(key)) next.add(key);
      return next;
    });
  const grid = layout === "grid";

  if (list.error) {
    return (
      <Alert icon={<IconAlertCircle size={16} />} title={t("error")} color="red">
        {errorMessage}
      </Alert>
    );
  }

  if (list.isInitialLoading) {
    if (grid) {
      return (
        <SimpleGrid cols={GRID_COLS} spacing="lg">
          {[1, 2, 3, 4].map((i) => (
            <Skeleton key={i} height={320} radius="lg" />
          ))}
        </SimpleGrid>
      );
    }
    return isMobile ? (
      <Stack gap="md">
        {[1, 2, 3].map((i) => (
          <Skeleton key={i} height={140} radius="md" />
        ))}
      </Stack>
    ) : (
      <Skeleton height={300} radius="md" />
    );
  }

  const emptyContent = (
    <Stack align="center" gap="md" py="xl">
      {empty.icon ?? <IconMoodEmpty size={48} color="var(--mantine-color-gray-5)" />}
      <HelperText ta="center">{empty.title}</HelperText>
      {empty.description && <HelperText ta="center">{empty.description}</HelperText>}
    </Stack>
  );

  // With cards the toolbar holds every control, so an empty list is a card.
  // A table keeps its header even when empty: its sorts and toggles (e.g.
  // "only favorites") must stay reachable to undo a filter that left nothing.
  if (list.items.length === 0 && (isMobile || grid)) {
    return (
      <DimmedLoader visible={list.isRefetching} loaderSize="lg">
        <Card shadow="sm" p="xl" radius="md" withBorder>
          {emptyContent}
        </Card>
      </DimmedLoader>
    );
  }

  if (grid) {
    return (
      <DimmedLoader visible={list.isRefetching} loaderSize="lg">
        <SimpleGrid cols={GRID_COLS} spacing="lg" verticalSpacing="lg">
          {list.items.map((item) => (
            <div key={getRowKey(item)} className={classes.gridCell}>
              {renderCard(item)}
            </div>
          ))}
        </SimpleGrid>
      </DimmedLoader>
    );
  }

  if (isMobile) {
    return (
      <DimmedLoader visible={list.isRefetching} loaderSize="lg">
        <Stack gap="md">
          {list.items.map((item) => {
            const key = getRowKey(item);
            const expanded = renderExpanded?.(item);
            const isOpen = !!expanded && open.has(key);
            return (
              <div key={key}>
                {renderCard(item)}
                {expanded && (
                  <TextButton
                    size="xs"
                    onClick={() => toggleOpen(key)}
                    rightSection={
                      isOpen ? <IconChevronUp size={14} /> : <IconChevronDown size={14} />
                    }
                  >
                    {isOpen ? t("list.hideDetails") : t("list.showDetails")}
                  </TextButton>
                )}
                {isOpen && <div className={classes.cardExpanded}>{expanded}</div>}
              </div>
            );
          })}
        </Stack>
      </DimmedLoader>
    );
  }

  const { query } = list;
  const table = (
    <Table.ScrollContainer minWidth={compact ? undefined : 600} type="native">
      <Table
        verticalSpacing={compact ? "xs" : "md"}
        horizontalSpacing={compact ? "sm" : "lg"}
        stickyHeader={!compact}
        highlightOnHover
      >
        <Table.Thead>
          <Table.Tr className={classes.headRow}>
            {columns.map((column) => {
              const orders = sortOrders(column);
              const active = orders.includes(query.sortBy);
              return (
                <Table.Th
                  key={column.key}
                  w={column.width}
                  visibleFrom={column.visibleFrom}
                  className={classes.th}
                  aria-sort={
                    active ? (query.sortDir === "asc" ? "ascending" : "descending") : undefined
                  }
                >
                  {orders.length > 0 ? (
                    <UnstyledButton
                      className={classes.sortButton}
                      data-active={active || undefined}
                      onClick={() => sortByHeader(list, orders)}
                    >
                      {column.header}
                      {active &&
                        (query.sortDir === "asc" ? (
                          <IconArrowUp size={12} />
                        ) : (
                          <IconArrowDown size={12} />
                        ))}
                    </UnstyledButton>
                  ) : column.headerToggle ? (
                    <ToggleIconButton {...column.headerToggle} size="sm" />
                  ) : (
                    column.header
                  )}
                </Table.Th>
              );
            })}
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {list.items.length === 0 && (
            <Table.Tr>
              <Table.Td colSpan={columns.length}>{emptyContent}</Table.Td>
            </Table.Tr>
          )}
          {list.items.map((item) => {
            const key = getRowKey(item);
            const accent = getRowAccent?.(item);
            const expanded = renderExpanded ? renderExpanded(item) : null;
            const isOpen = !!expanded && open.has(key);
            const onClick = onRowClick
              ? () => onRowClick(item)
              : expanded
                ? () => toggleOpen(key)
                : undefined;
            return (
              <Fragment key={key}>
                <Table.Tr
                  className={classes.row}
                  data-clickable={onClick ? true : undefined}
                  data-accent={accent ? true : undefined}
                  aria-expanded={expanded ? isOpen : undefined}
                  style={
                    accent
                      ? ({ "--row-accent": `var(--mantine-color-${accent}-4)` } as CSSProperties)
                      : undefined
                  }
                  onClick={onClick}
                >
                  {columns.map((column) => (
                    <Table.Td
                      key={column.key}
                      visibleFrom={column.visibleFrom}
                      onClick={column.interactive ? (e) => e.stopPropagation() : undefined}
                    >
                      {column.render(item)}
                    </Table.Td>
                  ))}
                </Table.Tr>
                {isOpen && (
                  <Table.Tr className={classes.expandedRow}>
                    <Table.Td colSpan={columns.length}>{expanded}</Table.Td>
                  </Table.Tr>
                )}
              </Fragment>
            );
          })}
        </Table.Tbody>
      </Table>
    </Table.ScrollContainer>
  );
  return (
    <DimmedLoader visible={list.isRefetching} loaderSize="lg">
      {compact ? (
        table
      ) : (
        <Card shadow="sm" p={0} radius="md" withBorder>
          {table}
        </Card>
      )}
    </DimmedLoader>
  );
}

/** The grid layout's columns: as many cards side by side as fit comfortably. */
const GRID_COLS = { base: 1, xs: 2, md: 3, xl: 4 };

function ListFooter<T, F extends string, S extends string>({
  list,
  compact = false,
}: {
  list: List<T, F, S>;
  /** Range and pages only: an embedded list keeps its page size. */
  compact?: boolean;
}) {
  const { t } = useTranslation("common");
  const { query } = list;
  const pages = pageCount(list.total, query.pageSize);
  const from = listOffset(query) + 1;
  const to = Math.min(listOffset(query) + list.items.length, list.total);

  return (
    <Group justify="space-between" gap="sm">
      <HelperText>{t("list.range", { from, to, total: list.total })}</HelperText>
      {pages > 1 && (
        <Pagination
          total={pages}
          value={Math.min(query.page, pages)}
          onChange={list.setPage}
          size="sm"
          siblings={1}
          boundaries={1}
          getControlProps={(control) => ({
            "aria-label": t(`list.pagination.${control}`),
          })}
        />
      )}
      {!compact && (
        <Group gap="xs" wrap="nowrap">
          <HelperText>{t("list.perPage")}</HelperText>
          <Select
            aria-label={t("list.perPage")}
            data={PAGE_SIZES.map(String)}
            value={String(query.pageSize)}
            onChange={(v) => v && list.setPageSize(Number(v))}
            size="xs"
            w={80}
            allowDeselect={false}
          />
        </Group>
      )}
    </Group>
  );
}

/** The namespaces a preset's labels name (`ns:key`), besides common. */
function presetNamespaces(preset: List<unknown>["preset"]): string[] {
  const keys = [
    ...(preset.filters ?? []).map((f) => f.label),
    ...(preset.toggles ?? []).map((o) => o.label),
    preset.multiFilter?.label,
    ...(preset.multiFilter?.options ?? []).map((o) => o.label),
    ...preset.sorts.map((s) => s.label),
  ];
  const namespaces = keys.flatMap((k) => (k && k.includes(":") ? [k.split(":")[0]] : []));
  return [...new Set(namespaces)];
}

/** The orders a column's header sorts by: none, one, or several to cycle. */
function sortOrders<T, S extends string>(column: ListColumn<T, S>): readonly S[] {
  if (!column.sortBy) return [];
  return typeof column.sortBy === "string" ? [column.sortBy] : column.sortBy;
}

/**
 * A click on a sorting header: one order toggles as before; several are
 * cycled, each descending then ascending, then on to the next.
 */
function sortByHeader<T, F extends string, S extends string>(
  list: List<T, F, S>,
  orders: readonly S[],
) {
  if (orders.length === 1) {
    list.toggleSort(orders[0]);
    return;
  }
  const { sortBy, sortDir } = list.query;
  const at = orders.indexOf(sortBy);
  if (at === -1) list.setSort(orders[0], "desc");
  else if (sortDir === "desc") list.setSort(sortBy, "asc");
  else list.setSort(orders[(at + 1) % orders.length], "desc");
}

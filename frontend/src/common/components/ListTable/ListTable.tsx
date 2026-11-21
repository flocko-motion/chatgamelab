import type { CSSProperties, ReactNode } from "react";
import {
  Alert,
  Card,
  Group,
  Pagination,
  Select,
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
  IconMoodEmpty,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useResponsiveDesign } from "@/common/hooks/useResponsiveDesign";
import type { ListQueryControls } from "@/common/hooks/useListQuery";
import type { ServerList } from "@/common/hooks/useServerList";
import { PAGE_SIZES, listOffset, pageCount, type SortDir } from "@/common/lib/listQuery";
import { DimmedLoader } from "@components/LoadingAnimation";
import { HelperText } from "@components/typography";
import { ToggleIconButton } from "@components/buttons";
import {
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
  /** Clicks in this cell don't open the row (buttons, toggles). */
  interactive?: boolean;
  /**
   * A toggle in the header that narrows the list to the column's subject,
   * e.g. a star for "only favorites". A shortcut only: the same toggle is
   * among the list's `toggles`. On a phone (cards, no headers) it sits in the
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

export interface ListFilterOption<F extends string> {
  value: F;
  label: string;
}

export interface ListSortOption<S extends string> {
  by: S;
  dir: SortDir;
  label: string;
}

export interface ListTableProps<T, F extends string, S extends string> {
  list: ServerList<T>;
  controls: ListQueryControls<F, S>;
  columns: ListColumn<T, S>[];
  getRowKey: (item: T) => string;
  /** How an item looks on a phone, where the table becomes cards. */
  renderCard: (item: T) => ReactNode;
  onRowClick?: (item: T) => void;
  /** A colour stripe on the row's left edge, e.g. for sponsored games. */
  getRowAccent?: (item: T) => MantineColor | undefined;
  filters?: ListFilterOption<F>[];
  /** Filters that are each on or off and combine (the query's `only`). */
  toggles?: ListFilterOption<string>[];
  /** The orders offered in the sort menu. */
  sorts?: ListSortOption<S>[];
  /** Actions shown at the right end of the toolbar, e.g. create and import. */
  actions?: ReactNode;
  empty: { title: string; description?: string; icon?: ReactNode };
  errorMessage: string;
}

/**
 * A list the server filters, sorts and pages: toolbar (search, filter, sort),
 * a table with sortable headers (cards on a phone), and a footer with the
 * pages. It renders only; the query lives in `useListQuery`, the page in
 * `useServerList`.
 */
export function ListTable<T, F extends string, S extends string>({
  list,
  controls,
  columns,
  getRowKey,
  renderCard,
  onRowClick,
  getRowAccent,
  filters,
  toggles,
  sorts,
  actions,
  empty,
  errorMessage,
}: ListTableProps<T, F, S>) {
  const { t } = useTranslation("common");
  const { isMobile } = useResponsiveDesign();
  const { query } = controls;

  // Always open: on a phone it takes what the icons leave, on a wider
  // screen up to a comfortable width.
  const search = (
    <div style={{ display: "flex", flex: 1, minWidth: 0, maxWidth: isMobile ? undefined : 320 }}>
      <SearchInput
        value={controls.searchInput}
        onChange={controls.setSearch}
        placeholder={t("search")}
        clearLabel={t("list.clearSearch")}
      />
    </div>
  );
  const sortSelector = sorts && sorts.length > 0 && (
    <SortSelector
      options={sorts.map((s) => ({ value: `${s.by}-${s.dir}`, label: s.label }))}
      value={`${query.sortBy}-${query.sortDir}`}
      onChange={(v) => {
        const match = sorts.find((s) => `${s.by}-${s.dir}` === v);
        if (match) controls.setSort(match.by, match.dir);
      }}
      label={t("games.sort.label")}
    />
  );

  return (
    <div className={classes.root}>
      <Group justify="space-between" wrap={isMobile ? "nowrap" : "wrap"} gap="sm">
        {/* Left: what narrows and orders the list. On a phone (cards, no
            headers) also the header shortcuts, the order and the search, each
            an icon: star, filter, sort, search. There the group takes the
            free space, so an opened search grows only up to the actions. */}
        <Group gap="sm" wrap="nowrap" style={{ minWidth: 0, flex: isMobile ? 1 : undefined }}>
          {isMobile &&
            columns.map(
              (column) =>
                column.headerToggle && (
                  <ToggleIconButton key={column.key} {...column.headerToggle} />
                ),
            )}
          {toggles && toggles.length > 0 && (
            <FilterToggles
              active={query.only}
              onToggle={controls.toggleOnly}
              options={toggles}
              menuLabel={t("list.filter")}
            />
          )}
          {filters && filters.length > 1 && (
            <FilterSegmentedControl<F>
              value={query.filter}
              onChange={controls.setFilter}
              options={filters}
            />
          )}
          {isMobile && sortSelector}
          {isMobile && search}
        </Group>
        {/* Right: the search on a wider screen (its headers sort), then the
            list's actions (create, import). */}
        <Group
          gap="sm"
          wrap="nowrap"
          style={
            isMobile
              ? { flexShrink: 0 }
              : { flex: 1, justifyContent: "flex-end", minWidth: 0 }
          }
        >
          {!isMobile && search}
          {actions}
        </Group>
      </Group>

      <div>
        <ListBody
          list={list}
          controls={controls}
          columns={columns}
          getRowKey={getRowKey}
          renderCard={renderCard}
          onRowClick={onRowClick}
          getRowAccent={getRowAccent}
          empty={empty}
          errorMessage={errorMessage}
          isMobile={isMobile}
        />
      </div>

      {!list.isInitialLoading && !list.error && list.total > 0 && (
        <ListFooter list={list} controls={controls} />
      )}
    </div>
  );
}

function ListBody<T, F extends string, S extends string>({
  list,
  controls,
  columns,
  getRowKey,
  renderCard,
  onRowClick,
  getRowAccent,
  empty,
  errorMessage,
  isMobile,
}: Omit<ListTableProps<T, F, S>, "filters" | "toggles" | "sorts" | "actions"> & {
  isMobile: boolean;
}) {
  const { t } = useTranslation("common");

  if (list.error) {
    return (
      <Alert icon={<IconAlertCircle size={16} />} title={t("error")} color="red">
        {errorMessage}
      </Alert>
    );
  }

  if (list.isInitialLoading) {
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

  // On a phone the toolbar holds every control, so an empty list is a card.
  // On a wider screen the table keeps its header even when empty: its sorts
  // and toggles (e.g. "only favorites") must stay reachable to undo a filter
  // that left nothing.
  if (list.items.length === 0 && isMobile) {
    return (
      <DimmedLoader visible={list.isRefetching} loaderSize="lg">
        <Card shadow="sm" p="xl" radius="md" withBorder>
          {emptyContent}
        </Card>
      </DimmedLoader>
    );
  }

  if (isMobile) {
    return (
      <DimmedLoader visible={list.isRefetching} loaderSize="lg">
        <Stack gap="md">
          {list.items.map((item) => (
            <div key={getRowKey(item)}>{renderCard(item)}</div>
          ))}
        </Stack>
      </DimmedLoader>
    );
  }

  const { query } = controls;
  return (
    <DimmedLoader visible={list.isRefetching} loaderSize="lg">
      <Card shadow="sm" p={0} radius="md" withBorder>
        <Table.ScrollContainer minWidth={600} type="native">
          <Table verticalSpacing="md" horizontalSpacing="lg" stickyHeader highlightOnHover>
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
                          onClick={() => sortByHeader(controls, orders)}
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
                const accent = getRowAccent?.(item);
                return (
                  <Table.Tr
                    key={getRowKey(item)}
                    className={classes.row}
                    data-clickable={onRowClick ? true : undefined}
                    data-accent={accent ? true : undefined}
                    style={
                      accent
                        ? ({ "--row-accent": `var(--mantine-color-${accent}-4)` } as CSSProperties)
                        : undefined
                    }
                    onClick={onRowClick ? () => onRowClick(item) : undefined}
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
                );
              })}
            </Table.Tbody>
          </Table>
        </Table.ScrollContainer>
      </Card>
    </DimmedLoader>
  );
}

function ListFooter<T, F extends string, S extends string>({
  list,
  controls,
}: {
  list: ServerList<T>;
  controls: ListQueryControls<F, S>;
}) {
  const { t } = useTranslation("common");
  const { query } = controls;
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
          onChange={controls.setPage}
          size="sm"
          siblings={1}
          boundaries={1}
          getControlProps={(control) => ({
            "aria-label": t(`list.pagination.${control}`),
          })}
        />
      )}
      <Group gap="xs" wrap="nowrap">
        <HelperText>{t("list.perPage")}</HelperText>
        <Select
          aria-label={t("list.perPage")}
          data={PAGE_SIZES.map(String)}
          value={String(query.pageSize)}
          onChange={(v) => v && controls.setPageSize(Number(v))}
          size="xs"
          w={80}
          allowDeselect={false}
        />
      </Group>
    </Group>
  );
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
function sortByHeader<F extends string, S extends string>(
  controls: ListQueryControls<F, S>,
  orders: readonly S[],
) {
  if (orders.length === 1) {
    controls.toggleSort(orders[0]);
    return;
  }
  const { sortBy, sortDir } = controls.query;
  const at = orders.indexOf(sortBy);
  if (at === -1) controls.setSort(orders[0], "desc");
  else if (sortDir === "desc") controls.setSort(sortBy, "asc");
  else controls.setSort(orders[(at + 1) % orders.length], "desc");
}

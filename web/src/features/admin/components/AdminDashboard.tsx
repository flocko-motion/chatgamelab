import { Fragment, useMemo, useState } from "react";
import {
  Alert,
  Badge,
  Box,
  Card,
  Container,
  Group,
  Loader,
  ScrollArea,
  SegmentedControl,
  SimpleGrid,
  Stack,
  Table,
  Text,
  Title,
  Tooltip,
  UnstyledButton,
} from "@mantine/core";
import {
  IconAlertCircle,
  IconChevronDown,
  IconChevronRight,
  IconInfoCircle,
  IconSelector,
} from "@tabler/icons-react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { useRequiredAuthenticatedApi } from "@/api/useAuthenticatedApi";
import type { ObjAdminStats, ObjInstitutionStats, ObjStatsMonth } from "@/api/generated";

type Metric = "newGames" | "sessions" | "turns";
type SortKey =
  | "name"
  | "workshops"
  | "games"
  | "newGames30"
  | "sessions30"
  | "turns30"
  | "lastActivity";

/** "2026-11" → "Nov. 26" in the UI language. */
function monthLabel(month: string, lang: string): string {
  const [y, m] = month.split("-").map(Number);
  return new Date(y, m - 1, 1).toLocaleDateString(lang, { month: "short", year: "2-digit" });
}

/** A single headline number with its 30-day value underneath. */
function StatTile({ label, value, sub }: { label: string; value: number | string; sub?: string }) {
  return (
    <Card withBorder padding="md" radius="md">
      <Text size="sm" c="dimmed">
        {label}
      </Text>
      <Text fz={32} fw={700} lh={1.2}>
        {value}
      </Text>
      {sub && (
        <Text size="xs" c="dimmed">
          {sub}
        </Text>
      )}
    </Card>
  );
}

/**
 * One measure per month as a single-series bar chart: one hue, bars anchored to the
 * baseline with rounded tops, value on hover, the latest month labelled directly.
 */
function MonthBars({ months, metric, title }: { months: ObjStatsMonth[]; metric: Metric; title: string }) {
  const { i18n } = useTranslation();
  const values = months.map((m) => m[metric] ?? 0);
  const max = Math.max(1, ...values);
  const height = 120;
  return (
    <Card withBorder padding="md" radius="md">
      <Text fw={600} size="sm" mb="sm">
        {title}
      </Text>
      <Box
        role="img"
        aria-label={`${title}: ${months.map((m, i) => `${monthLabel(m.month ?? "", i18n.language)} ${values[i]}`).join(", ")}`}
        style={{
          display: "grid",
          gridTemplateColumns: `repeat(${months.length}, 1fr)`,
          alignItems: "end",
          gap: 2,
          height,
          borderBottom: "1px solid var(--mantine-color-gray-4)",
        }}
      >
        {months.map((m, i) => {
          const v = values[i];
          const h = v === 0 ? 0 : Math.max(2, Math.round((v / max) * (height - 18)));
          const isLast = i === months.length - 1;
          return (
            <Tooltip key={m.month} label={`${monthLabel(m.month ?? "", i18n.language)}: ${v}`} withArrow>
              <Box style={{ display: "flex", flexDirection: "column", justifyContent: "flex-end", height: "100%", cursor: "default" }}>
                {isLast && v > 0 && (
                  <Text size="xs" ta="center" c="dimmed" lh={1.4}>
                    {v}
                  </Text>
                )}
                <Box
                  style={{
                    height: h,
                    background: "var(--mantine-primary-color-filled)",
                    borderRadius: "4px 4px 0 0",
                    marginInline: "15%",
                  }}
                />
              </Box>
            </Tooltip>
          );
        })}
      </Box>
      <Group justify="space-between" mt={4}>
        <Text size="xs" c="dimmed">
          {months[0] ? monthLabel(months[0].month ?? "", i18n.language) : ""}
        </Text>
        <Text size="xs" c="dimmed">
          {months.length ? monthLabel(months[months.length - 1].month ?? "", i18n.language) : ""}
        </Text>
      </Group>
    </Card>
  );
}

/** Month-by-month numbers as a table (the accessible view of the bars). */
function MonthTable({ months, showInstitutions }: { months: ObjStatsMonth[]; showInstitutions?: boolean }) {
  const { t, i18n } = useTranslation("common");
  return (
    <ScrollArea>
      <Table striped withTableBorder miw={420} fz="sm">
        <Table.Thead>
          <Table.Tr>
            <Table.Th>{t("admin.dashboard.month")}</Table.Th>
            {showInstitutions && <Table.Th ta="right">{t("admin.dashboard.newOrganisations")}</Table.Th>}
            <Table.Th ta="right">{t("admin.dashboard.newGames")}</Table.Th>
            <Table.Th ta="right">{t("admin.dashboard.sessions")}</Table.Th>
            <Table.Th ta="right">{t("admin.dashboard.turns")}</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {[...months].reverse().map((m) => (
            <Table.Tr key={m.month}>
              <Table.Td>{monthLabel(m.month ?? "", i18n.language)}</Table.Td>
              {showInstitutions && <Table.Td ta="right">{m.newInstitutions ?? 0}</Table.Td>}
              <Table.Td ta="right">{m.newGames ?? 0}</Table.Td>
              <Table.Td ta="right">{m.sessions ?? 0}</Table.Td>
              <Table.Td ta="right">{m.turns ?? 0}</Table.Td>
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>
    </ScrollArea>
  );
}

/** Sortable column header. */
function SortTh({
  k,
  sort,
  onSort,
  right,
  children,
}: {
  k: SortKey;
  sort: { key: SortKey; desc: boolean };
  onSort: (k: SortKey) => void;
  right?: boolean;
  children: React.ReactNode;
}) {
  return (
    <Table.Th ta={right ? "right" : undefined} aria-sort={sort.key === k ? (sort.desc ? "descending" : "ascending") : "none"}>
      <UnstyledButton onClick={() => onSort(k)} style={{ fontWeight: 600, fontSize: "inherit" }}>
        <Group gap={2} wrap="nowrap" justify={right ? "flex-end" : undefined}>
          {children}
          {sort.key === k ? (sort.desc ? " ↓" : " ↑") : <IconSelector size={12} />}
        </Group>
      </UnstyledButton>
    </Table.Th>
  );
}

function sortValue(o: ObjInstitutionStats, key: SortKey): number | string {
  switch (key) {
    case "name":
      return (o.name ?? "").toLowerCase();
    case "workshops":
      return o.workshops ?? 0;
    case "games":
      return o.games ?? 0;
    case "newGames30":
      return o.newGames?.last30 ?? 0;
    case "sessions30":
      return o.sessions?.last30 ?? 0;
    case "turns30":
      return o.turns?.last30 ?? 0;
    case "lastActivity":
      return o.lastActivity ? new Date(o.lastActivity).getTime() : 0;
  }
}

/** Admin dashboard: platform-wide numbers, monthly trend and one row per organisation. */
export function AdminDashboard() {
  const { t, i18n } = useTranslation("common");
  const api = useRequiredAuthenticatedApi();
  const [chartView, setChartView] = useState<"chart" | "table">("chart");
  const [sort, setSort] = useState<{ key: SortKey; desc: boolean }>({ key: "sessions30", desc: true });
  const [expanded, setExpanded] = useState<string | null>(null);

  const { data, isLoading, error } = useQuery({
    queryKey: ["adminStats"],
    queryFn: async () => (await api.admin.statsList()).data as ObjAdminStats,
  });

  const rows = useMemo(() => {
    const list = [...(data?.perInstitution ?? [])];
    list.sort((a, b) => {
      const va = sortValue(a, sort.key);
      const vb = sortValue(b, sort.key);
      const cmp = va < vb ? -1 : va > vb ? 1 : 0;
      return sort.desc ? -cmp : cmp;
    });
    return list;
  }, [data, sort]);

  if (isLoading) {
    return (
      <Group justify="center" py="xl">
        <Loader />
      </Group>
    );
  }
  if (error || !data) {
    return (
      <Container size="xl" py="xl">
        <Alert color="red" icon={<IconAlertCircle size={16} />}>
          {t("admin.dashboard.loadError")}
        </Alert>
      </Container>
    );
  }

  const date = (iso?: string) => (iso ? new Date(iso).toLocaleDateString(i18n.language) : "–");
  const months = data.months ?? [];
  const sortBy = (key: SortKey) =>
    setSort((s) => (s.key === key ? { key, desc: !s.desc } : { key, desc: key !== "name" }));
  return (
    <Container size="xl" py="xl">
      <Stack gap="lg">
        <Stack gap={4}>
          <Title order={2}>{t("admin.dashboard.title")}</Title>
          <Text size="sm" c="dimmed">
            {t("admin.dashboard.generatedAt", {
              date: new Date(data.generatedAt ?? "").toLocaleString(i18n.language, { dateStyle: "short", timeStyle: "short" }),
            })}
          </Text>
        </Stack>

        <Alert color="blue" variant="light" icon={<IconInfoCircle size={16} />}>
          {t("admin.dashboard.playsSince", { date: date(data.playsRecordedSince) })}
        </Alert>

        <SimpleGrid cols={{ base: 2, md: 4 }}>
          <StatTile
            label={t("admin.dashboard.organisations")}
            value={data.institutions ?? 0}
            sub={t("admin.dashboard.organisationsSub", {
              locked: data.institutionsLocked ?? 0,
              pending: data.consentPending ?? 0,
            })}
          />
          <StatTile
            label={t("admin.dashboard.newGames7")}
            value={data.all?.newGames?.last7 ?? 0}
            sub={t("admin.dashboard.last30", { n: data.all?.newGames?.last30 ?? 0 })}
          />
          <StatTile
            label={t("admin.dashboard.sessions7")}
            value={data.all?.sessions?.last7 ?? 0}
            sub={t("admin.dashboard.last30", { n: data.all?.sessions?.last30 ?? 0 })}
          />
          <StatTile
            label={t("admin.dashboard.turns7")}
            value={data.all?.turns?.last7 ?? 0}
            sub={t("admin.dashboard.last30", { n: data.all?.turns?.last30 ?? 0 })}
          />
        </SimpleGrid>

        <Card withBorder padding="lg" radius="md">
          <Group justify="space-between" mb="md" wrap="wrap">
            <Title order={4}>{t("admin.dashboard.perMonth")}</Title>
            <SegmentedControl
              size="xs"
              value={chartView}
              onChange={(v) => setChartView(v as "chart" | "table")}
              data={[
                { value: "chart", label: t("admin.dashboard.viewChart") },
                { value: "table", label: t("admin.dashboard.viewTable") },
              ]}
            />
          </Group>
          {chartView === "chart" ? (
            <SimpleGrid cols={{ base: 1, md: 3 }}>
              <MonthBars months={months} metric="newGames" title={t("admin.dashboard.newGames")} />
              <MonthBars months={months} metric="sessions" title={t("admin.dashboard.sessions")} />
              <MonthBars months={months} metric="turns" title={t("admin.dashboard.turns")} />
            </SimpleGrid>
          ) : (
            <MonthTable months={months} showInstitutions />
          )}
        </Card>

        <SimpleGrid cols={{ base: 1, md: 2 }}>
          <Card withBorder padding="lg" radius="md">
            <Title order={4} mb="xs">
              {t("admin.dashboard.funnelTitle")}
            </Title>
            <Text size="sm">
              {t("admin.dashboard.funnel", {
                submitted: data.funnel?.submitted ?? 0,
                confirmed: data.funnel?.confirmed ?? 0,
                withWorkshop: data.funnel?.withWorkshop ?? 0,
              })}
            </Text>
          </Card>
          <Card withBorder padding="lg" radius="md">
            <Title order={4} mb="xs">
              {t("admin.dashboard.outsideTitle")}
            </Title>
            <Text size="sm">
              {t("admin.dashboard.outside", {
                games: data.outsideOrganisations?.newGames?.last30 ?? 0,
                sessions: data.outsideOrganisations?.sessions?.last30 ?? 0,
                turns: data.outsideOrganisations?.turns?.last30 ?? 0,
              })}
            </Text>
          </Card>
        </SimpleGrid>

        <Card withBorder padding="lg" radius="md">
          <Title order={4} mb={4}>
            {t("admin.dashboard.perOrganisation")}
          </Title>
          <Text size="sm" c="dimmed" mb="md">
            {t("admin.dashboard.perOrganisationHint")}
          </Text>
          <ScrollArea>
            <Table striped highlightOnHover miw={1000} fz="sm">
              <Table.Thead>
                <Table.Tr>
                  <SortTh k="name" sort={sort} onSort={sortBy}>{t("admin.dashboard.organisation")}</SortTh>
                  <Table.Th>{t("admin.dashboard.members")}</Table.Th>
                  <SortTh k="workshops" sort={sort} onSort={sortBy} right>{t("admin.dashboard.workshops")}</SortTh>
                  <SortTh k="games" sort={sort} onSort={sortBy} right>{t("admin.dashboard.games")}</SortTh>
                  <SortTh k="newGames30" sort={sort} onSort={sortBy} right>{t("admin.dashboard.newGames30")}</SortTh>
                  <SortTh k="sessions30" sort={sort} onSort={sortBy} right>{t("admin.dashboard.sessions30")}</SortTh>
                  <SortTh k="turns30" sort={sort} onSort={sortBy} right>{t("admin.dashboard.turns30")}</SortTh>
                  <SortTh k="lastActivity" sort={sort} onSort={sortBy}>{t("admin.dashboard.lastActivity")}</SortTh>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {rows.map((o) => {
                  const open = expanded === o.id;
                  return (
                    <Fragment key={o.id}>
                      <Table.Tr style={{ cursor: "pointer" }} onClick={() => setExpanded(open ? null : (o.id ?? null))}>
                        <Table.Td>
                          <Group gap={6} wrap="nowrap">
                            {open ? <IconChevronDown size={14} /> : <IconChevronRight size={14} />}
                            <Stack gap={0}>
                              <Text size="sm" fw={500}>
                                {o.name}
                              </Text>
                              <Group gap={4}>
                                {o.city && (
                                  <Text size="xs" c="dimmed">
                                    {o.city}
                                  </Text>
                                )}
                                {o.locked && (
                                  <Badge size="xs" color="red">
                                    {t("admin.organizations.statusLocked")}
                                  </Badge>
                                )}
                                {o.consentPending && (
                                  <Badge size="xs" color="orange" variant="light">
                                    {t("admin.organizations.consentMissing")}
                                  </Badge>
                                )}
                              </Group>
                            </Stack>
                          </Group>
                        </Table.Td>
                        <Table.Td>
                          <Text size="xs">
                            {t("admin.dashboard.membersValue", {
                              heads: o.heads ?? 0,
                              staff: o.staff ?? 0,
                              individuals: o.individuals ?? 0,
                              participants: o.participants ?? 0,
                            })}
                          </Text>
                        </Table.Td>
                        <Table.Td ta="right">
                          {o.workshops ?? 0}
                          <Text span size="xs" c="dimmed">
                            {" "}
                            ({t("admin.dashboard.activeShort", { n: o.activeWorkshops ?? 0 })})
                          </Text>
                        </Table.Td>
                        <Table.Td ta="right">{o.games ?? 0}</Table.Td>
                        <Table.Td ta="right">
                          {o.newGames?.last30 ?? 0}
                          <Text span size="xs" c="dimmed"> / {o.newGames?.last7 ?? 0}</Text>
                        </Table.Td>
                        <Table.Td ta="right">
                          {o.sessions?.last30 ?? 0}
                          <Text span size="xs" c="dimmed"> / {o.sessions?.last7 ?? 0}</Text>
                        </Table.Td>
                        <Table.Td ta="right">
                          {o.turns?.last30 ?? 0}
                          <Text span size="xs" c="dimmed"> / {o.turns?.last7 ?? 0}</Text>
                        </Table.Td>
                        <Table.Td>{date(o.lastActivity)}</Table.Td>
                      </Table.Tr>
                      {open && (
                        <Table.Tr>
                          <Table.Td colSpan={8} bg="gray.0">
                            <MonthTable months={o.months ?? []} />
                          </Table.Td>
                        </Table.Tr>
                      )}
                    </Fragment>
                  );
                })}
              </Table.Tbody>
            </Table>
          </ScrollArea>
        </Card>
      </Stack>
    </Container>
  );
}

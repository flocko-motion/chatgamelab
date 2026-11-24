import { Fragment, useState, type ReactNode } from "react";
import { ActionIcon, Badge, Group, Loader, Stack, Table, Text, Tooltip, UnstyledButton } from "@mantine/core";
import { modals } from "@mantine/modals";
import { IconAlertTriangle, IconArrowBackUp, IconCpu, IconPencil, IconPlus, IconTable } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { TextButton } from "@components/buttons";
import { BodyText, HelperText, Label } from "@components/typography";
import { getAiQualityTierLabel } from "@/common/lib/aiQualityTier";
import type { EngineModelsPlatform, ModelSlotView, ModelTableRowView, Price } from "@/api/types";
import {
  PRICE_FIELDS,
  TIERS,
  useEngineModelsPage,
  type EngineModelsPage,
  type EntryDraft,
  type SlotDraft,
} from "../hooks/useEngineModelsPage";
import { SettingsCard } from "./SettingsCard";
import { AssignModelModal } from "./AssignModelModal";
import { ModelEntryModal } from "./ModelEntryModal";
import classes from "./EngineModels.module.css";

/**
 * The engine's models, for the admin: per platform the model each tier runs
 * each task on, and the model table they come from. Configurable platforms
 * change; the others are shown.
 */
export function EngineModels() {
  const { t } = useTranslation("common");
  const page = useEngineModelsPage();
  const [slotDraft, setSlotDraft] = useState<SlotDraft | null>(null);
  const [entryDraft, setEntryDraft] = useState<EntryDraft | null>(null);

  if (page.loading) return <Loader size="sm" />;

  return (
    <Stack gap="lg">
      {page.platforms.map((p) => (
        <Stack key={p.id} gap="lg">
          <SettingsCard
            icon={<IconCpu size={20} />}
            title={t("serverSettings.models.assignments.title", { platform: p.name })}
            description={t(
              p.configurable ? "serverSettings.models.assignments.description" : "serverSettings.models.fixed",
            )}
            busy={page.saving}
          >
            <Assignments page={page} platform={p} onOpen={(slot) => setSlotDraft({ platform: p.id, slot })} />
            {p.configurable && <ResetAll page={page} platform={p} what="assignments" />}
          </SettingsCard>
          <SettingsCard
            icon={<IconTable size={20} />}
            title={t("serverSettings.models.table.title", { platform: p.name })}
            description={t(p.configurable ? "serverSettings.models.table.description" : "serverSettings.models.fixed")}
          >
            <ModelTable page={page} platform={p} onOpen={(row) => setEntryDraft({ platform: p.id, row })} />
            {p.configurable && (
              <Group justify="space-between">
                <TextButton leftSection={<IconPlus size={16} />} onClick={() => setEntryDraft({ platform: p.id })}>
                  {t("serverSettings.models.table.add")}
                </TextButton>
                <ResetAll page={page} platform={p} what="entries" />
              </Group>
            )}
          </SettingsCard>
        </Stack>
      ))}

      <AssignModelModal
        page={page}
        draft={slotDraft}
        platform={page.platforms.find((p) => p.id === slotDraft?.platform)}
        onClose={() => {
          page.clearErrors();
          setSlotDraft(null);
        }}
      />
      <ModelEntryModal
        page={page}
        draft={entryDraft}
        onClose={() => {
          page.clearErrors();
          setEntryDraft(null);
        }}
      />
    </Stack>
  );
}

/** Gives a platform's tiers, or its model table, back the defaults, after asking. */
function ResetAll({ page, platform, what }: { page: EngineModelsPage; platform: EngineModelsPlatform; what: "assignments" | "entries" }) {
  const { t } = useTranslation("common");
  const ask = () =>
    modals.openConfirmModal({
      title: t(`serverSettings.models.resetAll.${what}.title`, { platform: platform.name }),
      children: <Text size="sm">{t(`serverSettings.models.resetAll.${what}.message`)}</Text>,
      labels: { confirm: t("serverSettings.models.resetAll.confirm"), cancel: t("cancel") },
      onConfirm: () => void page.resetAll(platform.id, what).catch(() => undefined),
    });
  return (
    <Group justify="flex-end">
      <TextButton leftSection={<IconArrowBackUp size={16} />} onClick={ask} disabled={!page.anyChanged(platform, what)}>
        {t("serverSettings.models.resetAll.button")}
      </TextButton>
    </Group>
  );
}

/** A value that differs from the default, highlighted. */
function Changed({ on, children }: { on: boolean; children: ReactNode }) {
  return on ? <span className={classes.changed}>{children}</span> : <>{children}</>;
}

/** One platform's tiers × tasks; a table on wide screens, a list per task on a phone. */
function Assignments({
  page,
  platform,
  onOpen,
}: {
  page: EngineModelsPage;
  platform: EngineModelsPlatform;
  onOpen: (slot: ModelSlotView) => void;
}) {
  const { t } = useTranslation("common");
  const types = page.types(platform);
  const cell = (slot: ModelSlotView | undefined) =>
    slot ? <SlotCell page={page} platform={platform} slot={slot} onOpen={() => onOpen(slot)} /> : "–";

  return (
    <>
      <Table visibleFrom="sm" verticalSpacing="xs" className={classes.grid}>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>{t("serverSettings.models.task")}</Table.Th>
            {TIERS.map((tier) => (
              <Table.Th key={tier}>{getAiQualityTierLabel(t, tier)}</Table.Th>
            ))}
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {types.map((type) => (
            <Table.Tr key={type}>
              <Table.Td>
                <Label>{t(`serverSettings.models.types.${type}`)}</Label>
              </Table.Td>
              {TIERS.map((tier) => (
                <Table.Td key={tier}>{cell(page.slot(platform, tier, type))}</Table.Td>
              ))}
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>

      <Stack hiddenFrom="sm" gap="md">
        {types.map((type) => (
          <Stack key={type} gap={4}>
            <Label>{t(`serverSettings.models.types.${type}`)}</Label>
            {TIERS.map((tier) => (
              <Group key={tier} gap="xs" wrap="nowrap" justify="space-between">
                <HelperText size="xs">{getAiQualityTierLabel(t, tier)}</HelperText>
                {cell(page.slot(platform, tier, type))}
              </Group>
            ))}
          </Stack>
        ))}
      </Stack>
    </>
  );
}

/** A tier's task: its model and a picture's quality, highlighted where they differ from the default. */
function SlotCell({
  page,
  platform,
  slot,
  onOpen,
}: {
  page: EngineModelsPage;
  platform: EngineModelsPlatform;
  slot: ModelSlotView;
  onOpen: () => void;
}) {
  const { t } = useTranslation("common");
  const content = (
    <Group gap={6} wrap="nowrap" className={page.slotChanged(slot) ? classes.changedCell : undefined}>
      <span className={classes.modelName}>
        <Changed on={slot.model !== slot.default}>{slot.model}</Changed>
        {slot.quality && (
          <>
            {" · "}
            <Changed on={slot.quality !== slot.defaultQuality}>{t(`serverSettings.models.quality.${slot.quality}`)}</Changed>
          </>
        )}
      </span>
      {page.unpriced(platform, slot.model) && (
        <Tooltip label={t("serverSettings.models.unpriced")} withArrow multiline w={260}>
          <IconAlertTriangle size={14} color="var(--mantine-color-orange-6)" aria-label={t("serverSettings.models.unpriced")} />
        </Tooltip>
      )}
    </Group>
  );
  if (!platform.configurable) return content;
  return (
    <UnstyledButton
      className={classes.cellButton}
      onClick={onOpen}
      aria-label={t("serverSettings.models.assign.cellLabel", {
        task: t(`serverSettings.models.types.${slot.type}`),
        tier: getAiQualityTierLabel(t, slot.tier),
        model: slot.model,
      })}
    >
      {content}
    </UnstyledButton>
  );
}

/** One platform's model table: each model with the tiers that run it; models no tier runs are greyed. */
function ModelTable({
  page,
  platform,
  onOpen,
}: {
  page: EngineModelsPage;
  platform: EngineModelsPlatform;
  onOpen: (row: ModelTableRowView) => void;
}) {
  const { t } = useTranslation("common");
  return (
    <Stack gap={0} className={classes.rows}>
      {platform.models.map((row) => {
        const changed = page.entryChanged(row);
        const media = (list: string[]) => list.map((m) => t(`serverSettings.models.media.${m}`)).join(", ");
        return (
        <div key={row.model} className={classes.row} data-unused={!row.inUse || undefined}>
          <Stack gap={2} className={classes.rowMain}>
            <Group gap={6} wrap="wrap">
              <span className={classes.modelName}>
                <Changed on={changed.added}>{row.model}</Changed>
              </span>
              {page.usedBy(platform, row.model).map((use) => (
                <Tooltip
                  key={use.tier}
                  withArrow
                  label={use.types.map((type) => t(`serverSettings.models.types.${type}`)).join(", ")}
                >
                  <Badge size="xs" variant="filled" color="ink">
                    {getAiQualityTierLabel(t, use.tier)}
                  </Badge>
                </Tooltip>
              ))}
              {!row.inUse && (
                <Badge size="xs" variant="outline" color="gray">
                  {t("serverSettings.models.table.unused")}
                </Badge>
              )}
              {!row.default && (
                <Badge size="xs" variant="light" color="highlight">
                  {t("serverSettings.models.table.added")}
                </Badge>
              )}
              {row.stale && (
                <Badge size="xs" variant="light" color="gray">
                  {t("serverSettings.models.table.stale")}
                </Badge>
              )}
            </Group>
            <HelperText size="xs">
              {t("serverSettings.models.table.takes")} <Changed on={changed.takes}>{media(row.facts.in)}</Changed>
              {" · "}
              {t("serverSettings.models.table.makes")} <Changed on={changed.makes}>{media(row.facts.out)}</Changed>
              {row.facts.window ? (
                <>
                  {" · "}
                  <Changed on={changed.window}>
                    {t("serverSettings.models.table.windowValue", { tokens: row.facts.window.toLocaleString() })}
                  </Changed>
                </>
              ) : null}
            </HelperText>
            <BodyText size="sm">
              <PriceLine price={row.facts.price} changed={changed.price} />
            </BodyText>
            <HelperText size="xs">
              {t("serverSettings.models.table.read", { date: row.facts.read, source: row.facts.source })}
            </HelperText>
          </Stack>
          {platform.configurable && (
            <Tooltip label={t("serverSettings.models.table.edit")} withArrow>
              <ActionIcon
                variant="subtle"
                color="ink"
                onClick={() => onOpen(row)}
                aria-label={`${t("serverSettings.models.table.edit")}: ${row.model}`}
              >
                <IconPencil size={16} />
              </ActionIcon>
            </Tooltip>
          )}
        </div>
        );
      })}
    </Stack>
  );
}

/** A price as one line: every rate the model has, in dollars, each highlighted where it differs from the code's. */
function PriceLine({ price, changed }: { price: Price; changed: (field: keyof Price) => boolean }) {
  const { t } = useTranslation("common");
  const fields = PRICE_FIELDS.filter((f) => price[f] || changed(f));
  if (fields.length === 0) return <>{t("serverSettings.models.price.none")}</>;
  return (
    <>
      {fields.map((f, i) => (
        <Fragment key={f}>
          {i > 0 && " · "}
          <Changed on={changed(f)}>{t(`serverSettings.models.price.${f}`, { value: `$${price[f] ?? 0}` })}</Changed>
        </Fragment>
      ))}
    </>
  );
}

import { useState } from "react";
import { Alert, Chip, Group, Modal, NumberInput, SimpleGrid, Stack, TextInput } from "@mantine/core";
import { useTranslation } from "react-i18next";
import { ActionButton, CancelButton, TextButton } from "@components/buttons";
import { BodyText, HelperText, Label } from "@components/typography";
import type { Medium, Price } from "@/api/types";
import { MEDIA, PRICE_FIELDS, type EngineModelsPage, type EntryDraft } from "../hooks/useEngineModelsPage";

/**
 * A model's entry in the model table: what it takes and makes, its prices
 * and its context window; a new model when the draft names no row. A code
 * model goes back to the code's figures, a model the admin added leaves the
 * table, but not while a tier runs it.
 */
export function ModelEntryModal({
  page,
  draft,
  onClose,
}: {
  page: EngineModelsPage;
  draft: EntryDraft | null;
  onClose: () => void;
}) {
  const { t } = useTranslation("common");
  return (
    <Modal
      opened={!!draft}
      onClose={onClose}
      size="lg"
      centered
      title={draft?.row ? draft.row.model : t("serverSettings.models.table.add")}
    >
      {draft && <EntryForm key={draft.row?.model ?? "new"} page={page} draft={draft} onClose={onClose} />}
    </Modal>
  );
}

function EntryForm({ page, draft, onClose }: { page: EngineModelsPage; draft: EntryDraft; onClose: () => void }) {
  const { t } = useTranslation("common");
  const row = draft.row;
  const facts = row?.facts;
  const [model, setModel] = useState(row?.model ?? "");
  const [takes, setTakes] = useState<Medium[]>(facts?.in ?? ["text"]);
  const [makes, setMakes] = useState<Medium[]>(facts?.out ?? ["text"]);
  const [price, setPrice] = useState<Price>(facts?.price ?? {});
  const [window, setWindow] = useState<number>(facts?.window ?? 0);

  // A code model resets to the code's figures; an added one leaves the
  // table, but not while a tier runs it.
  const resettable = row && (row.overridden || row.stale || (!row.default && !row.inUse));

  const save = async () => {
    try {
      await page.saveEntry(draft.platform, model.trim(), { in: takes, out: makes, price, window: window || undefined });
      onClose();
    } catch {
      // the refusal shows below
    }
  };
  const reset = async () => {
    if (!row) return;
    try {
      await page.resetEntry(draft.platform, row.model);
      onClose();
    } catch {
      // the refusal shows below
    }
  };

  return (
    <Stack gap="md">
      {!row && (
        <TextInput
          label={t("serverSettings.models.entry.model")}
          description={t("serverSettings.models.entry.modelHint")}
          value={model}
          onChange={(e) => setModel(e.currentTarget.value)}
        />
      )}

      <SimpleGrid cols={{ base: 1, sm: 2 }}>
        <Stack gap={4}>
          <Label>{t("serverSettings.models.entry.takes")}</Label>
          <Chip.Group multiple value={takes} onChange={(v) => setTakes(v as Medium[])}>
            <Group gap="xs">
              {MEDIA.map((m) => (
                <Chip key={m} value={m} size="xs">
                  {t(`serverSettings.models.media.${m}`)}
                </Chip>
              ))}
            </Group>
          </Chip.Group>
        </Stack>
        <Stack gap={4}>
          <Label>{t("serverSettings.models.entry.makes")}</Label>
          <Chip.Group multiple value={makes} onChange={(v) => setMakes(v as Medium[])}>
            <Group gap="xs">
              {MEDIA.map((m) => (
                <Chip key={m} value={m} size="xs">
                  {t(`serverSettings.models.media.${m}`)}
                </Chip>
              ))}
            </Group>
          </Chip.Group>
        </Stack>
      </SimpleGrid>

      <Stack gap={4}>
        <Label>{t("serverSettings.models.entry.prices")}</Label>
        <HelperText size="xs">{t("serverSettings.models.entry.pricesHint")}</HelperText>
        <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="sm" verticalSpacing="xs">
          {PRICE_FIELDS.map((field) => (
            <NumberInput
              key={field}
              size="sm"
              label={t(`serverSettings.models.entry.price.${field}`)}
              prefix="$"
              min={0}
              decimalScale={4}
              value={price[field] ?? ""}
              onChange={(v) => setPrice((p) => ({ ...p, [field]: typeof v === "number" ? v : undefined }))}
            />
          ))}
        </SimpleGrid>
      </Stack>

      <NumberInput
        label={t("serverSettings.models.entry.window")}
        description={t("serverSettings.models.entry.windowHint")}
        min={0}
        step={1000}
        thousandSeparator=","
        value={window || ""}
        onChange={(v) => setWindow(typeof v === "number" ? v : 0)}
      />

      {row?.default && (
        <HelperText size="xs">
          {t("serverSettings.models.entry.codeRead", { date: row.default.read, source: row.default.source })}
        </HelperText>
      )}

      {page.entryError !== null && (
        <Alert color="red" title={t("serverSettings.models.refused")}>
          <BodyText size="sm">{page.entryError}</BodyText>
        </Alert>
      )}

      <Group justify="space-between" wrap="wrap">
        {resettable ? (
          <TextButton onClick={() => void reset()}>
            {t(row?.default ? "serverSettings.models.entry.reset" : "serverSettings.models.entry.remove")}
          </TextButton>
        ) : (
          <span />
        )}
        <Group gap="sm">
          <CancelButton onClick={onClose}>{t("cancel")}</CancelButton>
          <ActionButton onClick={() => void save()} loading={page.saving} disabled={!model.trim()}>
            {t("save")}
          </ActionButton>
        </Group>
      </Group>
    </Stack>
  );
}

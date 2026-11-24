import { useState } from "react";
import { Alert, Group, Modal, Select, Stack } from "@mantine/core";
import { IconCircleCheck, IconCircleX, IconPlayerPlay } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { ActionButton, CancelButton, GenericButtonWithText, TextButton } from "@components/buttons";
import { BodyText, HelperText } from "@components/typography";
import { getAiQualityTierLabel } from "@/common/lib/aiQualityTier";
import type { EngineModelsPlatform } from "@/api/types";
import type { EngineModelsPage, SlotDraft } from "../hooks/useEngineModelsPage";

/**
 * Assigns a model to a tier's task: a model of the table that fits the task,
 * probed before it is saved if the admin likes; or the task's default back.
 */
export function AssignModelModal({
  page,
  draft,
  platform,
  onClose,
}: {
  page: EngineModelsPage;
  draft: SlotDraft | null;
  platform?: EngineModelsPlatform;
  onClose: () => void;
}) {
  const { t } = useTranslation("common");
  const slot = draft?.slot;
  return (
    <Modal opened={!!draft} onClose={onClose} title={t("serverSettings.models.assign.title")} centered>
      {draft && slot && platform && (
        <Stack gap="sm">
          <BodyText size="sm">
            {t("serverSettings.models.assign.where", {
              task: t(`serverSettings.models.types.${slot.type}`),
              tier: getAiQualityTierLabel(t, slot.tier),
            })}
          </BodyText>
          <AssignForm key={`${slot.tier}/${slot.type}`} page={page} draft={draft} platform={platform} onClose={onClose} />
        </Stack>
      )}
    </Modal>
  );
}

function AssignForm({
  page,
  draft,
  platform,
  onClose,
}: {
  page: EngineModelsPage;
  draft: SlotDraft;
  platform: EngineModelsPlatform;
  onClose: () => void;
}) {
  const { t } = useTranslation("common");
  const { slot } = draft;
  const [model, setModel] = useState(slot.model);
  const choices = page.choices(platform, slot.type);
  const result = page.probeResult(platform.id, slot.type, model);

  const save = async () => {
    try {
      await page.assign(platform.id, slot, model);
      onClose();
    } catch {
      // the refusal shows below
    }
  };
  const reset = async () => {
    try {
      await page.resetSlot(platform.id, slot);
      onClose();
    } catch {
      // the refusal shows below
    }
  };

  return (
    <Stack gap="md">
      <Select
        label={t("serverSettings.models.assign.model")}
        description={t("serverSettings.models.assign.modelHint", { model: slot.default })}
        data={choices}
        value={model}
        onChange={(v) => v && setModel(v)}
        searchable
        allowDeselect={false}
      />

      <Group gap="sm" wrap="wrap">
        {page.probeable(slot.type) ? (
          <GenericButtonWithText
            icon={<IconPlayerPlay size={16} />}
            variant="default"
            size="sm"
            loading={page.isProbing(platform.id, slot.type, model)}
            onClick={() => void page.probe(platform.id, slot.type, model)}
          >
            {t("serverSettings.models.probe.button")}
          </GenericButtonWithText>
        ) : (
          <HelperText size="xs">{t("serverSettings.models.probe.none")}</HelperText>
        )}
        {result &&
          (result.answers ? (
            <Group gap={4}>
              <IconCircleCheck size={16} color="var(--mantine-color-green-6)" />
              <HelperText size="xs" c="green">
                {t("serverSettings.models.probe.answers", { cost: result.cost.toFixed(4) })}
              </HelperText>
            </Group>
          ) : (
            <Group gap={4} wrap="nowrap">
              <IconCircleX size={16} color="var(--mantine-color-red-6)" />
              <HelperText size="xs" c="red">
                {t("serverSettings.models.probe.failed", { message: result.message ?? "" })}
              </HelperText>
            </Group>
          ))}
      </Group>

      {page.assignError !== null && (
        <Alert color="red" title={t("serverSettings.models.refused")}>
          <BodyText size="sm">{page.assignError}</BodyText>
        </Alert>
      )}

      <Group justify="space-between" wrap="wrap">
        {slot.assigned ? (
          <TextButton onClick={() => void reset()}>{t("serverSettings.models.assign.reset")}</TextButton>
        ) : (
          <span />
        )}
        <Group gap="sm">
          <CancelButton onClick={onClose}>{t("cancel")}</CancelButton>
          <ActionButton onClick={() => void save()} loading={page.saving} disabled={model === slot.model}>
            {t("save")}
          </ActionButton>
        </Group>
      </Group>
    </Stack>
  );
}

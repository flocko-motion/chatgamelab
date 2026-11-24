import { Badge, Group, Select, Stack } from "@mantine/core";
import { IconCircleCheck, IconCircleMinus, IconCircleX, IconKey, IconSparkles } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { DangerButton } from "@components/buttons";
import { HelperText } from "@components/typography";
import { getAiQualityTierOptions, normalizeAiQualityTier } from "@/common/lib/aiQualityTier";
import type { ServerSettingsState } from "../hooks/useServerSettings";
import { SettingsCard } from "./SettingsCard";

/** The site's AI tiers and its free-use key, for the admin. */
export function AiSettings({ state }: { state: ServerSettingsState }) {
  const { t } = useTranslation("common");
  const { settings, keyOptions, hasFreeUseKey, setFreeUseKey, update } = state;

  return (
    <Stack gap="lg">
      <SettingsCard
        icon={<IconSparkles size={20} />}
        title={t("serverSettings.defaultTier.title")}
        description={t("serverSettings.defaultTier.description")}
        busy={update.isPending}
      >
        <Select
          data={getAiQualityTierOptions(t)}
          value={normalizeAiQualityTier(settings?.defaultAiQualityTier) || "medium"}
          onChange={(value) => value && update.mutate({ defaultAiQualityTier: value })}
          disabled={update.isPending}
          size="sm"
          styles={{ wrapper: { maxWidth: 320 } }}
          description={t("aiQualityTier.newSessionsOnly")}
          inputWrapperOrder={["label", "input", "description", "error"]}
        />
      </SettingsCard>

      <SettingsCard
        icon={<IconKey size={20} />}
        title={t("serverSettings.freeUseKey.title")}
        description={t("serverSettings.freeUseKey.description")}
        busy={setFreeUseKey.isPending}
      >
        {hasFreeUseKey && settings && (
          <Group gap="sm" wrap="wrap">
            <Badge color="cyan" variant="light" size="lg">
              {settings.freeUseApiKeyName || "?"} ({settings.freeUseApiKeyPlatform || "?"})
            </Badge>
            <KeyHealth working={settings.freeUseApiKeyWorking} />
            <DangerButton size="xs" variant="outline" onClick={() => setFreeUseKey.mutate({ apiKeyId: null })}>
              {t("serverSettings.freeUseKey.remove")}
            </DangerButton>
          </Group>
        )}

        <Select
          placeholder={
            hasFreeUseKey
              ? t("serverSettings.freeUseKey.replacePlaceholder")
              : t("serverSettings.freeUseKey.selectPlaceholder")
          }
          data={keyOptions}
          value={null}
          onChange={(value) => value && setFreeUseKey.mutate({ apiKeyId: value })}
          disabled={keyOptions.length === 0 || setFreeUseKey.isPending}
          size="sm"
          styles={{ wrapper: { maxWidth: 420 } }}
          description={keyOptions.length === 0 ? t("serverSettings.freeUseKey.noKeys") : undefined}
          inputWrapperOrder={["label", "input", "description", "error"]}
        />

        <Select
          label={t("serverSettings.freeUseTier.title")}
          description={t("serverSettings.freeUseTier.description")}
          data={getAiQualityTierOptions(t, { includeEmpty: true })}
          value={normalizeAiQualityTier(settings?.freeUseAiQualityTier)}
          onChange={(value) => update.mutate({ freeUseAiQualityTier: value || "" })}
          disabled={update.isPending}
          size="sm"
          styles={{ wrapper: { maxWidth: 320 } }}
        />
        <HelperText size="xs">{t("aiQualityTier.newSessionsOnly")}</HelperText>
      </SettingsCard>
    </Stack>
  );
}

/** Whether the free-use key worked when last used. */
function KeyHealth({ working }: { working?: boolean }) {
  const { t } = useTranslation("common");
  if (working === true)
    return (
      <Group gap={4}>
        <IconCircleCheck size={16} color="var(--mantine-color-green-6)" />
        <HelperText size="xs" c="green">{t("serverSettings.freeUseKey.working")}</HelperText>
      </Group>
    );
  if (working === false)
    return (
      <Group gap={4}>
        <IconCircleX size={16} color="var(--mantine-color-red-6)" />
        <HelperText size="xs" c="red">{t("serverSettings.freeUseKey.notWorking")}</HelperText>
      </Group>
    );
  return (
    <Group gap={4}>
      <IconCircleMinus size={16} color="var(--mantine-color-gray-5)" />
      <HelperText size="xs">{t("serverSettings.freeUseKey.untested")}</HelperText>
    </Group>
  );
}

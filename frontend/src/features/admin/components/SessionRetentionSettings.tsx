import { NumberInput } from "@mantine/core";
import { IconClockHour4 } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { ServerSettingsState } from "../hooks/useServerSettings";
import { SettingsCard } from "./SettingsCard";

/** How long the site keeps a session not played, in hours, for the admin; 0 keeps every session. */
export function SessionRetentionSettings({ state }: { state: ServerSettingsState }) {
  const { t } = useTranslation("common");
  const { settings, update } = state;
  const hours = settings?.sessionRetentionHours ?? 48;

  const save = (raw: string) => {
    const value = Number(raw);
    if (raw.trim() === "" || !Number.isInteger(value) || value < 0 || value === hours) return;
    update.mutate({ sessionRetentionHours: value });
  };

  return (
    <SettingsCard
      icon={<IconClockHour4 size={20} />}
      title={t("serverSettings.sessionRetention.title")}
      description={t("serverSettings.sessionRetention.description")}
      busy={update.isPending}
    >
      <NumberInput
        key={hours}
        label={t("serverSettings.sessionRetention.label")}
        description={t("serverSettings.sessionRetention.hint")}
        defaultValue={hours}
        min={0}
        step={1}
        allowDecimal={false}
        allowNegative={false}
        disabled={update.isPending}
        onBlur={(event) => save(event.currentTarget.value)}
        size="sm"
        styles={{ wrapper: { maxWidth: 160 } }}
        inputWrapperOrder={["label", "input", "description", "error"]}
      />
    </SettingsCard>
  );
}

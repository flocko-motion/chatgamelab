import { Textarea } from "@mantine/core";
import { IconShieldLock } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { ServerSettingsState } from "../hooks/useServerSettings";
import { SettingsCard } from "./SettingsCard";

const AGE_GROUPS = [
  { id: "u13", field: "promptConstraintU13" },
  { id: "u13p", field: "promptConstraintU13p" },
  { id: "u18", field: "promptConstraintU18" },
] as const;

/** The site's youth-protection rules per age group, for the admin. */
export function YouthProtectionSettings({ state }: { state: ServerSettingsState }) {
  const { t } = useTranslation("common");
  const { settings, update } = state;

  return (
    <SettingsCard
      icon={<IconShieldLock size={20} />}
      title={t("serverSettings.constraints.title")}
      description={t("serverSettings.constraints.description")}
      busy={update.isPending}
    >
      {AGE_GROUPS.map(({ id, field }) => {
        const stored = settings?.[field] || "";
        return (
          <Textarea
            key={`${id}-${stored}`}
            label={t(`serverSettings.constraints.${id}Label`)}
            placeholder={t(`serverSettings.constraints.${id}Placeholder`)}
            description={t(`serverSettings.constraints.${id}Hint`)}
            size="sm"
            minRows={3}
            maxRows={6}
            autosize
            maxLength={500}
            defaultValue={stored}
            disabled={update.isPending}
            onBlur={(event) => {
              const next = event.currentTarget.value;
              if (next !== stored) update.mutate({ [field]: next });
            }}
          />
        );
      })}
    </SettingsCard>
  );
}

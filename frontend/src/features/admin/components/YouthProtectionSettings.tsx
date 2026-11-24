import { ActionIcon, Badge, Group, Textarea, Tooltip } from "@mantine/core";
import { IconArrowBackUp, IconShieldLock } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { RULE_LEVELS, type ServerSettingsState } from "../hooks/useServerSettings";
import { SettingsCard } from "./SettingsCard";

/**
 * The site's youth-protection rules per age group, for the admin: each is
 * the site's default until the admin changes it, and goes back to it.
 */
export function YouthProtectionSettings({ state }: { state: ServerSettingsState }) {
  const { t } = useTranslation("common");
  const { update } = state;

  return (
    <SettingsCard
      icon={<IconShieldLock size={20} />}
      title={t("serverSettings.constraints.title")}
      description={t("serverSettings.constraints.description")}
      busy={update.isPending}
    >
      {RULE_LEVELS.map((level) => {
        const rule = state.rule(level);
        return (
          <Textarea
            key={`${level}-${rule.value}`}
            label={
              <Group gap="xs" component="span">
                {t(`serverSettings.constraints.${level}Label`)}
                {rule.changed && (
                  <Badge size="xs" variant="light" color="accent">
                    {t("serverSettings.constraints.changed")}
                  </Badge>
                )}
              </Group>
            }
            description={t(`serverSettings.constraints.${level}Hint`)}
            size="sm"
            minRows={3}
            maxRows={6}
            autosize
            maxLength={500}
            defaultValue={rule.value}
            disabled={update.isPending}
            onBlur={(event) => state.setRule(level, event.currentTarget.value)}
            rightSectionPointerEvents="all"
            rightSection={
              rule.changed && (
                <Tooltip label={t("serverSettings.constraints.reset")} withArrow>
                  <ActionIcon
                    variant="subtle"
                    color="gray"
                    size="sm"
                    aria-label={t("serverSettings.constraints.reset")}
                    onClick={() => state.resetRule(level)}
                  >
                    <IconArrowBackUp size={16} />
                  </ActionIcon>
                </Tooltip>
              )
            }
          />
        );
      })}
    </SettingsCard>
  );
}

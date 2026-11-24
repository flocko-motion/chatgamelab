import {
  useApiKeys,
  useSetSystemFreeUseKey,
  useSystemSettings,
  useUpdateSystemSettings,
} from "@/api/hooks";

/** The site's age groups with a youth-protection rule of their own. */
export const RULE_LEVELS = ["u13", "u13p", "u18"] as const;
export type RuleLevel = (typeof RULE_LEVELS)[number];

const RULE_FIELD = {
  u13: "promptConstraintU13",
  u13p: "promptConstraintU13p",
  u18: "promptConstraintU18",
} as const;

/**
 * The site's settings for the admin: AI tiers, the free-use key and the
 * youth-protection rules, with the admin's own keys to choose the free-use
 * key from.
 */
export function useServerSettings() {
  const { data: settings, isLoading: settingsLoading } = useSystemSettings();
  const { data: apiKeys, isLoading: keysLoading } = useApiKeys();
  const setFreeUseKey = useSetSystemFreeUseKey();
  const update = useUpdateSystemSettings();

  const keyOptions = (apiKeys?.apiKeys ?? []).map((key) => ({
    value: key.id!,
    label: `${key.name || ""} (${key.platform || ""})`,
  }));

  /** A level's youth-protection rule as it applies, its default, and whether the admin changed it. */
  const rule = (level: RuleLevel) => {
    const value = settings?.[RULE_FIELD[level]] ?? "";
    const def = settings?.promptConstraintDefaults?.[level] ?? "";
    return { value, default: def, changed: !!value && value !== def };
  };
  /** Sets a level's rule; the default text, or none, gives back the default. */
  const setRule = (level: RuleLevel, text: string) => {
    if (text.trim() === rule(level).value) return;
    update.mutate({ [RULE_FIELD[level]]: text });
  };
  const resetRule = (level: RuleLevel) => update.mutate({ [RULE_FIELD[level]]: "" });

  return {
    loading: settingsLoading || keysLoading,
    rule,
    setRule,
    resetRule,
    settings,
    keyOptions,
    hasFreeUseKey: !!settings?.freeUseApiKeyId,
    setFreeUseKey,
    update,
  };
}

export type ServerSettingsState = ReturnType<typeof useServerSettings>;

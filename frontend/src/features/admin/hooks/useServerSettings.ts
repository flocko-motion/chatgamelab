import {
  useApiKeys,
  useSetSystemFreeUseKey,
  useSystemSettings,
  useUpdateSystemSettings,
} from "@/api/hooks";

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

  return {
    loading: settingsLoading || keysLoading,
    settings,
    keyOptions,
    hasFreeUseKey: !!settings?.freeUseApiKeyId,
    setFreeUseKey,
    update,
  };
}

export type ServerSettingsState = ReturnType<typeof useServerSettings>;

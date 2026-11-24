import { useCallback, useMemo, useState } from "react";
import {
  useApiKeys,
  useCheckApiKey,
  useCreateApiKey,
  useDeleteApiKey,
  useOwnApiKeys,
  usePlatforms,
  useSetDefaultApiKey,
  useUpdateUser,
  type OwnApiKey,
} from "@/api/hooks";
import type { PlatformInfo } from "@/api/types";
import { isListNarrowed } from "@/common/lib/listNarrowed";
import { useAuth } from "@/providers/AuthProvider";
import { ownKeyListPreset, type KeyPlatform } from "../lib/ownKeyListPresets";

interface CreateErrors {
  name?: string;
  platform?: string;
  key?: string;
}

/** A key name that says whose key it is, for which platform, and since when. */
function defaultKeyName(platformName: string, userName: string | undefined): string {
  const now = new Date();
  const date = [now.getDate(), now.getMonth() + 1]
    .map((n) => String(n).padStart(2, "0"))
    .concat(String(now.getFullYear()).slice(-2))
    .join(".");
  return [userName, platformName, date].filter(Boolean).join(" ");
}

/**
 * Everything the key page does, for ApiKeyManagement to render: the caller's
 * keys as one list, the default key and quality tier, and adding, checking,
 * deleting and choosing the default key. A key added is checked at once.
 */
export function useApiKeyManagement(t: (key: string) => string) {
  const { backendUser, retryBackendFetch } = useAuth();
  const { data: platforms } = usePlatforms();
  const { data: keysData } = useApiKeys();

  // ── Platforms: those that take keys, and those with stored keys only ────
  const keyPlatforms = useMemo<(KeyPlatform & { supported: boolean; info?: PlatformInfo })[]>(() => {
    const supported = (platforms ?? [])
      .filter((p) => p.userKeys)
      .map((p) => ({ id: p.id, name: p.name || p.id, supported: true, info: p }));
    const storedOnly = [...new Set((keysData?.apiKeys ?? []).map((k) => k.platform))]
      .filter((id) => id && !supported.some((p) => p.id === id))
      .map((id) => {
        const info = platforms?.find((p) => p.id === id);
        return { id, name: info?.name || id, supported: false, info };
      });
    return [...supported, ...storedOnly];
  }, [platforms, keysData]);

  const platformOf = (id: string) => keyPlatforms.find((p) => p.id === id);

  // ── The list ────────────────────────────────────────────────────────────
  const platformKey = keyPlatforms.map((p) => `${p.id}:${p.name}`).join(",");
  const preset = useMemo(
    () => ownKeyListPreset(keyPlatforms),
    // The preset changes only when the platforms do.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [platformKey],
  );
  const list = useOwnApiKeys(preset);
  const narrowed = isListNarrowed(list);

  // ── The default key and the quality tier ────────────────────────────────
  const defaultKey = keysData?.apiKeys.find((k) => k.isDefault);
  const defaultPlatform = defaultKey ? platformOf(defaultKey.platform)?.info : undefined;

  const updateUser = useUpdateUser();
  const setQualityTier = (tier: string | null) => {
    if (!backendUser?.id) return;
    updateUser.mutate(
      { id: backendUser.id, request: { aiQualityTier: tier || "" } },
      { onSuccess: () => retryBackendFetch() },
    );
  };

  const setDefaultApiKey = useSetDefaultApiKey();
  const makeDefault = (item: OwnApiKey) => {
    if (item.selfShare) setDefaultApiKey.mutate({ id: item.selfShare.id });
  };

  // ── Checking ────────────────────────────────────────────────────────────
  // A check runs per key; when it is done the key's status shows again, with
  // a flash, so a check that found the same status is seen to have run.
  const checkApiKey = useCheckApiKey();
  const [checking, setChecking] = useState<ReadonlySet<string>>(() => new Set());
  const [checkedAt, setCheckedAt] = useState<Readonly<Record<string, number>>>({});
  const runCheck = useCallback(
    async (shareId: string, keyId: string) => {
      setChecking((prev) => new Set(prev).add(keyId));
      try {
        await checkApiKey.mutateAsync({ id: shareId });
      } catch {
        // shown by the mutation's error handler; the status stays as it was
      } finally {
        setChecking((prev) => {
          const next = new Set(prev);
          next.delete(keyId);
          return next;
        });
        setCheckedAt((prev) => ({ ...prev, [keyId]: Date.now() }));
      }
    },
    [checkApiKey],
  );
  const checkKey = (item: OwnApiKey) => {
    if (item.selfShare) void runCheck(item.selfShare.id, item.key.id);
  };

  // ── Adding ──────────────────────────────────────────────────────────────
  const createApiKey = useCreateApiKey();
  const [creating, setCreating] = useState(false);
  const [createName, setCreateName] = useState("");
  const [createPlatform, setCreatePlatform] = useState("");
  const [createKey, setCreateKey] = useState("");
  const [createErrors, setCreateErrors] = useState<CreateErrors>({});

  const choosePlatform = (id: string) => {
    setCreatePlatform(id);
    setCreateName(defaultKeyName(platformOf(id)?.name ?? id, backendUser?.name));
  };

  const openCreate = () => {
    const first = keyPlatforms.find((p) => p.supported);
    setCreateKey("");
    setCreateErrors({});
    if (first) choosePlatform(first.id);
    setCreating(true);
  };

  const submitCreate = async () => {
    const errors: CreateErrors = {};
    if (!createName.trim()) errors.name = t("apiKeys.errors.nameRequired");
    if (!createPlatform) errors.platform = t("apiKeys.errors.platformRequired");
    if (!createKey.trim()) errors.key = t("apiKeys.errors.keyRequired");
    setCreateErrors(errors);
    if (Object.keys(errors).length > 0) return;
    try {
      const created = await createApiKey.mutateAsync({ name: createName, platform: createPlatform, key: createKey });
      setCreating(false);
      // A new key is checked at once, so its owner learns now whether it works.
      void runCheck(created.id, created.apiKeyId);
    } catch {
      // shown by the mutation's error handler
    }
  };

  // ── Deleting ────────────────────────────────────────────────────────────
  const deleteApiKey = useDeleteApiKey();
  const [deleting, setDeleting] = useState<OwnApiKey | null>(null);
  const confirmDelete = async () => {
    if (!deleting?.selfShare) return;
    try {
      await deleteApiKey.mutateAsync({ id: deleting.selfShare.id, cascade: true });
      setDeleting(null);
    } catch {
      // shown by the mutation's error handler
    }
  };

  return {
    list,
    narrowed,
    platformOf,
    defaultKey,
    defaultKeyPlatformName: defaultKey ? (platformOf(defaultKey.platform)?.name ?? defaultKey.platform) : undefined,
    qualityTier: {
      value: backendUser?.aiQualityTier,
      platform: defaultPlatform,
      set: setQualityTier,
      saving: updateUser.isPending,
    },
    makeDefault,
    makingDefault: setDefaultApiKey.isPending,
    checkKey,
    /** Whether a key is being checked, by its id. */
    isChecking: (item: OwnApiKey) => checking.has(item.key.id),
    /** When a key's last check here finished, by its id: a new value flashes its status. */
    checkedAt: (item: OwnApiKey) => checkedAt[item.key.id],
    openCreate,
    canCreate: keyPlatforms.some((p) => p.supported),
    createModal: {
      opened: creating,
      close: () => setCreating(false),
      platforms: keyPlatforms.filter((p) => p.supported),
      tiers: platformOf(createPlatform)?.info?.tiers ?? [],
      name: createName,
      setName: setCreateName,
      platform: createPlatform,
      setPlatform: choosePlatform,
      key: createKey,
      setKey: setCreateKey,
      errors: createErrors,
      submit: submitCreate,
      loading: createApiKey.isPending,
    },
    deleteModal: {
      item: deleting,
      opened: !!deleting,
      open: setDeleting,
      close: () => setDeleting(null),
      confirm: confirmDelete,
      loading: deleteApiKey.isPending,
    },
  };
}

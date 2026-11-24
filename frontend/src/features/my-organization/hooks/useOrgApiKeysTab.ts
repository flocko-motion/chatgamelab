import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  useApiKeys,
  useCheckApiKey,
  useInstitutionApiKeys,
  useOrganisationApiKeys,
  usePlatforms,
  useRemoveInstitutionApiKeyShare,
  useSetInstitutionFreeUseKey,
  useShareApiKeyWithInstitution,
} from "@/api/hooks";
import type { KeyShareView } from "@/api/types";
import { isListNarrowed } from "@/common/lib/listNarrowed";
import { orgKeyListPreset } from "../lib/orgKeyListPresets";
import { useNewKeyForm } from "@/features/api-keys/hooks/useNewKeyForm";
import { useOrgKeyChoice } from "./useOrgKeyChoice";

export interface UseOrgApiKeysTabOptions {
  institutionId: string;
  freeUseApiKeyShareId?: string;
}

/**
 * Everything the organisation's key tab does, for ApiKeysTab to render: the
 * list of keys given to the organisation, giving one of the caller's keys or
 * a new one, removing one, and the free-use key.
 */
export function useOrgApiKeysTab({ institutionId, freeUseApiKeyShareId }: UseOrgApiKeysTabOptions) {
  const { t } = useTranslation("common");
  // ── The list ────────────────────────────────────────────────────────────
  const { data: platforms } = usePlatforms();
  const preset = useMemo(() => orgKeyListPreset(platforms ?? []), [platforms]);
  const list = useOrganisationApiKeys(institutionId, preset);
  const platformName = (id?: string) => platforms?.find((p) => p.id === id)?.name ?? id ?? "";

  // Every key given to the organisation, for what is not a list of them: which
  // of the caller's keys can still be given, and the free-use key's name.
  const { data: institutionKeys } = useInstitutionApiKeys(institutionId);
  const { data: userKeys } = useApiKeys();

  // ── Giving a key: one of the caller's, or a new one ────────────────────
  // A new key is the caller's own, as if added on their key page, and is
  // shared with the organisation in the same step.
  const shareApiKey = useShareApiKeyWithInstitution();
  const checkApiKey = useCheckApiKey();
  const newKey = useNewKeyForm(t);
  const [shareOpened, setShareOpened] = useState(false);
  const [mode, setMode] = useState<"own" | "new">("own");
  const [selectedKeyId, setSelectedKeyId] = useState<string | null>(null);
  const ownShares = userKeys?.shares ?? [];
  const availableKeysToShare = (userKeys?.apiKeys ?? []).filter(
    (key) => !institutionKeys?.some((share) => share.apiKeyId === key.id),
  );

  const openShare = () => {
    newKey.reset();
    setSelectedKeyId(null);
    setMode(availableKeysToShare.length > 0 ? "own" : "new");
    setShareOpened(true);
  };

  const closeShare = () => {
    setShareOpened(false);
    setSelectedKeyId(null);
  };

  const submitShare = async () => {
    if (mode === "new") {
      const created = await newKey.submit();
      if (!created) return;
      await shareApiKey.mutateAsync({ shareId: created.id, institutionId });
      closeShare();
      // A new key is checked at once, so its owner learns now whether it works.
      checkApiKey.mutate({ id: created.id });
      return;
    }
    if (!selectedKeyId) return;
    // The owner's own-use share of this key (the server takes a share, not a key).
    const selfShare = ownShares.find((s) => s.apiKeyId === selectedKeyId && s.self);
    if (!selfShare?.id) return;
    await shareApiKey.mutateAsync({ shareId: selfShare.id, institutionId });
    closeShare();
  };

  // ── Removing ────────────────────────────────────────────────────────────
  const removeShare = useRemoveInstitutionApiKeyShare();
  const remove = async (share: KeyShareView) => {
    await removeShare.mutateAsync({ shareId: share.id, institutionId });
  };

  // ── The free-use key ────────────────────────────────────────────────────
  const setFreeUseKey = useSetInstitutionFreeUseKey();
  const keyChoice = useOrgKeyChoice({
    institutionId,
    apply: (shareId) => setFreeUseKey.mutateAsync({ institutionId, shareId }),
    applying: setFreeUseKey.isPending,
  });
  const currentFreeUseShare = freeUseApiKeyShareId
    ? institutionKeys?.find((s) => s.id === freeUseApiKeyShareId)
    : undefined;

  return {
    list,
    /** Narrowed by a search or a platform, so empty means no match. */
    narrowed: isListNarrowed(list),
    platformName,
    remove,
    removing: removeShare.isPending,
    canShare: availableKeysToShare.length > 0 || newKey.available,
    shareModal: {
      opened: shareOpened,
      open: openShare,
      close: closeShare,
      /** Whether the caller gives one of their keys or adds a new one. */
      mode,
      setMode,
      keys: availableKeysToShare,
      selectedKeyId,
      select: setSelectedKeyId,
      newKey,
      submit: submitShare,
      loading: shareApiKey.isPending || newKey.loading,
      ready: mode === "new" || !!selectedKeyId,
    },
    freeUse: {
      shareId: freeUseApiKeyShareId,
      share: currentFreeUseShare,
      options: keyChoice.options,
      select: (value: string | null) => value && keyChoice.choose(value),
      clear: () => setFreeUseKey.mutate({ institutionId, shareId: null }),
      saving: setFreeUseKey.isPending,
    },
    autoShareModal: keyChoice.autoShareModal,
  };
}

export type OrgApiKeysTabState = ReturnType<typeof useOrgApiKeysTab>;

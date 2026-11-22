import { useMemo, useState } from "react";
import {
  useApiKeys,
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
import { useOrgKeyChoice } from "./useOrgKeyChoice";

export interface UseOrgApiKeysTabOptions {
  institutionId: string;
  freeUseApiKeyShareId?: string;
}

/**
 * Everything the organisation's key tab does, for ApiKeysTab to render: the
 * list of keys given to the organisation, giving one of the caller's keys,
 * removing one, and the free-use key.
 */
export function useOrgApiKeysTab({ institutionId, freeUseApiKeyShareId }: UseOrgApiKeysTabOptions) {
  // ── The list ────────────────────────────────────────────────────────────
  const { data: platforms } = usePlatforms();
  const preset = useMemo(() => orgKeyListPreset(platforms ?? []), [platforms]);
  const list = useOrganisationApiKeys(institutionId, preset);
  const platformName = (id?: string) => platforms?.find((p) => p.id === id)?.name ?? id ?? "";

  // Every key given to the organisation, for what is not a list of them: which
  // of the caller's keys can still be given, and the free-use key's name.
  const { data: institutionKeys } = useInstitutionApiKeys(institutionId);
  const { data: userKeys } = useApiKeys();

  // ── Giving one of the caller's keys ─────────────────────────────────────
  const shareApiKey = useShareApiKeyWithInstitution();
  const [shareOpened, setShareOpened] = useState(false);
  const [selectedKeyId, setSelectedKeyId] = useState<string | null>(null);
  const ownShares = userKeys?.shares ?? [];
  const availableKeysToShare = (userKeys?.apiKeys ?? []).filter(
    (key) => !institutionKeys?.some((share) => share.apiKeyId === key.id),
  );

  const submitShare = async () => {
    if (!selectedKeyId) return;
    // The owner's own-use share of this key (the server takes a share, not a key).
    const selfShare = ownShares.find((s) => s.apiKeyId === selectedKeyId && s.self);
    if (!selfShare?.id) return;
    await shareApiKey.mutateAsync({ shareId: selfShare.id, institutionId });
    setShareOpened(false);
    setSelectedKeyId(null);
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
    canShare: availableKeysToShare.length > 0,
    shareModal: {
      opened: shareOpened,
      open: () => setShareOpened(true),
      close: () => setShareOpened(false),
      keys: availableKeysToShare,
      selectedKeyId,
      select: setSelectedKeyId,
      submit: submitShare,
      loading: shareApiKey.isPending,
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

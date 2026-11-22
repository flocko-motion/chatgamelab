import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useShareApiKeyWithInstitution } from "@/api/hooks";
import { useOrgKeyOptions } from "./useOrgKeyOptions";

interface UseOrgKeyChoiceOptions {
  institutionId: string;
  /** Uses the chosen key, a share of the organisation (null: none). */
  apply: (shareId: string | null) => Promise<unknown>;
  /** `apply` is running. */
  applying: boolean;
  /** Offer "no key" first, with this label. */
  emptyLabel?: string;
}

/**
 * Choosing a key for the organisation (a workshop's key, the free-use key)
 * from the organisation's keys and the caller's own. One of the caller's own
 * keys is first given to the organisation, after a confirmation, and its new
 * share is then used.
 */
export function useOrgKeyChoice({ institutionId, apply, applying, emptyLabel }: UseOrgKeyChoiceOptions) {
  const { t } = useTranslation("common");
  const { options, personalKeyIds, personalSelfShareIds, personalKeyNames } = useOrgKeyOptions(
    institutionId,
    emptyLabel ? { includeEmpty: true, emptyLabel } : undefined,
  );
  const shareApiKey = useShareApiKeyWithInstitution();
  const [pending, setPending] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const choose = async (value: string | null) => {
    if (value && personalKeyIds.has(value)) {
      setPending(value);
      setError(null);
      return;
    }
    await apply(value);
  };

  const confirm = async () => {
    if (!pending) return;
    const selfShareId = personalSelfShareIds.get(pending);
    if (!selfShareId) return;
    try {
      const newShare = await shareApiKey.mutateAsync({ shareId: selfShareId, institutionId });
      if (newShare?.id) await apply(newShare.id);
      setPending(null);
      setError(null);
    } catch {
      setError(t("myOrganization.apiKeys.loadError"));
    }
  };

  return {
    options,
    choose,
    autoShareModal: {
      opened: !!pending,
      keyName: pending ? (personalKeyNames.get(pending) ?? "") : "",
      close: () => {
        setPending(null);
        setError(null);
      },
      confirm,
      loading: shareApiKey.isPending || applying,
      error,
    },
  };
}

export type OrgKeyChoice = ReturnType<typeof useOrgKeyChoice>;

import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import {
  useInstitutionApiKeys,
  useApiKeys,
} from "@/api/hooks";
import { useAuth } from "@/providers/AuthProvider";
import type { ComboboxItemGroup } from "@mantine/core";

export interface OrgKeyOption {
  value: string;
  label: string;
}

export interface UseOrgKeyOptionsResult {
  /** Grouped options for Mantine Select (org keys + personal keys) */
  options: (OrgKeyOption | ComboboxItemGroup)[];
  /** Set of Select option values that are personal (not yet shared with org) — keyed by `personal:<apiKeyId>` */
  personalKeyIds: Set<string>;
  /** Map from personal option value (`personal:<apiKeyId>`) to the self-share ID needed for the share API call */
  personalSelfShareIds: Map<string, string>;
  /** Map from personal option value to the key name (for confirmation modal) */
  personalKeyNames: Map<string, string>;
  isLoading: boolean;
}

/**
 * Builds a combined Select options list with two groups:
 * - Organization Keys (already shared with the institution)
 * - Personal Keys (owned by current user, not yet shared)
 *
 * Personal keys use a `personal:<apiKeyId>` value prefix to distinguish them
 * from institution share IDs.
 */
export function useOrgKeyOptions(
  institutionId: string,
  opts?: { includeEmpty?: boolean; emptyLabel?: string },
): UseOrgKeyOptionsResult {
  const { t } = useTranslation("common");
  const { backendUser } = useAuth();

  const { data: institutionKeys, isLoading: instLoading } =
    useInstitutionApiKeys(institutionId);
  const { data: userKeysData, isLoading: userLoading } = useApiKeys();

  return useMemo(() => {
    const personalKeyIds = new Set<string>();
    const personalSelfShareIds = new Map<string, string>();
    const personalKeyNames = new Map<string, string>();

    // Organization keys (already shared with institution)
    const orgItems: OrgKeyOption[] =
      institutionKeys?.map((share) => ({
        value: share.id || "",
        label: `${share.apiKey?.name || t("unnamed")} (${share.apiKey?.platform || "?"})`,
      })) ?? [];

    // Personal keys: owned by current user, not already shared with this institution
    const ownKeys = userKeysData?.apiKeys ?? [];
    const ownShares = userKeysData?.shares ?? [];
    const sharedApiKeyIds = new Set(
      institutionKeys?.map((s) => s.apiKeyId).filter(Boolean) ?? [],
    );

    const personalItems: OrgKeyOption[] = [];
    for (const key of ownKeys) {
      if (key.userId !== backendUser?.id) continue;
      if (sharedApiKeyIds.has(key.id)) continue;

      // Find the user's self-share for this key (needed for the share API call)
      const selfShare = ownShares.find(
        (s) =>
          s.apiKeyId === key.id &&
          s.user &&
          !s.institution &&
          !s.workshop &&
          !s.game,
      );
      if (!selfShare?.id) continue;

      const optionValue = `personal:${key.id}`;
      personalKeyIds.add(optionValue);
      personalSelfShareIds.set(optionValue, selfShare.id);
      personalKeyNames.set(
        optionValue,
        `${key.name || t("unnamed")} (${key.platform || "?"})`,
      );

      personalItems.push({
        value: optionValue,
        label: `${key.name || t("unnamed")} (${key.platform || "?"})`,
      });
    }

    // Build grouped options
    const options: (OrgKeyOption | ComboboxItemGroup)[] = [];

    if (opts?.includeEmpty) {
      options.push({
        value: "",
        label: opts.emptyLabel || t("myOrganization.workshops.noDefaultApiKey"),
      });
    }

    if (orgItems.length > 0 && personalItems.length > 0) {
      // Both groups exist — use Mantine group format
      options.push({
        group: t("myOrganization.autoShare.orgKeyGroup"),
        items: orgItems,
      });
      options.push({
        group: t("myOrganization.autoShare.personalKeyGroup"),
        items: personalItems,
      });
    } else if (orgItems.length > 0) {
      // Only org keys — no grouping needed
      options.push(...orgItems);
    } else if (personalItems.length > 0) {
      // Only personal keys — show group header for clarity
      options.push({
        group: t("myOrganization.autoShare.personalKeyGroup"),
        items: personalItems,
      });
    }

    return {
      options,
      personalKeyIds,
      personalSelfShareIds,
      personalKeyNames,
      isLoading: instLoading || userLoading,
    };
  }, [
    institutionKeys,
    userKeysData,
    backendUser?.id,
    instLoading,
    userLoading,
    t,
    opts?.includeEmpty,
    opts?.emptyLabel,
  ]);
}

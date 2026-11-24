import { useApiKeys, useInstitutionApiKeys } from "@/api/hooks";
import { useAuth } from "@/providers/AuthProvider";

/** A key share the caller may let pay for something of theirs. */
export interface PayingKey {
  shareId: string;
  name: string;
  platform: string;
  /** Given to the caller's organisation, not their own. */
  isOrg: boolean;
}

/**
 * The key shares the caller may draw from to pay for a game shown elsewhere
 * (a share link, an embed): their own-use shares and their organisation's,
 * where the server says they may share them, and the key last worked. One
 * entry per share.
 */
export function usePayingKeys() {
  const { backendUser } = useAuth();
  const institutionId = backendUser?.role?.institution?.id;
  const { data: apiKeys, isLoading: ownLoading } = useApiKeys();
  const { data: institutionKeys, isLoading: orgLoading } = useInstitutionApiKeys(institutionId ?? "");

  const keys: PayingKey[] = [];
  const seen = new Set<string>();
  for (const share of apiKeys?.shares ?? []) {
    if (!share.id || seen.has(share.id) || !share.self || !share.mayShare) continue;
    const key = apiKeys?.apiKeys?.find((k) => k.id === share.apiKeyId);
    if (key?.lastUsageSuccess === false) continue;
    seen.add(share.id);
    keys.push({ shareId: share.id, name: key?.name ?? "", platform: key?.platform ?? "", isOrg: false });
  }
  for (const share of institutionKeys ?? []) {
    if (!share.id || seen.has(share.id) || !share.mayShare) continue;
    if (share.apiKey?.lastUsageSuccess === false) continue;
    seen.add(share.id);
    keys.push({
      shareId: share.id,
      name: share.apiKey?.name ?? "",
      platform: share.apiKey?.platform ?? "",
      isOrg: true,
    });
  }
  return { keys, loading: ownLoading || orgLoading };
}

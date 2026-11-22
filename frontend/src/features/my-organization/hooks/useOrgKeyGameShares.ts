import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useApiKeyGameShares, useRevokePrivateShare, useUpdateGameShare } from "@/api/hooks";
import type { ShareLinkView } from "@/api/types";

/**
 * The game share links that play on one of the organisation's keys, for
 * OrgKeyDetails to render: what each is for, how many plays it has left,
 * changing that and revoking it.
 */
export function useOrgKeyGameShares(shareId: string) {
  const { t } = useTranslation("common");
  const { data: gameShares, isLoading } = useApiKeyGameShares(shareId, "organization");
  const revokeShare = useRevokePrivateShare();
  const updateShare = useUpdateGameShare();

  // ── Changing the plays left ─────────────────────────────────────────────
  const [editing, setEditing] = useState<ShareLinkView | null>(null);
  const [maxSessions, setMaxSessions] = useState<number | string>("");

  const startEdit = (gameShare: ShareLinkView) => {
    setMaxSessions(gameShare.remaining ?? "");
    setEditing(gameShare);
  };
  const submitEdit = async () => {
    if (!editing?.gameId || !editing.id) return;
    try {
      await updateShare.mutateAsync({
        gameId: editing.gameId,
        shareId: editing.id,
        maxSessions: typeof maxSessions === "number" && maxSessions > 0 ? maxSessions : null,
      });
      setEditing(null);
    } catch {
      // shown by the mutation's error handler
    }
  };

  const revoke = (gameShare: ShareLinkView) => {
    if (gameShare.gameId && gameShare.id) {
      revokeShare.mutate({ gameId: gameShare.gameId, shareId: gameShare.id });
    }
  };

  /** Where the link was made: a workshop (named), the organisation, or personally. */
  const contextLabel = (gameShare: ShareLinkView): string => {
    if (gameShare.source === "workshop") {
      return gameShare.workshopName
        ? `${t("apiKeys.shares.context.workshop")}: ${gameShare.workshopName}`
        : t("apiKeys.shares.context.workshop");
    }
    if (gameShare.source === "organization") return t("apiKeys.shares.context.organization");
    return t("apiKeys.shares.context.personal");
  };

  const remainingLabel = (gameShare: ShareLinkView): string =>
    gameShare.remaining == null
      ? t("apiKeys.shares.unlimited")
      : t("apiKeys.shares.remaining", { count: gameShare.remaining });

  return {
    gameShares: gameShares ?? [],
    isLoading,
    contextLabel,
    remainingLabel,
    startEdit,
    revoke,
    editModal: {
      opened: !!editing,
      close: () => setEditing(null),
      maxSessions,
      setMaxSessions,
      submit: submitEdit,
      loading: updateShare.isPending,
    },
  };
}

import { useState } from "react";
import {
  useCreateVerifiedPage,
  useDeleteVerifiedPage,
  useVerifiedPages,
} from "@/api/hooks";
import type { VerifiedPageView } from "@/api/types";

/** A web address the server may take: a full https address. */
function isHttpsURL(value: string): boolean {
  try {
    return new URL(value).protocol === "https:";
  } catch {
    return false;
  }
}

/**
 * The organisation's verified pages and registering one: the address of the
 * page that frames its embedded games and the callback that says yes or no
 * to each start. The server decides who may; the tab shows once it answers
 * the list.
 */
export function useVerifiedPagesTab(institutionId: string | undefined) {
  const pages = useVerifiedPages(institutionId);
  const create = useCreateVerifiedPage();
  const remove = useDeleteVerifiedPage();

  const [pageUrl, setPageUrl] = useState("");
  const [callbackUrl, setCallbackUrl] = useState("");
  // The page asked to be removed, waiting for the confirmation.
  const [removing, setRemoving] = useState<VerifiedPageView | null>(null);

  const draftValid = isHttpsURL(pageUrl.trim()) && isHttpsURL(callbackUrl.trim());

  const add = async () => {
    if (!institutionId || !draftValid) return;
    try {
      await create.mutateAsync({
        institutionId,
        pageUrl: pageUrl.trim(),
        callbackUrl: callbackUrl.trim(),
      });
      setPageUrl("");
      setCallbackUrl("");
    } catch {
      // handled by the mutation
    }
  };

  const confirmRemove = async () => {
    if (!institutionId || !removing) return;
    try {
      await remove.mutateAsync({ id: removing.id, institutionId });
      setRemoving(null);
    } catch {
      // handled by the mutation
    }
  };

  return {
    /** Whether the server answers the list: the caller may manage the pages. */
    available: pages.isSuccess,
    pages: pages.data ?? [],
    draft: { pageUrl, setPageUrl, callbackUrl, setCallbackUrl, valid: draftValid },
    add,
    adding: create.isPending,
    removing,
    askRemove: setRemoving,
    cancelRemove: () => setRemoving(null),
    confirmRemove,
    removePending: remove.isPending,
  };
}

export type VerifiedPagesTabState = ReturnType<typeof useVerifiedPagesTab>;

import { useState } from "react";
import {
  useCreateVerifiedPage,
  useDeleteVerifiedPage,
  useVerifiedPages,
} from "@/api/hooks";
import type { VerifiedPageView } from "@/api/types";

/**
 * A full web address. Which ones it takes is the server's: https, and in
 * development plain http to the own machine.
 */
function isWebURL(value: string): boolean {
  try {
    const { protocol } = new URL(value);
    return protocol === "https:" || protocol === "http:";
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

  const draftValid = isWebURL(pageUrl.trim()) && isWebURL(callbackUrl.trim());

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
    /** An organisation has one verified page: registering is for one without. */
    mayRegister: (pages.data ?? []).length === 0,
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

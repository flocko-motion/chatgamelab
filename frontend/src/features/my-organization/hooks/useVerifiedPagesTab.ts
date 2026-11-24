import { useState } from "react";
import {
  useCreateVerifiedPage,
  useDeleteVerifiedPage,
  useUpdateVerifiedPage,
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
 * The organisation's verified pages, registering one and changing it: the
 * address of the page that frames its embedded games and the callback that
 * says yes or no to each start. One form does both; editing fills it with
 * the page. The server decides who may; the tab shows once it answers
 * the list.
 */
export function useVerifiedPagesTab(institutionId: string | undefined) {
  const pages = useVerifiedPages(institutionId);
  const create = useCreateVerifiedPage();
  const update = useUpdateVerifiedPage();
  const remove = useDeleteVerifiedPage();

  const [pageUrl, setPageUrl] = useState("");
  const [callbackUrl, setCallbackUrl] = useState("");
  // The page being changed in the form; null: the form registers a new one.
  const [editing, setEditing] = useState<VerifiedPageView | null>(null);
  // The page asked to be removed, waiting for the confirmation.
  const [removing, setRemoving] = useState<VerifiedPageView | null>(null);

  const draftValid = isWebURL(pageUrl.trim()) && isWebURL(callbackUrl.trim());

  const clearForm = () => {
    setEditing(null);
    setPageUrl("");
    setCallbackUrl("");
  };

  const startEdit = (page: VerifiedPageView) => {
    setEditing(page);
    setPageUrl(page.pageUrl);
    setCallbackUrl(page.callbackUrl);
  };

  /** Registers the page in the form, or saves the one being changed. */
  const submit = async () => {
    if (!institutionId || !draftValid) return;
    const page = { pageUrl: pageUrl.trim(), callbackUrl: callbackUrl.trim() };
    try {
      if (editing) {
        await update.mutateAsync({ id: editing.id, institutionId, ...page });
      } else {
        await create.mutateAsync({ institutionId, ...page });
      }
      clearForm();
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
    /** The page the form changes; null while it registers a new one. */
    editing,
    startEdit,
    cancelEdit: clearForm,
    submit,
    submitting: create.isPending || update.isPending,
    removing,
    askRemove: setRemoving,
    cancelRemove: () => setRemoving(null),
    confirmRemove,
    removePending: remove.isPending,
  };
}

export type VerifiedPagesTabState = ReturnType<typeof useVerifiedPagesTab>;

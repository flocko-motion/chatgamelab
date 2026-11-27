import { useState } from "react";
import {
  useCreateVerifiedPage,
  useDeleteVerifiedPage,
  useUpdateVerifiedPage,
  useVerifiedPages,
  verifiedPageFieldError,
  type VerifiedPageField,
} from "@/api/hooks";
import type { VerifiedPageView } from "@/api/types";
import { translateErrorCode } from "@/common/lib/errorHelpers";

/**
 * A full web address. Which ones it takes is the server's: https, and on a
 * development server plain http as well.
 */
function isWebURL(value: string): boolean {
  try {
    const { protocol } = new URL(value);
    return protocol === "https:" || protocol === "http:";
  } catch {
    return false;
  }
}

/** An address's site, scheme and host, as the server compares it; null for no address. */
function siteOf(value: string): string | null {
  try {
    const { protocol, hostname } = new URL(value);
    return `${protocol}//${hostname.toLowerCase()}`;
  } catch {
    return null;
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
  // The server's refusal of one address, shown at its field until that field changes.
  const [refusal, setRefusal] = useState<{ field: VerifiedPageField; code: string } | null>(null);

  const draftValid = isWebURL(pageUrl.trim()) && isWebURL(callbackUrl.trim());

  const clearForm = () => {
    setEditing(null);
    setPageUrl("");
    setCallbackUrl("");
    setRefusal(null);
  };

  const startEdit = (page: VerifiedPageView) => {
    setEditing(page);
    setPageUrl(page.pageUrl);
    setCallbackUrl(page.callbackUrl);
    setRefusal(null);
  };

  /** A field's change clears the refusal shown at it. */
  const changePageUrl = (value: string) => {
    setPageUrl(value);
    setRefusal((r) => (r?.field === "pageUrl" ? null : r));
  };
  const changeCallbackUrl = (value: string) => {
    setCallbackUrl(value);
    setRefusal((r) => (r?.field === "callbackUrl" ? null : r));
  };

  /** The translated refusal at a field, or undefined. */
  const errorAt = (field: VerifiedPageField) =>
    refusal?.field === field ? translateErrorCode(refusal.code).message : undefined;

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
    } catch (error) {
      // A refusal of an address shows at its field; the mutation shows any other.
      setRefusal(verifiedPageFieldError(error));
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
    draft: {
      pageUrl,
      setPageUrl: changePageUrl,
      callbackUrl,
      setCallbackUrl: changeCallbackUrl,
      valid: draftValid,
      pageUrlError: errorAt("pageUrl"),
      callbackUrlError: errorAt("callbackUrl"),
    },
    /** The page the form changes; null while it registers a new one. */
    editing,
    /** The changed address is on another site than the page's: its approval would go. */
    leavesSite: !!editing && siteOf(pageUrl.trim()) !== null && siteOf(pageUrl.trim()) !== editing.site,
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

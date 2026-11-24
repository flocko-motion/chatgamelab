import { useAllVerifiedPages, useApproveVerifiedPage } from "@/api/hooks";
import { VERIFIED_PAGE_LIST_PRESET } from "../lib/verifiedPageListPresets";

/**
 * The admins' allow list of verified pages, waiting first, and approving a
 * page or withdrawing the approval. The server decides who may approve
 * (mayApprove on each page).
 */
export function useVerifiedPagesApproval() {
  const list = useAllVerifiedPages(VERIFIED_PAGE_LIST_PRESET.list);
  const approve = useApproveVerifiedPage();
  return {
    list,
    /** The page whose approval is being changed. */
    pending: approve.isPending ? approve.variables?.id : undefined,
    setApproved: (id: string, approved: boolean) => approve.mutate({ id, approved }),
  };
}

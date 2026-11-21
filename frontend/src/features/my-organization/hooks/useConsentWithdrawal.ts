import { useDisclosure } from "@mantine/hooks";
import { useOrganisationConsent, useWithdrawConsent } from "@/api/hooks";
import { useAuth } from "@/providers/AuthProvider";

/**
 * A head's view of their organisation's consent to the declaration, and its
 * withdrawal, which suspends the organisation until a head consents again.
 * Only for a caller who may edit the organisation; null otherwise.
 */
export function useConsentWithdrawal() {
  const { backendUser, standing, refreshCaller } = useAuth();
  const institutionId = standing.mayConsentForInstitution ? backendUser?.role?.institution?.id : undefined;
  const consent = useOrganisationConsent(institutionId);
  const withdraw = useWithdrawConsent();
  const [confirming, { open, close }] = useDisclosure(false);

  const confirm = async () => {
    if (!institutionId) return;
    await withdraw.mutateAsync(institutionId);
    close();
    await refreshCaller(); // the organisation is suspended now
  };

  return {
    status: institutionId ? consent.data?.declaration : undefined,
    confirming,
    open,
    close,
    confirm,
    withdrawing: withdraw.isPending,
    failed: withdraw.isError,
  };
}

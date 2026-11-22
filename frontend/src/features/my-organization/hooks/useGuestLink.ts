import { useState } from "react";
import { useGuestToken, useResetGuestToken } from "@/api/hooks";
import { ErrorCodes, extractRawErrorCode } from "@/common/types/errorCodes";

/**
 * A Guest's re-login link, for GuestLinkModal to render: the token, and
 * replacing it ("reset access") after a confirmation.
 */
export function useGuestLink(guestId: string | null, onClose: () => void) {
  const { data: token, isError } = useGuestToken(guestId ?? undefined);
  const resetToken = useResetGuestToken();
  const [confirmReset, setConfirmReset] = useState(false);
  const [justReset, setJustReset] = useState(false);

  const close = () => {
    setJustReset(false);
    resetToken.reset();
    onClose();
  };

  const reset = async () => {
    if (!guestId) return;
    try {
      const result = await resetToken.mutateAsync(guestId);
      if (result?.token) setJustReset(true);
      setConfirmReset(false);
    } catch {
      // Shown from resetError below; the dialogue stays open so it is visible.
    }
  };

  // A reset of an account without a word code answers not_found: the update
  // matched no row. That deserves a different message from a failed attempt.
  const resetError: "noToken" | "failed" | null = !resetToken.isError
    ? null
    : extractRawErrorCode(resetToken.error) === ErrorCodes.NOT_FOUND
      ? "noToken"
      : "failed";

  return {
    token,
    /** The Guest has no re-login link (or it could not be read). */
    missing: isError || token === null,
    justReset,
    close,
    confirmReset,
    askReset: () => setConfirmReset(true),
    cancelReset: () => setConfirmReset(false),
    reset,
    resetting: resetToken.isPending,
    resetError,
  };
}

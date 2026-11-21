import { api } from "@/api/client/api";
import { useState } from "react";
import { Alert, Center, Loader, Modal } from "@mantine/core";
import { IconAlertCircle, IconLink, IconRefresh } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useResetGuestToken } from "@/api/hooks";
import { FullscreenQrOverlay } from "@components/share";
import { DangerButton } from "@/common/components/buttons/DangerButton";
import { buildShareUrl } from "@/common/lib/url";
import { ErrorCodes } from "@/common/types/errorCodes";
import { guestShareLinkPath } from "@/common/lib/wordToken";
import { ConfirmationModal } from "./ConfirmationModal";

interface GuestLinkModalProps {
  /** Opens the modal when set. */
  guestId: string | null;
  guestName?: string;
  onClose: () => void;
}

/** The re-login link of a Guest in a workshop as QR code, with "reset access". */
export function GuestLinkModal({
  guestId,
  guestName,
  onClose,
}: GuestLinkModalProps) {
  const { t } = useTranslation("common");
  const queryClient = useQueryClient();
  const resetToken = useResetGuestToken();
  const [confirmReset, setConfirmReset] = useState(false);
  const [justReset, setJustReset] = useState(false);

  const queryKey = ["guestToken", guestId];
  const { data: token, isError } = useQuery({
    queryKey,
    queryFn: async () => {
      const response = await api.getApiWorkshopsGuestsGuestIdToken(
        guestId!,
      );
      return response.data?.token ?? null;
    },
    enabled: !!guestId,
    staleTime: 0,
    gcTime: 0,
  });

  // A reset of an account without a word code answers not_found: the update matched
  // no row. That deserves a different message from a failed attempt. Reads the code
  // the way MembersTab does, not the status, so it survives a route change.
  const resetErrorIsNotFound =
    (resetToken.error as { error?: { code?: string } } | null)?.error?.code ===
    ErrorCodes.NOT_FOUND;

  const handleClose = () => {
    setJustReset(false);
    resetToken.reset();
    onClose();
  };

  const handleReset = async () => {
    if (!guestId) return;
    try {
      const result = await resetToken.mutateAsync(guestId);
      if (result?.token) {
        queryClient.setQueryData(queryKey, result.token);
        setJustReset(true);
      }
      setConfirmReset(false);
    } catch {
      // The rejection is rendered from resetToken.isError below; swallowing it
      // here only keeps it from becoming an unhandled promise rejection. The
      // dialogue stays open so the message is visible.
    }
  };

  const title = guestName
    ? t("myOrganization.workshops.guestLinkTitleNamed", {
        name: guestName,
      })
    : t("myOrganization.workshops.guestLinkTitle");

  if (!guestId) return null;

  if (!token) {
    return (
      <Modal opened onClose={handleClose} title={title} size="md">
        {isError || token === null ? (
          <Alert color="red" icon={<IconAlertCircle size={16} />}>
            {t("myOrganization.workshops.noGuestToken")}
          </Alert>
        ) : (
          <Center py="xl">
            <Loader />
          </Center>
        )}
      </Modal>
    );
  }

  return (
    <>
      <FullscreenQrOverlay
        opened={!confirmReset}
        onClose={handleClose}
        title={title}
        icon={<IconLink size={28} />}
        color="blue"
        url={buildShareUrl(guestShareLinkPath(token))}
        actions={
          <DangerButton
            size="md"
            leftSection={<IconRefresh size={16} />}
            onClick={() => setConfirmReset(true)}
          >
            {t("myOrganization.workshops.resetAccess")}
          </DangerButton>
        }
      >
        {justReset && (
          <Alert color="green">
            {t("myOrganization.workshops.resetAccessDone")}
          </Alert>
        )}
      </FullscreenQrOverlay>
      <ConfirmationModal
        opened={confirmReset}
        onClose={() => setConfirmReset(false)}
        title={t("myOrganization.workshops.resetAccessTitle")}
        message={t("myOrganization.workshops.resetAccessMessage")}
        warning={t("myOrganization.workshops.resetAccessWarning")}
        confirmIcon={<IconRefresh size={18} />}
        confirmColor="red"
        onConfirm={handleReset}
        isLoading={resetToken.isPending}
        error={
          resetToken.isError
            ? resetErrorIsNotFound
              ? // Nothing to reset: this account has no word code. Retrying cannot
                // help, so say what the read side says instead of "try again".
                t("myOrganization.workshops.noGuestToken")
              : t("myOrganization.workshops.resetAccessError")
            : null
        }
      />
    </>
  );
}

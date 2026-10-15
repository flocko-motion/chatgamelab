import { useState } from "react";
import { Alert, Center, Loader, Modal } from "@mantine/core";
import { IconAlertCircle, IconLink, IconRefresh } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useRequiredAuthenticatedApi } from "@/api/useAuthenticatedApi";
import { useResetParticipantToken } from "@/api/hooks";
import { FullscreenQrOverlay } from "@components/share";
import { DangerButton } from "@/common/components/buttons/DangerButton";
import { buildShareUrl } from "@/common/lib/url";
import { ErrorCodes } from "@/common/types/errorCodes";
import { participantShareLinkPath } from "@/common/lib/wordToken";
import { ConfirmationModal } from "./ConfirmationModal";

interface ParticipantLinkModalProps {
  /** Opens the modal when set. */
  participantId: string | null;
  participantName?: string;
  onClose: () => void;
}

/** A participant's re-login link as QR code, with "reset access". */
export function ParticipantLinkModal({
  participantId,
  participantName,
  onClose,
}: ParticipantLinkModalProps) {
  const { t } = useTranslation("common");
  const api = useRequiredAuthenticatedApi();
  const queryClient = useQueryClient();
  const resetToken = useResetParticipantToken();
  const [confirmReset, setConfirmReset] = useState(false);
  const [justReset, setJustReset] = useState(false);

  const queryKey = ["participantToken", participantId];
  const { data: token, isError } = useQuery({
    queryKey,
    queryFn: async () => {
      const response = await api.workshops.participantsTokenList(
        participantId!,
      );
      return response.data?.token ?? null;
    },
    enabled: !!participantId,
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
    if (!participantId) return;
    try {
      const result = await resetToken.mutateAsync(participantId);
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

  const title = participantName
    ? t("myOrganization.workshops.participantLinkTitleNamed", {
        name: participantName,
      })
    : t("myOrganization.workshops.participantLinkTitle");

  if (!participantId) return null;

  if (!token) {
    return (
      <Modal opened onClose={handleClose} title={title} size="md">
        {isError || token === null ? (
          <Alert color="red" icon={<IconAlertCircle size={16} />}>
            {t("myOrganization.workshops.noParticipantToken")}
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
        url={buildShareUrl(participantShareLinkPath(token))}
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
                t("myOrganization.workshops.noParticipantToken")
              : t("myOrganization.workshops.resetAccessError")
            : null
        }
      />
    </>
  );
}

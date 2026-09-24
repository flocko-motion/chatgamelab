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

  const handleClose = () => {
    setJustReset(false);
    resetToken.reset();
    onClose();
  };

  const handleReset = async () => {
    if (!participantId) return;
    const result = await resetToken.mutateAsync(participantId);
    if (result?.token) {
      queryClient.setQueryData(queryKey, result.token);
      setJustReset(true);
    }
    setConfirmReset(false);
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
            ? t("myOrganization.workshops.resetAccessError")
            : null
        }
      />
    </>
  );
}

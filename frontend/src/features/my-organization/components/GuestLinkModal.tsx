import { Alert, Center, Loader, Modal } from "@mantine/core";
import { IconAlertCircle, IconLink, IconRefresh } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { FullscreenQrOverlay } from "@components/share";
import { DangerButton } from "@/common/components/buttons/DangerButton";
import { buildShareUrl } from "@/common/lib/url";
import { guestShareLinkPath } from "@/common/lib/wordToken";
import { useGuestLink } from "../hooks/useGuestLink";
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
  const state = useGuestLink(guestId, onClose);
  const { token } = state;

  const title = guestName
    ? t("myOrganization.workshops.guestLinkTitleNamed", {
        name: guestName,
      })
    : t("myOrganization.workshops.guestLinkTitle");

  if (!guestId) return null;

  if (!token) {
    return (
      <Modal opened onClose={state.close} title={title} size="md">
        {state.missing ? (
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
        opened={!state.confirmReset}
        onClose={state.close}
        title={title}
        icon={<IconLink size={28} />}
        color="blue"
        url={buildShareUrl(guestShareLinkPath(token))}
        actions={
          <DangerButton
            size="md"
            leftSection={<IconRefresh size={16} />}
            onClick={state.askReset}
          >
            {t("myOrganization.workshops.resetAccess")}
          </DangerButton>
        }
      >
        {state.justReset && (
          <Alert color="green">
            {t("myOrganization.workshops.resetAccessDone")}
          </Alert>
        )}
      </FullscreenQrOverlay>
      <ConfirmationModal
        opened={state.confirmReset}
        onClose={state.cancelReset}
        title={t("myOrganization.workshops.resetAccessTitle")}
        message={t("myOrganization.workshops.resetAccessMessage")}
        warning={t("myOrganization.workshops.resetAccessWarning")}
        confirmIcon={<IconRefresh size={18} />}
        confirmColor="red"
        onConfirm={state.reset}
        isLoading={state.resetting}
        error={
          state.resetError === "noToken"
            ? // Nothing to reset: this account has no word code. Retrying cannot
              // help, so say what the read side says instead of "try again".
              t("myOrganization.workshops.noGuestToken")
            : state.resetError === "failed"
              ? t("myOrganization.workshops.resetAccessError")
              : null
        }
      />
    </>
  );
}

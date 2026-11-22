import { Badge, Group, Modal, Stack } from "@mantine/core";
import { IconCalendar, IconClock, IconLink } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { FullscreenQrOverlay } from "@components/share";
import { DangerButton } from "@components/buttons";
import { BodyText, HelperText } from "@components/typography";
import { buildShareUrl } from "@/common/lib/url";
import { inviteShareLinkPath } from "@/common/lib/wordToken";
import type { WorkshopInvites } from "../hooks/useWorkshopInvites";
import { InviteModal } from "./InviteModal";

/** A workshop's invite link as a QR code (revocable), and the invite by email. */
export function WorkshopInviteModals({ invites }: { invites: WorkshopInvites }) {
  const { t } = useTranslation("common");
  const { linkModal, emailModal } = invites;
  const invite = linkModal.invite;
  const title = t("myOrganization.workshops.inviteLinkTitle", { name: linkModal.workshop?.name });
  const createdAt = invite?.meta?.createdAt ? new Date(invite.meta.createdAt) : null;
  const expiresAt = invite?.expiresAt ? new Date(invite.expiresAt) : null;

  return (
    <>
      {invite?.inviteToken ? (
        <FullscreenQrOverlay
          opened={!!linkModal.workshop}
          onClose={linkModal.close}
          title={title}
          icon={<IconLink size={28} />}
          color="blue"
          url={buildShareUrl(inviteShareLinkPath(invite.inviteToken))}
          actions={
            <DangerButton
              size="md"
              onClick={() => {
                if (invite.id) linkModal.revoke(invite.id);
              }}
              loading={linkModal.revoking}
            >
              {t("myOrganization.workshops.revokeInvite")}
            </DangerButton>
          }
        >
          <Group gap="xl" justify="center">
            {createdAt && (
              <Stack gap={2}>
                <HelperText>{t("myOrganization.workshops.inviteCreatedAt")}</HelperText>
                <Group gap="xs">
                  <IconCalendar size={14} />
                  <BodyText size="sm">{createdAt.toLocaleDateString()}</BodyText>
                </Group>
              </Stack>
            )}
            {expiresAt && (
              <Stack gap={2}>
                <HelperText>{t("myOrganization.workshops.inviteExpiresAt")}</HelperText>
                <Group gap="xs">
                  <IconClock size={14} />
                  <BodyText size="sm">{expiresAt.toLocaleDateString()}</BodyText>
                </Group>
              </Stack>
            )}
            {invite.usesCount !== undefined && invite.usesCount > 0 && (
              <Stack gap={2}>
                <HelperText>{t("myOrganization.workshops.inviteUsage")}</HelperText>
                <Badge size="sm" variant="light">
                  {t("myOrganization.workshops.usedCount", { count: invite.usesCount })}
                </Badge>
              </Stack>
            )}
          </Group>
        </FullscreenQrOverlay>
      ) : (
        <Modal opened={!!linkModal.workshop} onClose={linkModal.close} title={title} size="md">
          <HelperText>{t("myOrganization.workshops.noActiveInvite")}</HelperText>
        </Modal>
      )}

      <InviteModal
        opened={emailModal.opened}
        onClose={emailModal.close}
        title={t("myOrganization.workshops.inviteByEmailTitle")}
        description={t("myOrganization.workshops.inviteByEmailDescription")}
        onSubmit={emailModal.submit}
        isLoading={emailModal.loading}
        error={emailModal.error}
      />
    </>
  );
}

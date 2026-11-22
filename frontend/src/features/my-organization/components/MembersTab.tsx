import { Stack, Card, Group } from "@mantine/core";
import { IconCrown, IconCrownOff, IconTrash, IconLogout } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { InvitesList } from "@/features/admin/components/InvitesList";
import { ActionButton, DangerButton } from "@components/buttons";
import { CardTitle } from "@components/typography";
import type { OrganisationView } from "@/api/types";
import { useMembersTab } from "../hooks/useMembersTab";
import { MembersList } from "./MembersList";
import { InviteModal } from "./InviteModal";
import { ConfirmationModal } from "./ConfirmationModal";

interface MembersTabProps {
  institution: OrganisationView | null | undefined;
}

export function MembersTab({ institution }: MembersTabProps) {
  const { t } = useTranslation("common");
  const state = useMembersTab(institution);
  const { mayInvite, seesInvites, institutionId } = state;
  const { inviteModal, promoteModal, demoteModal, removeModal, leaveModal } = state;

  return (
    <>
      <Stack gap="lg">
        {/* Action buttons */}
        <Group gap="sm" wrap="wrap">
          {mayInvite && (
            <ActionButton onClick={inviteModal.open} size="sm">
              {t("myOrganization.inviteTeacher")}
            </ActionButton>
          )}
        </Group>

        <MembersList state={state} />

        {/* Pending Invites */}
        {seesInvites && institutionId && (
          <Card shadow="sm" padding="lg" radius="md" withBorder>
            <Stack gap="md">
              <CardTitle>{t("admin.invites.title")}</CardTitle>
              <InvitesList institutionId={institutionId} showInstitutionColumn={false} />
            </Stack>
          </Card>
        )}

        {/* Leave Organization */}
        <Group justify="flex-start">
          <DangerButton onClick={leaveModal.open} size="sm">
            {t("myOrganization.leaveOrganization")}
          </DangerButton>
        </Group>
      </Stack>

      {/* Modals */}
      <InviteModal
        opened={inviteModal.opened}
        onClose={inviteModal.close}
        title={t("myOrganization.inviteTeacherTitle")}
        description={t("myOrganization.inviteTeacherDescription")}
        onSubmit={inviteModal.submit}
        isLoading={inviteModal.loading}
        error={inviteModal.error}
      />

      <ConfirmationModal
        opened={!!promoteModal.member}
        onClose={promoteModal.close}
        title={t("myOrganization.promoteTitle")}
        message={t("myOrganization.promoteConfirm", {
          name: promoteModal.member?.name,
          email: promoteModal.member?.email || t("myOrganization.noEmail"),
        })}
        warning={t("myOrganization.promoteWarning")}
        warningColor="yellow"
        confirmIcon={<IconCrown size={16} />}
        confirmColor="violet"
        onConfirm={promoteModal.confirm}
        isLoading={promoteModal.loading}
      />

      <ConfirmationModal
        opened={!!demoteModal.member}
        onClose={demoteModal.close}
        title={t("myOrganization.demoteTitle")}
        message={t("myOrganization.demoteConfirm", {
          name: demoteModal.member?.name,
          email: demoteModal.member?.email || t("myOrganization.noEmail"),
        })}
        confirmIcon={<IconCrownOff size={16} />}
        confirmColor="gray"
        onConfirm={demoteModal.confirm}
        isLoading={demoteModal.loading}
      />

      <ConfirmationModal
        opened={!!removeModal.member}
        onClose={removeModal.close}
        title={t("myOrganization.removeTitle")}
        message={t("myOrganization.removeConfirm", { name: removeModal.member?.name })}
        warning={t("myOrganization.removeWarning")}
        warningColor="orange"
        confirmIcon={<IconTrash size={16} />}
        confirmColor="red"
        onConfirm={removeModal.confirm}
        isLoading={removeModal.loading}
      />

      <ConfirmationModal
        opened={leaveModal.opened}
        onClose={leaveModal.close}
        title={t("myOrganization.leaveTitle")}
        message={t("myOrganization.leaveConfirm", { name: institution?.name })}
        warning={t("myOrganization.leaveWarning")}
        warningColor="red"
        confirmIcon={<IconLogout size={16} />}
        confirmColor="red"
        onConfirm={leaveModal.confirm}
        isLoading={leaveModal.loading}
        error={leaveModal.error}
      />
    </>
  );
}

import { useState, useCallback } from "react";
import { Stack, Card, Group, Title } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import {
  IconCrown,
  IconCrownOff,
  IconTrash,
  IconLogout,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "@/api/queryKeys";
import { useNavigate } from "@tanstack/react-router";
import { useRequiredAuthenticatedApi } from "@/api/useAuthenticatedApi";
import { useAuth } from "@/providers/AuthProvider";
import {
  isAtLeastHead,
  getUserInstitutionId,
  Role,
  hasRole,
} from "@/common/lib/roles";
import { InvitesList } from "@/features/admin/components/InvitesList";
import { ActionButton } from "@/common/components/buttons/ActionButton";
import { MembersList } from "./MembersList";
import { InviteModal } from "./InviteModal";
import { ConfirmationModal } from "./ConfirmationModal";
import type { ObjUser, ObjInstitution } from "@/api/generated";

interface MembersTabProps {
  members: ObjUser[];
  isLoading: boolean;
  institution: ObjInstitution | null | undefined;
}

export function MembersTab({
  members,
  isLoading,
  institution,
}: MembersTabProps) {
  const { t } = useTranslation("common");
  const api = useRequiredAuthenticatedApi();
  const queryClient = useQueryClient();
  const { backendUser, retryBackendFetch } = useAuth();

  const [userToPromote, setUserToPromote] = useState<ObjUser | null>(null);
  const [userToDemote, setUserToDemote] = useState<ObjUser | null>(null);
  const [userToRemove, setUserToRemove] = useState<ObjUser | null>(null);

  const [
    inviteTeacherModalOpened,
    { open: openInviteTeacherModal, close: closeInviteTeacherModal },
  ] = useDisclosure(false);
  const [
    inviteUserModalOpened,
    { open: openInviteUserModal, close: closeInviteUserModal },
  ] = useDisclosure(false);
  const [inviteTeacherError, setInviteTeacherError] = useState<string | null>(
    null,
  );
  const [inviteUserError, setInviteUserError] = useState<string | null>(null);
  const [
    promoteModalOpened,
    { open: openPromoteModal, close: closePromoteModal },
  ] = useDisclosure(false);
  const [
    demoteModalOpened,
    { open: openDemoteModal, close: closeDemoteModal },
  ] = useDisclosure(false);
  const [
    removeModalOpened,
    { open: openRemoveModal, close: closeRemoveModal },
  ] = useDisclosure(false);
  const [leaveModalOpened, { open: openLeaveModal, close: closeLeaveModal }] =
    useDisclosure(false);
  const [leaveError, setLeaveError] = useState<string | null>(null);

  const institutionId = getUserInstitutionId(backendUser);
  const isHead = hasRole(backendUser, Role.Head);
  const isStaff = hasRole(backendUser, Role.Staff);
  const isIndividual = hasRole(backendUser, Role.Individual);
  const canSeeEmails = isAtLeastHead(backendUser) || isStaff;
  const canSeeRoles = !isIndividual;

  // Helper to extract error message from API error
  const getErrorMessage = (error: unknown): string => {
    if (error && typeof error === "object" && "error" in error) {
      const apiError = error as { error?: { message?: string } };
      const message = apiError.error?.message || "";
      if (message.includes("pending invite already exists")) {
        return t("myOrganization.inviteAlreadyExists");
      }
      return message || t("myOrganization.inviteError");
    }
    return t("myOrganization.inviteError");
  };

  // Mutations
  const inviteTeacherMutation = useMutation({
    mutationFn: async (email: string) => {
      if (!institutionId) throw new Error("No institution");
      await api.invites.institutionCreate({
        institutionId,
        role: "staff",
        invitedEmail: email,
      });
    },
    onSuccess: () => {
      setInviteTeacherError(null);
      closeInviteTeacherModal();
      queryClient.invalidateQueries({ queryKey: queryKeys.invites });
      queryClient.invalidateQueries({
        queryKey: queryKeys.institutionInvites(institutionId!),
      });
    },
    onError: (error) => {
      setInviteTeacherError(getErrorMessage(error));
    },
  });

  const inviteUserMutation = useMutation({
    mutationFn: async (email: string) => {
      if (!institutionId) throw new Error("No institution");
      await api.invites.institutionCreate({
        institutionId,
        role: "individual",
        invitedEmail: email,
      });
    },
    onSuccess: () => {
      setInviteUserError(null);
      closeInviteUserModal();
      queryClient.invalidateQueries({ queryKey: queryKeys.invites });
      queryClient.invalidateQueries({
        queryKey: queryKeys.institutionInvites(institutionId!),
      });
    },
    onError: (error) => {
      setInviteUserError(getErrorMessage(error));
    },
  });

  const promoteMutation = useMutation({
    mutationFn: async (userId: string) => {
      if (!institutionId) throw new Error("No institution");
      await api.users.roleCreate(userId, { role: "head", institutionId });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.institutionMembers(institutionId!),
      });
      closePromoteModal();
      setUserToPromote(null);
    },
  });

  const demoteMutation = useMutation({
    mutationFn: async (userId: string) => {
      if (!institutionId) throw new Error("No institution");
      await api.users.roleCreate(userId, { role: "staff", institutionId });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.institutionMembers(institutionId!),
      });
      closeDemoteModal();
      setUserToDemote(null);
    },
  });

  const removeMemberMutation = useMutation({
    mutationFn: async (userId: string) => {
      if (!institutionId) throw new Error("No institution");
      await api.institutions.membersDelete(institutionId, userId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.institutionMembers(institutionId!),
      });
      closeRemoveModal();
      setUserToRemove(null);
    },
  });

  const navigate = useNavigate();

  const leaveOrganizationMutation = useMutation({
    mutationFn: async () => {
      if (!institutionId || !backendUser?.id)
        throw new Error("No institution or user");
      await api.institutions.membersDelete(institutionId, backendUser.id);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.institutionMembers(institutionId!),
      });
      queryClient.invalidateQueries({ queryKey: queryKeys.backendUser });
      retryBackendFetch(); // Refresh user data to update header
      setLeaveError(null);
      closeLeaveModal();
      navigate({ to: "/dashboard" });
    },
    onError: (error: unknown) => {
      if (error && typeof error === "object" && "error" in error) {
        const apiError = error as {
          error?: { code?: string; message?: string };
        };
        if (apiError.error?.code === "last_head") {
          setLeaveError(t("myOrganization.lastHeadError"));
          return;
        }
      }
      setLeaveError(t("myOrganization.leaveError"));
    },
  });

  // Handlers
  const handlePromote = useCallback(
    (user: ObjUser) => {
      setUserToPromote(user);
      openPromoteModal();
    },
    [openPromoteModal],
  );

  const handleDemote = useCallback(
    (user: ObjUser) => {
      setUserToDemote(user);
      openDemoteModal();
    },
    [openDemoteModal],
  );

  const handleRemove = useCallback(
    (user: ObjUser) => {
      setUserToRemove(user);
      openRemoveModal();
    },
    [openRemoveModal],
  );

  return (
    <>
      <Stack gap="lg">
        {/* Action buttons */}
        <Group gap="sm" wrap="wrap">
          {isHead && (
            <ActionButton onClick={openInviteTeacherModal} size="sm">
              {t("myOrganization.inviteTeacher")}
            </ActionButton>
          )}
          {(isHead || isStaff) && (
            <ActionButton onClick={openInviteUserModal} size="sm">
              {t("myOrganization.inviteUser")}
            </ActionButton>
          )}
        </Group>

        {/* Members List */}
        <MembersList
          members={members}
          isLoading={isLoading}
          currentUserId={backendUser?.id}
          canSeeEmails={canSeeEmails}
          canSeeRoles={canSeeRoles}
          isHead={isHead}
          isStaff={isStaff}
          onPromote={handlePromote}
          onDemote={handleDemote}
          onRemove={handleRemove}
        />

        {/* Pending Invites */}
        {(isHead || isStaff) && institutionId && (
          <Card shadow="sm" padding="lg" radius="md" withBorder>
            <Title order={3} mb="md">
              {t("admin.invites.title")}
            </Title>
            <InvitesList
              institutionId={institutionId}
              showInstitutionColumn={false}
            />
          </Card>
        )}

        {/* Leave Organization */}
        <Group justify="flex-start">
          <ActionButton onClick={openLeaveModal} color="red" size="sm">
            {t("myOrganization.leaveOrganization")}
          </ActionButton>
        </Group>
      </Stack>

      {/* Modals */}
      <InviteModal
        opened={inviteTeacherModalOpened}
        onClose={() => {
          setInviteTeacherError(null);
          closeInviteTeacherModal();
        }}
        title={t("myOrganization.inviteTeacherTitle")}
        description={t("myOrganization.inviteTeacherDescription")}
        onSubmit={(email) => inviteTeacherMutation.mutate(email)}
        isLoading={inviteTeacherMutation.isPending}
        error={inviteTeacherError}
      />

      <InviteModal
        opened={inviteUserModalOpened}
        onClose={() => {
          setInviteUserError(null);
          closeInviteUserModal();
        }}
        title={t("myOrganization.inviteUserTitle")}
        description={t("myOrganization.inviteUserDescription")}
        onSubmit={(email) => inviteUserMutation.mutate(email)}
        isLoading={inviteUserMutation.isPending}
        error={inviteUserError}
      />

      <ConfirmationModal
        opened={promoteModalOpened}
        onClose={closePromoteModal}
        title={t("myOrganization.promoteTitle")}
        message={t("myOrganization.promoteConfirm", {
          name: userToPromote?.name,
          email: userToPromote?.email || t("myOrganization.noEmail"),
        })}
        warning={t("myOrganization.promoteWarning")}
        warningColor="yellow"
        confirmIcon={<IconCrown size={16} />}
        confirmColor="violet"
        onConfirm={() =>
          userToPromote?.id && promoteMutation.mutate(userToPromote.id)
        }
        isLoading={promoteMutation.isPending}
      />

      <ConfirmationModal
        opened={demoteModalOpened}
        onClose={closeDemoteModal}
        title={t("myOrganization.demoteTitle")}
        message={t("myOrganization.demoteConfirm", {
          name: userToDemote?.name,
          email: userToDemote?.email || t("myOrganization.noEmail"),
        })}
        confirmIcon={<IconCrownOff size={16} />}
        confirmColor="gray"
        onConfirm={() =>
          userToDemote?.id && demoteMutation.mutate(userToDemote.id)
        }
        isLoading={demoteMutation.isPending}
      />

      <ConfirmationModal
        opened={removeModalOpened}
        onClose={closeRemoveModal}
        title={t("myOrganization.removeTitle")}
        message={t("myOrganization.removeConfirm", {
          name: userToRemove?.name,
        })}
        warning={t("myOrganization.removeWarning")}
        warningColor="orange"
        confirmIcon={<IconTrash size={16} />}
        confirmColor="red"
        onConfirm={() =>
          userToRemove?.id && removeMemberMutation.mutate(userToRemove.id)
        }
        isLoading={removeMemberMutation.isPending}
      />

      <ConfirmationModal
        opened={leaveModalOpened}
        onClose={() => {
          setLeaveError(null);
          closeLeaveModal();
        }}
        title={t("myOrganization.leaveTitle")}
        message={t("myOrganization.leaveConfirm", { name: institution?.name })}
        warning={t("myOrganization.leaveWarning")}
        warningColor="red"
        confirmIcon={<IconLogout size={16} />}
        confirmColor="red"
        onConfirm={() => leaveOrganizationMutation.mutate()}
        isLoading={leaveOrganizationMutation.isPending}
        error={leaveError}
      />
    </>
  );
}

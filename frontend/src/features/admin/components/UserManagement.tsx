import type { ReactNode } from "react";
import {
  Alert,
  Badge,
  Card,
  Container,
  Group,
  Modal,
  Stack,
  TextInput,
  Tooltip,
} from "@mantine/core";
import {
  IconAlertCircle,
  IconEdit,
  IconShieldCheck,
  IconShieldOff,
  IconTrash,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { UserListSort } from "@/api/hooks";
import type { UserView } from "@/api/types";
import { ListTable, type ListColumn } from "@components/ListTable";
import { ActionButton, DangerButton, GenericIconButton, TextButton } from "@components/buttons";
import { BodyText, HelperText, PageTitle } from "@components/typography";
import { getRoleColor, useTranslateRole } from "@/common/lib/roles";
import { formatDate } from "@/common/lib/formatters";
import { USER_LIST_PRESET, type UserColumnKey } from "../lib/userListPresets";
import { useUserManagement, type UserManagementState } from "../hooks/useUserManagement";

type UserColumn = ListColumn<UserView, UserListSort>;

/** The admins' user list: search, guests in or out, orders, and per user the admin actions. */
export function UserManagement() {
  const { t, i18n } = useTranslation("common");
  const state = useUserManagement();
  const translateRole = useTranslateRole(t("admin.users.noRole"));
  const { list } = state;

  // registration completes an account; until then it is in sign-up and holds no role
  const inSignUp = (user: UserView) => user.registered === false;

  const roleBadge = (user: UserView, size?: "sm") =>
    !inSignUp(user) && user.role?.role ? (
      <Badge variant="light" color={getRoleColor(user.role.role)} size={size}>
        {translateRole(user.role.role)}
      </Badge>
    ) : inSignUp(user) ? (
      <Badge variant="outline" color="gray" size={size}>
        {t("admin.users.signingUp")}
      </Badge>
    ) : (
      <HelperText>{t("admin.users.noRole")}</HelperText>
    );

  const nameWithMe = (user: UserView) => (
    <Group gap="xs" wrap="nowrap">
      {inSignUp(user) ? (
        <HelperText>{t("admin.users.signUpInProgress")}</HelperText>
      ) : (
        <BodyText size="sm" fw={500}>
          {user.name}
        </BodyText>
      )}
      {state.isCurrentUser(user) && (
        <Badge size="xs" variant="filled" color="violet">
          {t("admin.users.me")}
        </Badge>
      )}
    </Group>
  );

  const joined = (user: UserView) => formatDate(user.meta?.createdAt, i18n.language) || "-";

  const columns: Record<UserColumnKey, UserColumn> = {
    name: { key: "name", header: t("admin.users.name"), sortBy: "name", render: nameWithMe },
    email: {
      key: "email",
      header: t("admin.users.email"),
      sortBy: "email",
      visibleFrom: "md",
      render: (user) => <HelperText>{user.email || "-"}</HelperText>,
    },
    role: {
      key: "role",
      header: t("admin.users.role"),
      sortBy: "role",
      render: (user) => roleBadge(user),
    },
    organisation: {
      key: "organisation",
      header: t("admin.users.organization"),
      sortBy: "organisation",
      visibleFrom: "lg",
      render: (user) => <BodyText size="sm">{user.role?.institution?.name || "-"}</BodyText>,
    },
    joined: {
      key: "joined",
      header: t("admin.users.joined"),
      sortBy: "createdAt",
      width: 120,
      render: (user) => <HelperText>{joined(user)}</HelperText>,
    },
    actions: {
      key: "actions",
      header: t("admin.users.actions"),
      width: 160,
      interactive: true,
      render: (user) => <UserActions user={user} state={state} />,
    },
  };

  const renderCard = (user: UserView) => (
    <Card
      withBorder
      padding="sm"
      radius="md"
      bg={state.isCurrentUser(user) ? "violet.0" : undefined}
    >
      <Stack gap="xs">
        <Group justify="space-between" wrap="nowrap">
          {nameWithMe(user)}
          {user.role?.role && roleBadge(user, "sm")}
        </Group>
        {user.email && <HelperText>{user.email}</HelperText>}
        {user.role?.institution?.name && <HelperText>{user.role.institution.name}</HelperText>}
        <Group justify="space-between">
          <HelperText>
            {t("admin.users.joined")}: {joined(user)}
          </HelperText>
          <UserActions user={user} state={state} size="sm" />
        </Group>
      </Stack>
    </Card>
  );

  return (
    <Container size="xl" py="xl">
      <Stack gap="lg">
        <PageTitle>{t("admin.users.title")}</PageTitle>

        <ListTable
          list={list}
          columns={USER_LIST_PRESET.columns.map((key) => columns[key])}
          getRowKey={(user) => user.id}
          renderCard={renderCard}
          getRowAccent={(user) => (state.isCurrentUser(user) ? "violet" : undefined)}
          empty={{
            title: list.query.search ? t("admin.users.noResults") : t("admin.users.empty"),
          }}
          errorMessage={t("admin.users.loadError")}
        />
      </Stack>

      <Modal
        opened={state.deleteModal.opened}
        onClose={state.deleteModal.close}
        title={t("admin.users.deleteTitle")}
      >
        <Stack gap="md">
          <BodyText>{t("admin.users.deleteConfirm", { name: state.deleteModal.user?.name })}</BodyText>
          <ConfirmRow
            onCancel={state.deleteModal.close}
            onConfirm={state.deleteModal.confirm}
            loading={state.deleteModal.loading}
            danger
            icon={<IconTrash size={16} />}
            label={t("delete")}
          />
        </Stack>
      </Modal>

      <Modal
        opened={state.promoteModal.opened}
        onClose={state.promoteModal.close}
        title={t("admin.users.promoteTitle")}
      >
        <Stack gap="md">
          <BodyText>
            {t("admin.users.promoteConfirm", {
              name: state.promoteModal.user?.name,
              email: state.promoteModal.user?.email || t("admin.users.noEmail"),
            })}
          </BodyText>
          <Alert color="yellow" icon={<IconAlertCircle size={16} />}>
            {t("admin.users.promoteWarning")}
          </Alert>
          <ConfirmRow
            onCancel={state.promoteModal.close}
            onConfirm={state.promoteModal.confirm}
            loading={state.promoteModal.loading}
            icon={<IconShieldCheck size={16} />}
            label={t("admin.users.makeAdmin")}
          />
        </Stack>
      </Modal>

      <Modal
        opened={state.removeAdminModal.opened}
        onClose={state.removeAdminModal.close}
        title={t("admin.users.removeAdminTitle")}
      >
        <Stack gap="md">
          <BodyText>
            {t("admin.users.removeAdminConfirm", {
              name: state.removeAdminModal.user?.name,
              email: state.removeAdminModal.user?.email || t("admin.users.noEmail"),
            })}
          </BodyText>
          <Alert color="orange" icon={<IconAlertCircle size={16} />}>
            {t("admin.users.removeAdminWarning")}
          </Alert>
          <ConfirmRow
            onCancel={state.removeAdminModal.close}
            onConfirm={state.removeAdminModal.confirm}
            loading={state.removeAdminModal.loading}
            danger
            icon={<IconShieldOff size={16} />}
            label={t("admin.users.removeAdmin")}
          />
        </Stack>
      </Modal>

      <Modal
        opened={state.editModal.opened}
        onClose={state.editModal.close}
        title={t("admin.users.editTitle")}
      >
        <Stack gap="md">
          <HelperText>
            {t("admin.users.editingUser", { name: state.editModal.user?.name })}
          </HelperText>
          <TextInput
            label={t("admin.users.name")}
            placeholder={t("admin.users.namePlaceholder")}
            value={state.editModal.name}
            onChange={(e) => state.editModal.setName(e.currentTarget.value)}
          />
          <ConfirmRow
            onCancel={state.editModal.close}
            onConfirm={state.editModal.confirm}
            loading={state.editModal.loading}
            disabled={state.editModal.unchanged}
            icon={<IconEdit size={16} />}
            label={t("admin.users.editName")}
          />
        </Stack>
      </Modal>
    </Container>
  );
}

/** A user's admin actions, as far as the caller may take them. */
function UserActions({
  user,
  state,
  size,
}: {
  user: UserView;
  state: UserManagementState;
  size?: "sm";
}) {
  const { t } = useTranslation("common");
  const iconSize = size === "sm" ? 14 : 16;
  const action = (label: string, color: string, icon: ReactNode, onClick: () => void) => (
    <Tooltip label={label} withArrow>
      <GenericIconButton icon={icon} color={color} size={size} onClick={onClick} aria-label={label} />
    </Tooltip>
  );
  return (
    <Group gap="xs" wrap="nowrap">
      {state.canPromote(user) &&
        action(t("admin.users.makeAdmin"), "green", <IconShieldCheck size={iconSize} />, () =>
          state.requestPromote(user),
        )}
      {state.canRemoveAdmin(user) &&
        action(t("admin.users.removeAdmin"), "red", <IconShieldOff size={iconSize} />, () =>
          state.requestRemoveAdmin(user),
        )}
      {state.canEdit(user) &&
        action(t("admin.users.editName"), "blue", <IconEdit size={iconSize} />, () =>
          state.requestEdit(user),
        )}
      {state.canDelete(user) &&
        action(t("delete"), "red", <IconTrash size={iconSize} />, () => state.requestDelete(user))}
    </Group>
  );
}

/** A dialog's cancel and confirm, as the user dialogs have them. */
function ConfirmRow({
  onCancel,
  onConfirm,
  loading,
  disabled,
  danger,
  icon,
  label,
}: {
  onCancel: () => void;
  onConfirm: () => void;
  loading: boolean;
  disabled?: boolean;
  /** The action takes something away (deleting, demoting). */
  danger?: boolean;
  icon: ReactNode;
  label: string;
}) {
  const { t } = useTranslation("common");
  const Confirm = danger ? DangerButton : ActionButton;
  return (
    <Group justify="flex-end">
      <TextButton onClick={onCancel}>{t("cancel")}</TextButton>
      <Confirm onClick={onConfirm} loading={loading} disabled={disabled} leftSection={icon}>
        {label}
      </Confirm>
    </Group>
  );
}

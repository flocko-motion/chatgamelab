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
  IconLock,
  IconLockOpen,
  IconPlus,
  IconSend,
  IconTrash,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { OrganisationListSort } from "@/api/hooks";
import type { OrganisationView } from "@/api/types";
import { ListTable, type ListColumn } from "@components/ListTable";
import {
  ActionButton,
  DangerButton,
  GenericIconButton,
  PlusIconButton,
  TextButton,
} from "@components/buttons";
import { BodyText, CardTitle, HelperText, PageTitle } from "@components/typography";
import { useResponsiveDesign } from "@/common/hooks/useResponsiveDesign";
import {
  ORGANISATION_LIST_PRESET,
  type OrganisationColumnKey,
} from "../lib/organisationListPresets";
import { headNames, operatorAndCity } from "../lib/organizationHeads";
import {
  useOrganizationsManagement,
  type OrganizationsManagementState,
} from "../hooks/useOrganizationsManagement";
import { InvitesList } from "./InvitesList";
import { OrganisationMembers } from "./OrganisationMembers";
import { OrganizationStatusBadges } from "./OrganizationStatusBadges";
import { OrganizationSuspendModal } from "./OrganizationSuspendModal";
import { RecentRegistrationsList } from "./RecentRegistrationsList";

type OrganisationColumn = ListColumn<OrganisationView, OrganisationListSort>;

/**
 * The admins' organisation page: every organisation (its query in the URL; a
 * click on one opens its members), the recent self-registrations and every
 * invite (both embedded, their queries kept locally so the lists don't share
 * the URL).
 */
export function OrganizationsManagement() {
  const { t } = useTranslation("common");
  const { isMobile } = useResponsiveDesign();
  const state = useOrganizationsManagement();
  const { list } = state;

  const membersBadge = (org: OrganisationView, size?: "sm") => (
    <Badge variant="light" color="blue" size={size}>
      {org.members?.length || 0} {t("admin.organizations.membersCount")}
    </Badge>
  );

  const columns: Record<OrganisationColumnKey, OrganisationColumn> = {
    name: {
      key: "name",
      header: t("admin.organizations.name"),
      sortBy: "name",
      render: (org) => (
        <Stack gap={0}>
          <BodyText size="sm" fw={500}>
            {org.name}
          </BodyText>
          {operatorAndCity(org) && <HelperText>{operatorAndCity(org)}</HelperText>}
        </Stack>
      ),
    },
    heads: {
      key: "heads",
      header: t("admin.organizations.heads"),
      visibleFrom: "md",
      render: (org) => <BodyText size="sm">{headNames(org) || "–"}</BodyText>,
    },
    status: {
      key: "status",
      header: t("admin.organizations.status"),
      render: (org) => <OrganizationStatusBadges org={org} />,
    },
    members: {
      key: "members",
      header: t("admin.organizations.members"),
      render: (org) => membersBadge(org),
    },
    actions: {
      key: "actions",
      header: t("admin.organizations.actions"),
      width: 190,
      interactive: true,
      render: (org) => <OrganisationActions org={org} state={state} />,
    },
  };

  const renderCard = (org: OrganisationView) => (
    <Card withBorder padding="sm" radius="md">
      <Stack gap="xs">
        <Group justify="space-between" wrap="nowrap">
          <BodyText fw={600}>{org.name}</BodyText>
          {membersBadge(org, "sm")}
        </Group>
        {operatorAndCity(org) && <HelperText>{operatorAndCity(org)}</HelperText>}
        {headNames(org) && (
          <BodyText size="sm">
            {t("admin.organizations.heads")}: {headNames(org)}
          </BodyText>
        )}
        <OrganizationStatusBadges org={org} />
        <Group justify="flex-end">
          <OrganisationActions org={org} state={state} size="sm" />
        </Group>
      </Stack>
    </Card>
  );

  const createAction = isMobile ? (
    <Tooltip label={t("admin.organizations.create")} withArrow>
      <PlusIconButton
        onClick={state.openCreate}
        variant="filled"
        aria-label={t("admin.organizations.create")}
      />
    </Tooltip>
  ) : (
    <ActionButton leftSection={<IconPlus size={16} />} onClick={state.openCreate}>
      {t("admin.organizations.create")}
    </ActionButton>
  );

  const { createModal, editModal, inviteModal, deleteModal } = state;

  return (
    <Container size="xl" py="xl">
      <Stack gap="lg">
        <PageTitle>{t("admin.organizations.title")}</PageTitle>

        <ListTable
          list={list}
          columns={ORGANISATION_LIST_PRESET.columns.map((key) => columns[key])}
          getRowKey={(org) => org.id}
          renderCard={renderCard}
          renderExpanded={(org) => <OrganisationMembers org={org} />}
          actions={createAction}
          empty={{ title: t("admin.organizations.empty") }}
          errorMessage={t("admin.organizations.loadError")}
        />

        {/* Organisations that registered themselves lately */}
        <Card shadow="sm" padding="lg" radius="md" withBorder>
          <Stack gap="xs">
            <CardTitle>{t("admin.registrations.title")}</CardTitle>
            <HelperText>{t("admin.registrations.description")}</HelperText>
            <RecentRegistrationsList />
          </Stack>
        </Card>

        <Card shadow="sm" padding="lg" radius="md" withBorder>
          <Stack gap="md">
            <CardTitle>{t("admin.invites.title")}</CardTitle>
            <InvitesList showInstitutionColumn />
          </Stack>
        </Card>
      </Stack>

      <Modal
        opened={createModal.opened}
        onClose={createModal.close}
        title={t("admin.organizations.createTitle")}
      >
        <Stack gap="md">
          <TextInput
            label={t("admin.organizations.name")}
            placeholder={t("admin.organizations.namePlaceholder")}
            value={createModal.name}
            onChange={(e) => createModal.setName(e.currentTarget.value)}
          />
          <HelperText>{t("admin.organizations.detailsOptional")}</HelperText>
          <TextInput
            label={t("admin.organizations.operator")}
            value={createModal.details.operator}
            onChange={(e) => createModal.setDetail("operator", e.currentTarget.value)}
          />
          <TextInput
            label={t("admin.organizations.street")}
            value={createModal.details.street}
            onChange={(e) => createModal.setDetail("street", e.currentTarget.value)}
          />
          <Group grow>
            <TextInput
              label={t("admin.organizations.postalCode")}
              value={createModal.details.postalCode}
              onChange={(e) => createModal.setDetail("postalCode", e.currentTarget.value)}
            />
            <TextInput
              label={t("admin.organizations.city")}
              value={createModal.details.city}
              onChange={(e) => createModal.setDetail("city", e.currentTarget.value)}
            />
          </Group>
          <Group justify="flex-end">
            <TextButton onClick={createModal.close}>{t("cancel")}</TextButton>
            <ActionButton
              onClick={createModal.confirm}
              loading={createModal.loading}
              disabled={!createModal.canConfirm}
            >
              {t("create")}
            </ActionButton>
          </Group>
        </Stack>
      </Modal>

      <Modal
        opened={editModal.opened}
        onClose={editModal.close}
        title={t("admin.organizations.editTitle")}
      >
        <Stack gap="md">
          <TextInput
            label={t("admin.organizations.name")}
            value={editModal.name}
            onChange={(e) => editModal.setName(e.currentTarget.value)}
          />
          <Group justify="flex-end">
            <TextButton onClick={editModal.close}>{t("cancel")}</TextButton>
            <ActionButton
              onClick={editModal.confirm}
              loading={editModal.loading}
              disabled={!editModal.canConfirm}
            >
              {t("save")}
            </ActionButton>
          </Group>
        </Stack>
      </Modal>

      <Modal
        opened={inviteModal.opened}
        onClose={inviteModal.close}
        title={t("admin.organizations.inviteTitle")}
      >
        <Stack gap="md">
          <HelperText>
            {t("admin.organizations.inviteDescription", { name: inviteModal.org?.name })}
          </HelperText>
          <TextInput
            label={t("admin.organizations.email")}
            placeholder={t("admin.organizations.emailPlaceholder")}
            type="email"
            value={inviteModal.email}
            onChange={(e) => inviteModal.setEmail(e.currentTarget.value)}
          />
          {inviteModal.error && (
            <Alert color="red" icon={<IconAlertCircle size={16} />}>
              {inviteModal.error}
            </Alert>
          )}
          <Group justify="flex-end">
            <TextButton onClick={inviteModal.close}>{t("cancel")}</TextButton>
            <ActionButton
              onClick={inviteModal.confirm}
              loading={inviteModal.loading}
              disabled={!inviteModal.canConfirm}
              leftSection={<IconSend size={16} />}
            >
              {t("admin.organizations.sendInvite")}
            </ActionButton>
          </Group>
        </Stack>
      </Modal>

      <OrganizationSuspendModal org={state.suspendModal.org} onClose={state.suspendModal.close} />

      <Modal
        opened={deleteModal.opened}
        onClose={deleteModal.close}
        title={t("admin.organizations.deleteTitle")}
      >
        <Stack gap="md">
          <BodyText>
            {t("admin.organizations.deleteConfirm", { name: deleteModal.org?.name })}
          </BodyText>
          <Group justify="flex-end">
            <TextButton onClick={deleteModal.close}>{t("cancel")}</TextButton>
            <DangerButton onClick={deleteModal.confirm} loading={deleteModal.loading}>
              {t("delete")}
            </DangerButton>
          </Group>
        </Stack>
      </Modal>
    </Container>
  );
}

/** An organisation's admin actions, as far as the server's flags allow them. */
function OrganisationActions({
  org,
  state,
  size,
}: {
  org: OrganisationView;
  state: OrganizationsManagementState;
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
      {action(t("admin.organizations.sendInvite"), "blue", <IconSend size={iconSize} />, () =>
        state.requestInvite(org),
      )}
      {state.canEdit(org) &&
        action(t("edit"), "gray", <IconEdit size={iconSize} />, () => state.requestEdit(org))}
      {state.canSuspend(org) &&
        action(
          org.suspendedAt ? t("admin.organizations.lift") : t("admin.organizations.suspend"),
          org.suspendedAt ? "green" : "orange",
          org.suspendedAt ? <IconLockOpen size={iconSize} /> : <IconLock size={iconSize} />,
          () => state.requestSuspension(org),
        )}
      {state.canDelete(org) &&
        action(t("delete"), "red", <IconTrash size={iconSize} />, () => state.requestDelete(org))}
    </Group>
  );
}

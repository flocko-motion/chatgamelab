import { Card, Group, Stack, Badge, Tooltip } from "@mantine/core";
import { IconCrown, IconCrownOff } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { MemberListSort } from "@/api/hooks";
import type { UserView } from "@/api/types";
import { ListTable, type ListColumn } from "@components/ListTable";
import { DeleteIconButton, GenericIconButton } from "@components/buttons";
import { BodyText, HelperText } from "@components/typography";
import { getRoleColor, useTranslateRole } from "@/common/lib/roles";
import type { MemberColumnKey } from "../lib/memberListPresets";
import type { MembersTabState } from "../hooks/useMembersTab";

type MemberColumn = ListColumn<UserView, MemberListSort>;

/** The organisation's members: search, order, pages; each with its actions. */
export function MembersList({ state }: { state: MembersTabState }) {
  const { t } = useTranslation("common");
  const translateRole = useTranslateRole(t("myOrganization.noRole"));
  const { list } = state;

  const nameWithMe = (member: UserView, strong = false) => (
    <Group gap="xs" wrap="nowrap" style={{ minWidth: 0 }}>
      <BodyText size="sm" fw={strong ? 600 : undefined}>
        {member.name}
      </BodyText>
      {state.isCurrentUser(member) && (
        <Badge size="xs" variant="filled" color="violet">
          {t("myOrganization.me")}
        </Badge>
      )}
    </Group>
  );

  const roleBadge = (member: UserView) => (
    <Badge color={getRoleColor(member.role?.role)} variant="light" size="sm">
      {translateRole(member.role?.role)}
    </Badge>
  );

  const actions = (member: UserView) => (
    <Group gap="xs" wrap="nowrap">
      {state.canPromote(member) && (
        <Tooltip label={t("myOrganization.promoteToHead")}>
          <GenericIconButton
            icon={<IconCrown size={16} />}
            color="violet"
            onClick={() => state.requestPromote(member)}
            aria-label={t("myOrganization.promoteToHead")}
          />
        </Tooltip>
      )}
      {state.canDemote(member) && (
        <Tooltip label={t("myOrganization.demoteToStaff")}>
          <GenericIconButton
            icon={<IconCrownOff size={16} />}
            onClick={() => state.requestDemote(member)}
            aria-label={t("myOrganization.demoteToStaff")}
          />
        </Tooltip>
      )}
      {state.canRemove(member) && (
        <Tooltip label={t("myOrganization.removeMember")}>
          <DeleteIconButton
            onClick={() => state.requestRemove(member)}
            aria-label={t("myOrganization.removeMember")}
          />
        </Tooltip>
      )}
    </Group>
  );

  const allColumns: Record<MemberColumnKey, MemberColumn> = {
    name: {
      key: "name",
      header: t("myOrganization.name"),
      sortBy: "name",
      render: (member) => nameWithMe(member),
    },
    email: {
      key: "email",
      header: t("myOrganization.email"),
      sortBy: "email",
      visibleFrom: "sm",
      render: (member) => <HelperText>{member.email || "-"}</HelperText>,
    },
    role: {
      key: "role",
      header: t("myOrganization.role"),
      sortBy: "role",
      width: 140,
      render: roleBadge,
    },
    actions: {
      key: "actions",
      header: t("myOrganization.actions"),
      width: 120,
      interactive: true,
      render: actions,
    },
  };
  const columns = state.columns.map((key) => allColumns[key]);
  const showsEmail = state.columns.includes("email");

  const renderCard = (member: UserView) => (
    <Card
      withBorder
      padding="sm"
      radius="md"
      bg={state.isCurrentUser(member) ? "violet.0" : undefined}
    >
      <Stack gap="xs">
        <Group justify="space-between" wrap="nowrap">
          {nameWithMe(member, true)}
          {state.canSeeRoles && member.role?.role && roleBadge(member)}
        </Group>
        {showsEmail && member.email && <HelperText>{member.email}</HelperText>}
        {state.columns.includes("actions") && (
          <Group gap="xs" justify="flex-end">
            {actions(member)}
          </Group>
        )}
      </Stack>
    </Card>
  );

  return (
    <ListTable
      list={list}
      columns={columns}
      getRowKey={(member) => member.id}
      renderCard={renderCard}
      getRowAccent={(member) => (state.isCurrentUser(member) ? "violet" : undefined)}
      empty={{
        title: state.narrowed ? t("myOrganization.noResults") : t("myOrganization.empty"),
      }}
      errorMessage={t("myOrganization.loadError")}
    />
  );
}

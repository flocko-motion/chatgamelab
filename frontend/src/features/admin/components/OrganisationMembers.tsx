import { Badge, Box, Card, Group, Stack, Tooltip } from "@mantine/core";
import { modals } from "@mantine/modals";
import { IconTrash } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { MemberListSort } from "@/api/hooks";
import type { OrganisationView, UserView } from "@/api/types";
import { ListTable, type ListColumn } from "@components/ListTable";
import { GenericIconButton } from "@components/buttons";
import { BodyText, HelperText, Label } from "@components/typography";
import { useTranslateRole } from "@/common/lib/roles";
import { MEMBER_LIST_PRESET, type MemberColumnKey } from "../lib/memberListPresets";
import { useOrganizationMembers } from "../hooks/useOrganizationMembers";

type MemberColumn = ListColumn<UserView, MemberListSort>;

/** An organisation's members, embedded in its opened row, with removing a member. */
export function OrganisationMembers({ org }: { org: OrganisationView }) {
  const { t } = useTranslation("common");
  const translateRole = useTranslateRole();
  const { list, remove, removing } = useOrganizationMembers(org.id);

  const requestRemove = (member: UserView) =>
    modals.openConfirmModal({
      title: t("admin.organizations.removeMember"),
      children: (
        <BodyText size="sm">
          {t("admin.organizations.removeMemberConfirm", {
            name: member.name || member.email || "?",
          })}
        </BodyText>
      ),
      labels: { confirm: t("admin.organizations.removeMember"), cancel: t("cancel") },
      confirmProps: { color: "red" },
      onConfirm: () => remove(member),
    });

  const removeButton = (member: UserView) => (
    <Tooltip label={t("admin.organizations.removeMember")} withArrow>
      <GenericIconButton
        icon={<IconTrash size={14} />}
        color="red"
        size="sm"
        onClick={() => requestRemove(member)}
        loading={removing}
        aria-label={t("admin.organizations.removeMember")}
      />
    </Tooltip>
  );

  const roleBadge = (member: UserView) => (
    <Badge size="sm" variant="light">
      {translateRole(member.role?.role)}
    </Badge>
  );

  const columns: Record<MemberColumnKey, MemberColumn> = {
    name: {
      key: "name",
      header: t("admin.organizations.memberName"),
      sortBy: "name",
      render: (member) => <BodyText size="sm">{member.name || "-"}</BodyText>,
    },
    email: {
      key: "email",
      header: t("admin.organizations.memberEmail"),
      sortBy: "email",
      render: (member) => <HelperText>{member.email || "-"}</HelperText>,
    },
    role: {
      key: "role",
      header: t("admin.organizations.memberRole"),
      sortBy: "role",
      render: roleBadge,
    },
    actions: {
      key: "actions",
      header: t("admin.organizations.actions"),
      width: 80,
      interactive: true,
      render: removeButton,
    },
  };

  const renderCard = (member: UserView) => (
    <Card withBorder padding="sm" radius="md">
      <Group justify="space-between" wrap="nowrap">
        <Stack gap={2} style={{ minWidth: 0 }}>
          <BodyText size="sm" fw={500}>
            {member.name || "-"}
          </BodyText>
          {member.email && <HelperText>{member.email}</HelperText>}
        </Stack>
        <Group gap="xs" wrap="nowrap">
          {roleBadge(member)}
          {removeButton(member)}
        </Group>
      </Group>
    </Card>
  );

  return (
    <Box bg="gray.0" p="md">
      <Stack gap="xs">
        <Label>{t("admin.organizations.membersList")}</Label>
        <ListTable
          compact
          list={list}
          columns={MEMBER_LIST_PRESET.columns.map((key) => columns[key])}
          getRowKey={(member) => member.id}
          renderCard={renderCard}
          empty={{ title: t("admin.organizations.noMembers") }}
          errorMessage={t("admin.organizations.membersLoadError")}
        />
      </Stack>
    </Box>
  );
}

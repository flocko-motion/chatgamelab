import { Badge, Card, Group, Stack, Tooltip } from "@mantine/core";
import { modals } from "@mantine/modals";
import { IconTrash } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { InviteListSort } from "@/api/hooks";
import type { InviteCardView } from "@/api/types";
import { ListTable, type ListColumn } from "@components/ListTable";
import { GenericIconButton } from "@components/buttons";
import { BodyText, HelperText } from "@components/typography";
import { useTranslateRole } from "@/common/lib/roles";
import { formatDate } from "@/common/lib/formatters";
import type { InviteColumnKey } from "../lib/inviteListPresets";
import { useInvitesList } from "../hooks/useInvitesList";

const STATUS_COLORS: Record<string, string> = {
  pending: "blue",
  accepted: "green",
  declined: "gray",
  revoked: "red",
  expired: "orange",
};

const statusColor = (status?: string) => STATUS_COLORS[status || "pending"] ?? "gray";

const shownDate = (invite: InviteCardView) => invite.meta.modifiedAt || invite.meta.createdAt;

type InviteColumn = ListColumn<InviteCardView, InviteListSort>;

interface InvitesListProps {
  /** One organisation's invites; without it every invite the caller may see (admins). */
  institutionId?: string;
  /** Show each invite's organisation (the admins' list of every invite). */
  showInstitutionColumn?: boolean;
}

/**
 * An invite list with search and orders, and revoking a pending invite. It
 * shares its page with other lists, so its query stays out of the URL.
 */
export function InvitesList({ institutionId, showInstitutionColumn = false }: InvitesListProps) {
  const { t, i18n } = useTranslation("common");
  const translateRole = useTranslateRole();
  const state = useInvitesList(institutionId, showInstitutionColumn ? "all" : "organisation");
  const { config, list } = state;

  const translateStatus = (status?: string) =>
    t(`admin.invites.statuses.${status || "pending"}`, status || "pending");
  const invited = (invite: InviteCardView) => invite.invitedEmail || invite.invitedUserId || "-";
  const date = (invite: InviteCardView) => formatDate(shownDate(invite), i18n.language) || "-";

  const requestRevoke = (invite: InviteCardView) =>
    modals.openConfirmModal({
      title: t("admin.invites.revokeConfirm.title"),
      children: <BodyText size="sm">{t("admin.invites.revokeConfirm.message")}</BodyText>,
      labels: { confirm: t("admin.invites.revokeConfirm.confirm"), cancel: t("cancel") },
      confirmProps: { color: "red" },
      onConfirm: () => state.revoke(invite),
    });

  const revokeButton = (invite: InviteCardView, size?: "sm") =>
    state.canRevoke(invite) && (
      <Tooltip label={t("admin.invites.revokeConfirm.title")} withArrow>
        <GenericIconButton
          icon={<IconTrash size={size === "sm" ? 14 : 16} />}
          color="red"
          size={size}
          onClick={() => requestRevoke(invite)}
          loading={state.revoking}
          aria-label={t("admin.invites.revokeConfirm.title")}
        />
      </Tooltip>
    );

  const columns: Record<InviteColumnKey, InviteColumn> = {
    organisation: {
      key: "organisation",
      header: t("admin.invites.organization"),
      sortBy: "organisation",
      render: (invite) => <BodyText size="sm">{invite.institutionName || "-"}</BodyText>,
    },
    role: {
      key: "role",
      header: t("admin.invites.role"),
      sortBy: "role",
      render: (invite) => (
        <BodyText size="sm" fw={600}>
          {translateRole(invite.role)}
        </BodyText>
      ),
    },
    invited: {
      key: "invited",
      header: t("admin.invites.invitedEmail"),
      render: (invite) => <BodyText size="sm">{invited(invite)}</BodyText>,
    },
    status: {
      key: "status",
      header: t("admin.invites.status"),
      sortBy: "status",
      render: (invite) => (
        <Badge color={statusColor(invite.status)}>{translateStatus(invite.status)}</Badge>
      ),
    },
    modified: {
      key: "modified",
      header: t("admin.invites.modifiedAt"),
      sortBy: "modifiedAt",
      visibleFrom: "md",
      width: 120,
      render: (invite) => <BodyText size="sm">{date(invite)}</BodyText>,
    },
    actions: {
      key: "actions",
      header: t("admin.invites.actions"),
      width: 90,
      interactive: true,
      render: (invite) => revokeButton(invite),
    },
  };

  const renderCard = (invite: InviteCardView) => (
    <Card withBorder padding="sm" radius="md">
      <Stack gap="xs">
        <Group justify="space-between" wrap="nowrap">
          <BodyText size="sm" fw={600}>
            {invited(invite)}
          </BodyText>
          <Badge size="sm" color={statusColor(invite.status)}>
            {translateStatus(invite.status)}
          </Badge>
        </Group>
        <Group gap="xs" wrap="wrap">
          <Badge size="xs" variant="light">
            {translateRole(invite.role)}
          </Badge>
          {config.columns.includes("organisation") && (
            <Badge size="xs" variant="outline" color="gray">
              {invite.institutionName || "-"}
            </Badge>
          )}
        </Group>
        <Group justify="space-between" align="center">
          <HelperText>{date(invite)}</HelperText>
          {revokeButton(invite, "sm")}
        </Group>
      </Stack>
    </Card>
  );

  return (
    <ListTable
      list={list}
      columns={config.columns.map((key) => columns[key])}
      getRowKey={(invite) => invite.id}
      renderCard={renderCard}
      empty={{
        title: list.query.search ? t("admin.invites.noResults") : t("admin.invites.empty"),
      }}
      errorMessage={t("admin.invites.loadError")}
    />
  );
}

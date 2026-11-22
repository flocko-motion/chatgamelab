import type { ReactNode } from "react";
import { Badge, Card, Group, Modal, Stack, Switch, TextInput, Tooltip } from "@mantine/core";
import {
  IconCalendar,
  IconLink,
  IconLogin,
  IconMail,
  IconPlayerPause,
  IconPlayerPlay,
  IconPlus,
  IconSettings,
  IconTrash,
  IconUser,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { WorkshopListSort } from "@/api/hooks";
import type { WorkshopView } from "@/api/types";
import { ListTable, type ListColumn } from "@components/ListTable";
import {
  ActionButton,
  DeleteIconButton,
  GenericIconButton,
  PlusIconButton,
  TextButton,
} from "@components/buttons";
import { BodyText, HelperText } from "@components/typography";
import { useResponsiveDesign } from "@/common/hooks/useResponsiveDesign";
import { WORKSHOP_COLUMNS, type WorkshopColumnKey } from "../lib/workshopListPresets";
import { useWorkshopsTab, type WorkshopsTabState } from "../hooks/useWorkshopsTab";
import { ConfirmationModal } from "./ConfirmationModal";
import { SingleWorkshopSettings } from "./SingleWorkshopSettings";
import { WorkshopInviteModals } from "./WorkshopInviteModals";

interface WorkshopsTabProps {
  institutionId: string;
  institutionName?: string;
  /** Auto-open the create workshop modal on mount */
  autoCreate?: boolean;
}

type WorkshopColumn = ListColumn<WorkshopView, WorkshopListSort>;

const createdDate = (w: WorkshopView) =>
  w.meta?.createdAt ? new Date(w.meta.createdAt).toLocaleDateString() : "";

/**
 * The organisation's workshops: search, the toggle for active ones, orders and
 * pages. A workshop's settings and participants open in a modal.
 */
export function WorkshopsTab({ institutionId, institutionName, autoCreate }: WorkshopsTabProps) {
  const { t } = useTranslation("common");
  const { isMobile } = useResponsiveDesign();
  const state = useWorkshopsTab({ institutionId, autoCreate });
  const { list, createModal, deleteModal, settingsModal } = state;

  const statusBadge = (w: WorkshopView, size: "xs" | "sm" = "sm") => (
    <Badge color={w.active ? "green" : "gray"} variant="light" size={size}>
      {w.active ? t("myOrganization.workshops.active") : t("myOrganization.workshops.inactive")}
    </Badge>
  );

  const allColumns = useWorkshopColumns(state, statusBadge);
  const columns = WORKSHOP_COLUMNS.map((key) => allColumns[key]);

  const renderCard = (w: WorkshopView) => {
    const created = createdDate(w);
    return (
      <Card shadow="sm" padding="md" radius="md" withBorder>
        <Stack gap="xs">
          <Group justify="space-between" wrap="nowrap" gap="sm">
            <Stack gap={2} style={{ minWidth: 0 }}>
              <BodyText size="sm" fw={500}>
                {w.name}
              </BodyText>
              <Group gap="xs">
                {statusBadge(w, "xs")}
                <Group gap={4}>
                  <IconUser size={12} color="var(--mantine-color-gray-6)" />
                  <HelperText>{w.participants?.length || 0}</HelperText>
                </Group>
                {created && (
                  <Group gap={4}>
                    <IconCalendar size={12} color="var(--mantine-color-gray-6)" />
                    <HelperText>{created}</HelperText>
                  </Group>
                )}
              </Group>
            </Stack>
            <SettingsButton onClick={() => state.openSettings(w)} />
          </Group>
          <Group justify="space-between" wrap="nowrap">
            {w.active ? (
              <ActionButton
                onClick={() => state.enter(w)}
                leftSection={<IconLogin size={14} />}
                size="compact-sm"
                color="accent"
              >
                {t("myOrganization.workshops.enterWorkshop")}
              </ActionButton>
            ) : (
              <div />
            )}
            <WorkshopActionIcons workshop={w} state={state} />
          </Group>
        </Stack>
      </Card>
    );
  };

  const createAction = isMobile ? (
    <Tooltip label={t("myOrganization.workshops.create")} withArrow>
      <PlusIconButton
        onClick={createModal.open}
        variant="filled"
        aria-label={t("myOrganization.workshops.create")}
      />
    </Tooltip>
  ) : (
    <ActionButton onClick={createModal.open} leftSection={<IconPlus size={18} />}>
      {t("myOrganization.workshops.create")}
    </ActionButton>
  );

  return (
    <>
      <Stack gap="lg">
        <HelperText>{t("myOrganization.workshops.subtitle")}</HelperText>

        <ListTable
          list={list}
          columns={columns}
          getRowKey={(w) => w.id}
          renderCard={renderCard}
          onRowClick={state.openSettings}
          actions={createAction}
          empty={{
            title: state.narrowed
              ? t("myOrganization.workshops.noResults")
              : t("myOrganization.workshops.empty"),
          }}
          errorMessage={t("myOrganization.workshops.loadError")}
        />
      </Stack>

      {/* A workshop's settings and participants */}
      <Modal
        opened={!!settingsModal.workshopId}
        onClose={settingsModal.close}
        title={t("myOrganization.workshops.settings")}
        size="xl"
        fullScreen={isMobile}
      >
        {settingsModal.workshopId && (
          <SingleWorkshopSettings
            workshopId={settingsModal.workshopId}
            institutionId={institutionId}
            institutionName={institutionName}
            embedded
          />
        )}
      </Modal>

      <Modal
        opened={createModal.opened}
        onClose={createModal.close}
        title={t("myOrganization.workshops.createTitle")}
        size="md"
      >
        <Stack gap="md">
          <TextInput
            label={t("myOrganization.workshops.nameLabel")}
            placeholder={t("myOrganization.workshops.namePlaceholder")}
            value={createModal.name}
            onChange={(e) => createModal.setName(e.currentTarget.value)}
            required
          />
          <Switch
            label={t("myOrganization.workshops.activeLabel")}
            description={t("myOrganization.workshops.activeDescription")}
            checked={createModal.active}
            onChange={(e) => createModal.setActive(e.currentTarget.checked)}
          />
          <Group justify="flex-end" mt="md">
            <TextButton onClick={createModal.close}>{t("cancel")}</TextButton>
            <ActionButton
              onClick={createModal.submit}
              loading={createModal.loading}
              disabled={!createModal.name.trim()}
            >
              {t("myOrganization.workshops.create")}
            </ActionButton>
          </Group>
        </Stack>
      </Modal>

      <ConfirmationModal
        opened={!!deleteModal.workshop}
        onClose={deleteModal.close}
        title={t("myOrganization.workshops.deleteTitle")}
        message={t("myOrganization.workshops.deleteConfirm", { name: deleteModal.workshop?.name })}
        warning={t("myOrganization.workshops.deleteWarning")}
        warningColor="red"
        confirmIcon={<IconTrash size={16} />}
        confirmColor="red"
        onConfirm={deleteModal.confirm}
        isLoading={deleteModal.loading}
      />

      <WorkshopInviteModals invites={state.invites} />
    </>
  );
}

/** The workshop list's columns, by key. */
function useWorkshopColumns(
  state: WorkshopsTabState,
  statusBadge: (w: WorkshopView) => ReactNode,
): Record<WorkshopColumnKey, WorkshopColumn> {
  const { t } = useTranslation("common");
  return {
    name: {
      key: "name",
      header: t("myOrganization.workshops.nameLabel"),
      sortBy: "name",
      render: (w) => (
        <BodyText size="sm" fw={500}>
          {w.name}
        </BodyText>
      ),
    },
    status: {
      key: "status",
      header: t("myOrganization.workshops.status"),
      width: 110,
      render: statusBadge,
    },
    participants: {
      key: "participants",
      header: t("myOrganization.workshops.participants"),
      sortBy: "participantCount",
      width: 120,
      visibleFrom: "md",
      render: (w) => <HelperText>{w.participants?.length || 0}</HelperText>,
    },
    created: {
      key: "created",
      header: t("games.fields.created"),
      sortBy: "createdAt",
      width: 110,
      visibleFrom: "lg",
      render: (w) => <HelperText>{createdDate(w) || "-"}</HelperText>,
    },
    actions: {
      key: "actions",
      header: t("actions"),
      interactive: true,
      render: (w) => (
        <Group gap="xs" wrap="nowrap" justify="flex-end">
          {w.active && (
            <ActionButton
              onClick={() => state.enter(w)}
              leftSection={<IconLogin size={16} />}
              size="compact-sm"
              color="accent"
            >
              {t("myOrganization.workshops.enterWorkshop")}
            </ActionButton>
          )}
          <WorkshopActionIcons workshop={w} state={state} />
          <SettingsButton onClick={() => state.openSettings(w)} />
        </Group>
      ),
    },
  };
}

function SettingsButton({ onClick }: { onClick: () => void }) {
  const { t } = useTranslation("common");
  return (
    <Tooltip label={t("myOrganization.workshops.settings")}>
      <GenericIconButton
        icon={<IconSettings size={16} />}
        onClick={onClick}
        aria-label={t("myOrganization.workshops.settings")}
      />
    </Tooltip>
  );
}

/** Invite link, invite by email, (de)activate, delete. */
function WorkshopActionIcons({
  workshop,
  state,
}: {
  workshop: WorkshopView;
  state: WorkshopsTabState;
}) {
  const { t } = useTranslation("common");
  const { invites } = state;
  const hasInvite = invites.hasUsableInvite(workshop);
  const linkLabel = hasInvite
    ? t("myOrganization.workshops.viewInviteLink")
    : t("myOrganization.workshops.createInviteLink");
  const activeLabel = workshop.active
    ? t("myOrganization.workshops.deactivate")
    : t("myOrganization.workshops.activate");

  return (
    <Group gap="xs" wrap="nowrap">
      <Tooltip label={linkLabel}>
        <GenericIconButton
          icon={<IconLink size={16} />}
          color={hasInvite ? "blue" : "gray"}
          onClick={() => invites.openLink(workshop)}
          loading={invites.creatingLink}
          aria-label={linkLabel}
        />
      </Tooltip>
      <Tooltip label={t("myOrganization.workshops.inviteByEmail")}>
        <GenericIconButton
          icon={<IconMail size={16} />}
          onClick={() => invites.openEmail(workshop)}
          aria-label={t("myOrganization.workshops.inviteByEmail")}
        />
      </Tooltip>
      <Tooltip label={activeLabel}>
        <GenericIconButton
          icon={workshop.active ? <IconPlayerPause size={16} /> : <IconPlayerPlay size={16} />}
          color={workshop.active ? "orange" : "green"}
          onClick={() => state.toggleActive(workshop)}
          loading={state.toggling}
          aria-label={activeLabel}
        />
      </Tooltip>
      <Tooltip label={t("delete")}>
        <DeleteIconButton onClick={() => state.requestDelete(workshop)} aria-label={t("delete")} />
      </Tooltip>
    </Group>
  );
}

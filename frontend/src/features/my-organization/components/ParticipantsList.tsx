import { Card, Group, Stack, TextInput, Tooltip } from "@mantine/core";
import {
  IconCalendar,
  IconCheck,
  IconLink,
  IconPencil,
  IconPlayerPlay,
  IconTrash,
  IconUser,
  IconX,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { ParticipantListSort } from "@/api/hooks";
import type { WorkshopParticipantView } from "@/api/types";
import { ListTable, type ListColumn } from "@components/ListTable";
import { DeleteIconButton, GenericIconButton } from "@components/buttons";
import { BodyText, HelperText, Label } from "@components/typography";
import { useWorkshopParticipants } from "../hooks/useWorkshopParticipants";
import { ConfirmationModal } from "./ConfirmationModal";
import { GuestLinkModal } from "./GuestLinkModal";

type ParticipantColumn = ListColumn<WorkshopParticipantView, ParticipantListSort>;

const joinedDate = (p: WorkshopParticipantView) =>
  p.meta?.createdAt ? new Date(p.meta.createdAt).toLocaleDateString() : null;

/**
 * A workshop's participants, embedded in its settings: names edited in place,
 * a Guest's re-login link, removing.
 */
export function ParticipantsList({ workshopId }: { workshopId: string }) {
  const { t } = useTranslation("common");
  const state = useWorkshopParticipants(workshopId);
  const { list, edit } = state;
  const unnamed = t("myOrganization.workshops.unnamedParticipant");

  const nameInput = (
    <TextInput
      size="xs"
      value={edit.name}
      onChange={(e) => edit.setName(e.currentTarget.value)}
      aria-label={t("myOrganization.workshops.participantName")}
      placeholder={t("myOrganization.workshops.participantName")}
      style={{ flex: 1 }}
      autoFocus
      onKeyDown={(e) => {
        if (e.key === "Enter") edit.save();
        if (e.key === "Escape") edit.cancel();
      }}
    />
  );

  const actions = (participant: WorkshopParticipantView) =>
    edit.participantId === participant.id ? (
      <Group gap="xs" wrap="nowrap">
        <Tooltip label={t("save")}>
          <GenericIconButton
            icon={<IconCheck size={14} />}
            color="green"
            size="sm"
            onClick={edit.save}
            loading={edit.saving}
            aria-label={t("save")}
          />
        </Tooltip>
        <Tooltip label={t("cancel")}>
          <GenericIconButton
            icon={<IconX size={14} />}
            color="red"
            size="sm"
            onClick={edit.cancel}
            aria-label={t("cancel")}
          />
        </Tooltip>
      </Group>
    ) : (
      <Group gap="xs" wrap="nowrap">
        {state.hasGuestLink(participant) && (
          <Tooltip label={t("myOrganization.workshops.shareGuestLink")}>
            <GenericIconButton
              icon={<IconLink size={14} />}
              color="blue"
              size="sm"
              onClick={() => state.showGuestLink(participant)}
              aria-label={t("myOrganization.workshops.shareGuestLink")}
            />
          </Tooltip>
        )}
        <Tooltip label={t("myOrganization.workshops.editParticipant")}>
          <GenericIconButton
            icon={<IconPencil size={14} />}
            size="sm"
            onClick={() => edit.start(participant)}
            aria-label={t("myOrganization.workshops.editParticipant")}
          />
        </Tooltip>
        {state.canRemove() && (
          <Tooltip label={t("myOrganization.workshops.removeParticipant")}>
            <DeleteIconButton
              size="sm"
              onClick={() => state.requestRemove(participant)}
              aria-label={t("myOrganization.workshops.removeParticipant")}
            />
          </Tooltip>
        )}
      </Group>
    );

  const columns: ParticipantColumn[] = [
    {
      key: "name",
      header: t("myOrganization.workshops.participantName"),
      sortBy: "name",
      render: (p) =>
        edit.participantId === p.id ? (
          nameInput
        ) : (
          <Group gap="xs" wrap="nowrap">
            <IconUser size={14} color="var(--mantine-color-gray-6)" />
            <BodyText size="sm">{p.name || unnamed}</BodyText>
          </Group>
        ),
    },
    {
      key: "joined",
      header: t("myOrganization.workshops.joined"),
      sortBy: "createdAt",
      render: (p) => <HelperText>{joinedDate(p) || "-"}</HelperText>,
    },
    {
      key: "games",
      header: t("myOrganization.workshops.games"),
      sortBy: "gamesCount",
      render: (p) => <BodyText size="sm">{p.gamesCount || 0}</BodyText>,
    },
    {
      key: "actions",
      header: t("actions"),
      width: 110,
      interactive: true,
      render: actions,
    },
  ];

  const renderCard = (p: WorkshopParticipantView) => {
    const joined = joinedDate(p);
    return (
      <Card padding="xs" radius="sm" withBorder>
        <Group justify="space-between" wrap="nowrap">
          <Group gap="xs" wrap="nowrap" style={{ flex: 1, minWidth: 0 }}>
            <IconUser size={14} color="var(--mantine-color-gray-6)" />
            {edit.participantId === p.id ? (
              nameInput
            ) : (
              <Stack gap={2} style={{ minWidth: 0 }}>
                <BodyText size="sm" fw={500}>
                  {p.name || unnamed}
                </BodyText>
                <Group gap="sm">
                  {joined && (
                    <Group gap={4}>
                      <IconCalendar size={10} color="var(--mantine-color-gray-6)" />
                      <HelperText>
                        {t("myOrganization.workshops.participantJoined", { date: joined })}
                      </HelperText>
                    </Group>
                  )}
                  <Group gap={4}>
                    <IconPlayerPlay size={10} color="var(--mantine-color-gray-6)" />
                    <HelperText>
                      {t("myOrganization.workshops.participantGames", {
                        count: p.gamesCount || 0,
                      })}
                    </HelperText>
                  </Group>
                </Group>
              </Stack>
            )}
          </Group>
          {actions(p)}
        </Group>
      </Card>
    );
  };

  return (
    <>
      <Stack gap="xs">
        <Label>
          {t("myOrganization.workshops.participants")} ({list.total})
        </Label>
        <ListTable
          compact
          list={list}
          columns={columns}
          getRowKey={(p) => p.id}
          renderCard={renderCard}
          empty={{ title: t("myOrganization.workshops.noParticipants") }}
          errorMessage={t("myOrganization.workshops.loadError")}
        />
      </Stack>

      <GuestLinkModal
        guestId={state.guestLinkModal.guest?.id ?? null}
        guestName={state.guestLinkModal.guest?.name}
        onClose={state.guestLinkModal.close}
      />

      <ConfirmationModal
        opened={!!state.removeModal.participant}
        onClose={state.removeModal.close}
        onConfirm={state.removeModal.confirm}
        title={t("myOrganization.workshops.removeParticipantTitle")}
        message={t("myOrganization.workshops.removeParticipantMessage", {
          name: state.removeModal.participant?.name || unnamed,
        })}
        warning={t(
          state.removeModal.participant?.permanent !== false
            ? "myOrganization.workshops.removeParticipantWarningPermanent"
            : "myOrganization.workshops.removeParticipantWarningVisiting",
        )}
        confirmIcon={<IconTrash size={16} />}
        confirmColor="red"
        isLoading={state.removeModal.loading}
      />
    </>
  );
}

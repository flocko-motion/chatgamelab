import { useState } from "react";
import {
  Stack,
  Card,
  Group,
  Badge,
  Loader,
  Center,
  Modal,
  TextInput,
  Switch,
  Alert,
  CopyButton,
  ActionIcon,
  Tooltip,
  Text,
  Collapse,
  Checkbox,
  Select,
  Code,
} from "@mantine/core";
import { useDisclosure, useDebouncedValue } from "@mantine/hooks";
import {
  IconPlus,
  IconTrash,
  IconCopy,
  IconCheck,
  IconAlertCircle,
  IconPlayerPause,
  IconPlayerPlay,
  IconChevronDown,
  IconChevronRight,
  IconUser,
  IconCalendar,
  IconKey,
  IconLink,
  IconClock,
  IconLogin,
  IconPencil,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "@tanstack/react-router";
import { useResponsiveDesign } from "@/common/hooks/useResponsiveDesign";
import { useWorkshopMode } from "@/providers/WorkshopModeProvider";
import { ROUTES } from "@/common/routes/routes";
import {
  useWorkshops,
  useCreateWorkshop,
  useUpdateWorkshop,
  useDeleteWorkshop,
  useCreateWorkshopInvite,
  useRevokeInvite,
  useApiKeys,
  useSetWorkshopApiKey,
  useUpdateParticipant,
  useRemoveParticipant,
} from "@/api/hooks";
import {
  ExpandableSearch,
  SortSelector,
  type SortOption,
} from "@/common/components/controls";
import { ActionButton } from "@/common/components/buttons/ActionButton";
import { TextButton } from "@/common/components/buttons/TextButton";
import { DangerButton } from "@/common/components/buttons/DangerButton";
import { ConfirmationModal } from "./ConfirmationModal";
import type { ObjWorkshop, ObjWorkshopParticipant } from "@/api/generated";

interface WorkshopsTabProps {
  institutionId: string;
}

export function WorkshopsTab({ institutionId }: WorkshopsTabProps) {
  const { t } = useTranslation("common");
  const navigate = useNavigate();
  const { enterWorkshopMode } = useWorkshopMode();
  useResponsiveDesign(); // Keep hook for future responsive needs

  const handleEnterWorkshop = (workshop: ObjWorkshop) => {
    if (!workshop.id || !workshop.name) return;
    enterWorkshopMode(workshop.id, workshop.name);
    navigate({ to: ROUTES.MY_WORKSHOP as "/" });
  };

  const [
    createModalOpened,
    { open: openCreateModal, close: closeCreateModal },
  ] = useDisclosure(false);
  const [
    deleteModalOpened,
    { open: openDeleteModal, close: closeDeleteModal },
  ] = useDisclosure(false);
  const [
    inviteLinkModalOpened,
    { open: openInviteLinkModal, close: closeInviteLinkModal },
  ] = useDisclosure(false);

  const [newWorkshopName, setNewWorkshopName] = useState("");
  const [newWorkshopActive, setNewWorkshopActive] = useState(true);
  const [selectedWorkshop, setSelectedWorkshop] = useState<ObjWorkshop | null>(
    null,
  );
  const [newlyCreatedInvite, setNewlyCreatedInvite] = useState<{
    id?: string;
    inviteToken?: string;
    expiresAt?: string;
    usesCount?: number;
    meta?: { createdAt?: string };
  } | null>(null);
  const [expandedWorkshops, setExpandedWorkshops] = useState<Set<string>>(
    new Set(),
  );

  // Participant editing state
  const [editingParticipant, setEditingParticipant] =
    useState<ObjWorkshopParticipant | null>(null);
  const [participantNewName, setParticipantNewName] = useState("");
  const [participantToRemove, setParticipantToRemove] =
    useState<ObjWorkshopParticipant | null>(null);

  // Search, filter, and sort state
  const [searchQuery, setSearchQuery] = useState("");
  const [debouncedSearch] = useDebouncedValue(searchQuery, 300);
  const [hideInactive, setHideInactive] = useState(false);
  const [sortValue, setSortValue] = useState("createdAt-desc");

  // Parse sort value
  const [sortField, sortDir] = sortValue.split("-") as [
    "name" | "createdAt" | "participantCount",
    "asc" | "desc",
  ];

  const {
    data: workshops,
    isLoading,
    isError,
  } = useWorkshops({
    institutionId,
    search: debouncedSearch || undefined,
    sortBy: sortField,
    sortDir,
    activeOnly: hideInactive || undefined,
  });
  const { data: apiKeys } = useApiKeys();
  const createWorkshop = useCreateWorkshop();
  const updateWorkshop = useUpdateWorkshop();
  const deleteWorkshop = useDeleteWorkshop();
  const createInvite = useCreateWorkshopInvite();
  const revokeInvite = useRevokeInvite();
  const setWorkshopApiKey = useSetWorkshopApiKey();
  const updateParticipant = useUpdateParticipant();
  const removeParticipant = useRemoveParticipant();

  // Build API key options for select
  const apiKeyOptions = [
    { value: "", label: t("myOrganization.workshops.noDefaultApiKey") },
    ...(apiKeys?.map((key) => ({
      value: key.id || "",
      label: key.apiKey?.name || key.apiKey?.platform || "Unknown",
    })) || []),
  ];

  const handleCreateWorkshop = async () => {
    if (!newWorkshopName.trim()) return;

    await createWorkshop.mutateAsync({
      name: newWorkshopName.trim(),
      institutionId,
      active: newWorkshopActive,
      public: false,
    });

    setNewWorkshopName("");
    setNewWorkshopActive(true);
    closeCreateModal();
  };

  const handleToggleActive = async (workshop: ObjWorkshop) => {
    if (!workshop.id) return;
    await updateWorkshop.mutateAsync({
      id: workshop.id,
      name: workshop.name || "",
      active: !workshop.active,
      public: workshop.public,
    });
  };

  const handleViewInviteLink = (workshop: ObjWorkshop) => {
    setNewlyCreatedInvite(null);
    setSelectedWorkshop(workshop);
    openInviteLinkModal();
  };

  const handleCreateAndViewInvite = async (workshop: ObjWorkshop) => {
    if (!workshop.id) return;
    const invite = await createInvite.mutateAsync({ workshopId: workshop.id });
    setNewlyCreatedInvite(invite);
    setSelectedWorkshop(workshop);
    openInviteLinkModal();
  };

  const handleRevokeInviteAndClose = async (inviteId: string) => {
    await revokeInvite.mutateAsync(inviteId);
    setNewlyCreatedInvite(null);
    closeInviteLinkModal();
  };

  const handleSetApiKey = async (
    workshopId: string,
    apiKeyShareId: string | null,
  ) => {
    await setWorkshopApiKey.mutateAsync({ workshopId, apiKeyShareId });
  };

  const handleOpenDeleteModal = (workshop: ObjWorkshop) => {
    setSelectedWorkshop(workshop);
    openDeleteModal();
  };

  const handleDeleteWorkshop = async () => {
    if (!selectedWorkshop?.id) return;

    await deleteWorkshop.mutateAsync(selectedWorkshop.id);
    closeDeleteModal();
    setSelectedWorkshop(null);
  };

  // Participant handlers
  const handleEditParticipant = (participant: ObjWorkshopParticipant) => {
    setEditingParticipant(participant);
    setParticipantNewName(participant.name || "");
  };

  const handleSaveParticipantName = async () => {
    if (!editingParticipant?.id || !participantNewName.trim()) return;
    await updateParticipant.mutateAsync({
      participantId: editingParticipant.id,
      name: participantNewName.trim(),
    });
    setEditingParticipant(null);
    setParticipantNewName("");
  };

  const handleCancelEditParticipant = () => {
    setEditingParticipant(null);
    setParticipantNewName("");
  };

  const handleConfirmRemoveParticipant = async () => {
    if (!participantToRemove?.id) return;
    await removeParticipant.mutateAsync(participantToRemove.id);
    setParticipantToRemove(null);
  };

  // Workshop settings handler
  const handleUpdateWorkshopSettings = async (
    workshop: ObjWorkshop,
    settings: Partial<{
      showAiModelSelector: boolean;
      showPublicGames: boolean;
      showOtherParticipantsGames: boolean;
      useSpecificAiModel: string | null;
    }>,
  ) => {
    if (!workshop.id) return;
    await updateWorkshop.mutateAsync({
      id: workshop.id,
      name: workshop.name || "",
      active: workshop.active || false,
      public: workshop.public || false,
      showAiModelSelector:
        settings.showAiModelSelector ?? workshop.showAiModelSelector ?? false,
      showPublicGames:
        settings.showPublicGames ?? workshop.showPublicGames ?? false,
      showOtherParticipantsGames:
        settings.showOtherParticipantsGames ??
        workshop.showOtherParticipantsGames ??
        true,
      useSpecificAiModel:
        settings.useSpecificAiModel ?? workshop.useSpecificAiModel ?? undefined,
    });
  };

  if (isLoading) {
    return (
      <Center py="xl">
        <Loader />
      </Center>
    );
  }

  if (isError) {
    return (
      <Alert
        color="red"
        icon={<IconAlertCircle size={16} />}
        title={t("error")}
      >
        {t("myOrganization.workshops.loadError")}
      </Alert>
    );
  }

  const sortOptions: SortOption[] = [
    {
      value: "createdAt-desc",
      label: t("myOrganization.workshops.sort.createdAtDesc"),
    },
    {
      value: "createdAt-asc",
      label: t("myOrganization.workshops.sort.createdAtAsc"),
    },
    { value: "name-asc", label: t("myOrganization.workshops.sort.nameAsc") },
    { value: "name-desc", label: t("myOrganization.workshops.sort.nameDesc") },
    {
      value: "participantCount-desc",
      label: t("myOrganization.workshops.sort.participantsDesc"),
    },
    {
      value: "participantCount-asc",
      label: t("myOrganization.workshops.sort.participantsAsc"),
    },
  ];

  return (
    <>
      <Stack gap="lg">
        {/* Subtitle */}
        <Text size="sm" c="dimmed">
          {t("myOrganization.workshops.subtitle")}
        </Text>

        {/* Controls row: Create button on left, Search, Hide inactive, Sort on right */}
        <Group justify="space-between" wrap="wrap" gap="sm">
          <ActionButton
            onClick={openCreateModal}
            leftSection={<IconPlus size={16} />}
            size="sm"
          >
            {t("myOrganization.workshops.create")}
          </ActionButton>
          <Group gap="sm">
            <ExpandableSearch
              value={searchQuery}
              onChange={setSearchQuery}
              placeholder={t("search")}
            />
            <Checkbox
              label={t("myOrganization.workshops.hideInactive")}
              checked={hideInactive}
              onChange={(e) => setHideInactive(e.currentTarget.checked)}
              size="sm"
            />
            <SortSelector
              options={sortOptions}
              value={sortValue}
              onChange={setSortValue}
              label={t("myOrganization.workshops.sort.label")}
            />
          </Group>
        </Group>

        {/* Workshops list */}
        {workshops && workshops.length > 0 ? (
          <Stack gap="md">
            {workshops.map((workshop) => {
              const isExpanded = expandedWorkshops.has(workshop.id || "");
              const toggleExpand = () => {
                const newSet = new Set(expandedWorkshops);
                if (isExpanded) {
                  newSet.delete(workshop.id || "");
                } else {
                  newSet.add(workshop.id || "");
                }
                setExpandedWorkshops(newSet);
              };
              const createdDate = workshop.meta?.createdAt
                ? new Date(workshop.meta.createdAt).toLocaleDateString()
                : "";

              return (
                <Card
                  key={workshop.id}
                  shadow="sm"
                  padding="md"
                  radius="md"
                  withBorder
                >
                  {/* Workshop header - always visible */}
                  <Group justify="space-between" wrap="nowrap" gap="sm">
                    <Group gap="sm" style={{ flex: 1, minWidth: 0 }}>
                      <ActionIcon
                        variant="subtle"
                        onClick={toggleExpand}
                        size="sm"
                      >
                        {isExpanded ? (
                          <IconChevronDown size={16} />
                        ) : (
                          <IconChevronRight size={16} />
                        )}
                      </ActionIcon>
                      <Stack gap={2} style={{ minWidth: 0 }}>
                        <Text size="sm" fw={500} truncate>
                          {workshop.name}
                        </Text>
                        <Group gap="xs">
                          <Badge
                            color={workshop.active ? "green" : "gray"}
                            variant="light"
                            size="xs"
                          >
                            {workshop.active
                              ? t("myOrganization.workshops.active")
                              : t("myOrganization.workshops.inactive")}
                          </Badge>
                          <Group gap={4}>
                            <IconUser size={12} color="gray" />
                            <Text size="xs" c="dimmed">
                              {workshop.participants?.length || 0}
                            </Text>
                          </Group>
                          {createdDate && (
                            <Group gap={4}>
                              <IconCalendar size={12} color="gray" />
                              <Text size="xs" c="dimmed">
                                {createdDate}
                              </Text>
                            </Group>
                          )}
                        </Group>
                      </Stack>
                    </Group>
                    <Group gap="xs" wrap="nowrap">
                      {/* Enter Workshop Mode button - only for active workshops */}
                      {workshop.active && (
                        <Tooltip
                          label={t("myOrganization.workshops.enterWorkshop")}
                        >
                          <ActionIcon
                            variant="subtle"
                            color="violet"
                            onClick={() => handleEnterWorkshop(workshop)}
                          >
                            <IconLogin size={16} />
                          </ActionIcon>
                        </Tooltip>
                      )}
                      {(() => {
                        const existingInvite = workshop.invites?.find(
                          (inv) => inv.status === "pending" && inv.inviteToken,
                        );
                        return (
                          <Tooltip
                            label={
                              existingInvite
                                ? t("myOrganization.workshops.viewInviteLink")
                                : t("myOrganization.workshops.createInviteLink")
                            }
                          >
                            <ActionIcon
                              variant="subtle"
                              color={existingInvite ? "blue" : "gray"}
                              onClick={() =>
                                existingInvite
                                  ? handleViewInviteLink(workshop)
                                  : handleCreateAndViewInvite(workshop)
                              }
                              loading={createInvite.isPending}
                            >
                              <IconLink size={16} />
                            </ActionIcon>
                          </Tooltip>
                        );
                      })()}
                      <Tooltip
                        label={
                          workshop.active
                            ? t("myOrganization.workshops.deactivate")
                            : t("myOrganization.workshops.activate")
                        }
                      >
                        <ActionIcon
                          variant="subtle"
                          color={workshop.active ? "orange" : "green"}
                          onClick={() => handleToggleActive(workshop)}
                          loading={updateWorkshop.isPending}
                        >
                          {workshop.active ? (
                            <IconPlayerPause size={16} />
                          ) : (
                            <IconPlayerPlay size={16} />
                          )}
                        </ActionIcon>
                      </Tooltip>
                      <Tooltip label={t("delete")}>
                        <ActionIcon
                          variant="subtle"
                          color="red"
                          onClick={() => handleOpenDeleteModal(workshop)}
                        >
                          <IconTrash size={16} />
                        </ActionIcon>
                      </Tooltip>
                    </Group>
                  </Group>

                  {/* Expandable settings section */}
                  <Collapse in={isExpanded}>
                    <Stack
                      gap="md"
                      mt="md"
                      pt="md"
                      style={{
                        borderTop: "1px solid var(--mantine-color-gray-3)",
                      }}
                    >
                      {/* Default API Key Section */}
                      <Stack gap="xs">
                        <Text size="sm" fw={500}>
                          <Group gap="xs">
                            <IconKey size={14} />
                            {t("myOrganization.workshops.defaultApiKey")}
                          </Group>
                        </Text>
                        <Select
                          size="xs"
                          data={apiKeyOptions}
                          value={workshop.defaultApiKeyShareId || ""}
                          onChange={(value) =>
                            handleSetApiKey(workshop.id!, value || null)
                          }
                          placeholder={t(
                            "myOrganization.workshops.selectApiKey",
                          )}
                          clearable
                          disabled={setWorkshopApiKey.isPending}
                        />
                        <Text size="xs" c="dimmed">
                          {t("myOrganization.workshops.defaultApiKeyHint")}
                        </Text>
                      </Stack>

                      {/* Workshop Settings Section */}
                      <Stack gap="xs">
                        <Text size="sm" fw={500}>
                          {t("myOrganization.workshops.settings")}
                        </Text>
                        <Switch
                          size="xs"
                          label={t(
                            "myOrganization.workshops.showAiModelSelector",
                          )}
                          checked={workshop.showAiModelSelector || false}
                          onChange={(e) =>
                            handleUpdateWorkshopSettings(workshop, {
                              showAiModelSelector: e.currentTarget.checked,
                            })
                          }
                        />
                        <Switch
                          size="xs"
                          label={t("myOrganization.workshops.showPublicGames")}
                          checked={workshop.showPublicGames || false}
                          onChange={(e) =>
                            handleUpdateWorkshopSettings(workshop, {
                              showPublicGames: e.currentTarget.checked,
                            })
                          }
                        />
                        <Switch
                          size="xs"
                          label={t(
                            "myOrganization.workshops.showOtherParticipantsGames",
                          )}
                          checked={
                            workshop.showOtherParticipantsGames !== false
                          }
                          onChange={(e) =>
                            handleUpdateWorkshopSettings(workshop, {
                              showOtherParticipantsGames:
                                e.currentTarget.checked,
                            })
                          }
                        />
                      </Stack>

                      {/* Participants Section */}
                      <Stack gap="xs">
                        <Text size="sm" fw={500}>
                          {t("myOrganization.workshops.participants")} (
                          {workshop.participants?.length || 0})
                        </Text>
                        {workshop.participants &&
                        workshop.participants.length > 0 ? (
                          <Stack gap="sm">
                            {workshop.participants.map((participant) => {
                              const joinedDate = participant.meta?.createdAt
                                ? new Date(
                                    participant.meta.createdAt,
                                  ).toLocaleDateString()
                                : null;
                              const isEditing =
                                editingParticipant?.id === participant.id;

                              return (
                                <Card
                                  key={participant.id}
                                  padding="xs"
                                  radius="sm"
                                  withBorder
                                >
                                  <Group justify="space-between" wrap="nowrap">
                                    <Group
                                      gap="xs"
                                      style={{ flex: 1, minWidth: 0 }}
                                    >
                                      <IconUser size={14} color="gray" />
                                      {isEditing ? (
                                        <TextInput
                                          size="xs"
                                          value={participantNewName}
                                          onChange={(e) =>
                                            setParticipantNewName(
                                              e.currentTarget.value,
                                            )
                                          }
                                          placeholder={t(
                                            "myOrganization.workshops.participantName",
                                          )}
                                          style={{ flex: 1 }}
                                          autoFocus
                                          onKeyDown={(e) => {
                                            if (e.key === "Enter")
                                              handleSaveParticipantName();
                                            if (e.key === "Escape")
                                              handleCancelEditParticipant();
                                          }}
                                        />
                                      ) : (
                                        <Stack gap={2} style={{ minWidth: 0 }}>
                                          <Text size="sm" fw={500} truncate>
                                            {participant.name ||
                                              t(
                                                "myOrganization.workshops.anonymousParticipant",
                                              )}
                                          </Text>
                                          <Group gap="sm">
                                            {joinedDate && (
                                              <Group gap={4}>
                                                <IconCalendar
                                                  size={10}
                                                  color="gray"
                                                />
                                                <Text size="xs" c="dimmed">
                                                  {t(
                                                    "myOrganization.workshops.participantJoined",
                                                    { date: joinedDate },
                                                  )}
                                                </Text>
                                              </Group>
                                            )}
                                            <Group gap={4}>
                                              <IconPlayerPlay
                                                size={10}
                                                color="gray"
                                              />
                                              <Text size="xs" c="dimmed">
                                                {t(
                                                  "myOrganization.workshops.participantGames",
                                                  {
                                                    count:
                                                      participant.gamesCount ||
                                                      0,
                                                  },
                                                )}
                                              </Text>
                                            </Group>
                                          </Group>
                                        </Stack>
                                      )}
                                    </Group>
                                    <Group gap="xs" wrap="nowrap">
                                      {isEditing ? (
                                        <>
                                          <Tooltip label={t("save")}>
                                            <ActionIcon
                                              variant="subtle"
                                              color="green"
                                              size="sm"
                                              onClick={
                                                handleSaveParticipantName
                                              }
                                              loading={
                                                updateParticipant.isPending
                                              }
                                            >
                                              <IconCheck size={14} />
                                            </ActionIcon>
                                          </Tooltip>
                                          <Tooltip label={t("cancel")}>
                                            <ActionIcon
                                              variant="subtle"
                                              color="gray"
                                              size="sm"
                                              onClick={
                                                handleCancelEditParticipant
                                              }
                                            >
                                              <IconAlertCircle size={14} />
                                            </ActionIcon>
                                          </Tooltip>
                                        </>
                                      ) : (
                                        <>
                                          <Tooltip
                                            label={t(
                                              "myOrganization.workshops.editParticipant",
                                            )}
                                          >
                                            <ActionIcon
                                              variant="subtle"
                                              color="gray"
                                              size="sm"
                                              onClick={() =>
                                                handleEditParticipant(
                                                  participant,
                                                )
                                              }
                                            >
                                              <IconPencil size={14} />
                                            </ActionIcon>
                                          </Tooltip>
                                          <Tooltip
                                            label={t(
                                              "myOrganization.workshops.removeParticipant",
                                            )}
                                          >
                                            <ActionIcon
                                              variant="subtle"
                                              color="red"
                                              size="sm"
                                              onClick={() =>
                                                setParticipantToRemove(
                                                  participant,
                                                )
                                              }
                                            >
                                              <IconTrash size={14} />
                                            </ActionIcon>
                                          </Tooltip>
                                        </>
                                      )}
                                    </Group>
                                  </Group>
                                </Card>
                              );
                            })}
                          </Stack>
                        ) : (
                          <Text size="sm" c="dimmed">
                            {t("myOrganization.workshops.noParticipants")}
                          </Text>
                        )}
                      </Stack>
                    </Stack>
                  </Collapse>
                </Card>
              );
            })}
          </Stack>
        ) : (
          <Card shadow="sm" padding="xl" radius="md" withBorder>
            <Stack align="center" gap="md">
              <Text c="dimmed" ta="center">
                {t("myOrganization.workshops.empty")}
              </Text>
              <ActionButton
                onClick={openCreateModal}
                leftSection={<IconPlus size={16} />}
              >
                {t("myOrganization.workshops.createFirst")}
              </ActionButton>
            </Stack>
          </Card>
        )}
      </Stack>

      {/* Create Workshop Modal */}
      <Modal
        opened={createModalOpened}
        onClose={closeCreateModal}
        title={t("myOrganization.workshops.createTitle")}
        size="md"
      >
        <Stack gap="md">
          <TextInput
            label={t("myOrganization.workshops.nameLabel")}
            placeholder={t("myOrganization.workshops.namePlaceholder")}
            value={newWorkshopName}
            onChange={(e) => setNewWorkshopName(e.currentTarget.value)}
            required
          />
          <Switch
            label={t("myOrganization.workshops.activeLabel")}
            description={t("myOrganization.workshops.activeDescription")}
            checked={newWorkshopActive}
            onChange={(e) => setNewWorkshopActive(e.currentTarget.checked)}
          />
          <Group justify="flex-end" mt="md">
            <TextButton onClick={closeCreateModal}>{t("cancel")}</TextButton>
            <ActionButton
              onClick={handleCreateWorkshop}
              loading={createWorkshop.isPending}
              disabled={!newWorkshopName.trim()}
            >
              {t("myOrganization.workshops.create")}
            </ActionButton>
          </Group>
        </Stack>
      </Modal>

      {/* Delete Confirmation Modal */}
      <ConfirmationModal
        opened={deleteModalOpened}
        onClose={closeDeleteModal}
        title={t("myOrganization.workshops.deleteTitle")}
        message={t("myOrganization.workshops.deleteConfirm", {
          name: selectedWorkshop?.name,
        })}
        warning={t("myOrganization.workshops.deleteWarning")}
        warningColor="red"
        confirmIcon={<IconTrash size={16} />}
        confirmColor="red"
        onConfirm={handleDeleteWorkshop}
        isLoading={deleteWorkshop.isPending}
      />

      {/* View Invite Link Modal */}
      <Modal
        opened={inviteLinkModalOpened}
        onClose={closeInviteLinkModal}
        title={t("myOrganization.workshops.inviteLinkTitle", {
          name: selectedWorkshop?.name,
        })}
        size="md"
      >
        {(() => {
          // Use newly created invite if available, otherwise look for existing one
          const existingInvite =
            newlyCreatedInvite ||
            selectedWorkshop?.invites?.find(
              (inv) => inv.status === "pending" && inv.inviteToken,
            );
          if (!existingInvite?.inviteToken) {
            return (
              <Text c="dimmed">
                {t("myOrganization.workshops.noActiveInvite")}
              </Text>
            );
          }
          const inviteLink = `${window.location.origin}/invites/${existingInvite.inviteToken}/accept`;
          const createdAt = existingInvite.meta?.createdAt
            ? new Date(existingInvite.meta.createdAt)
            : null;
          const expiresAt = existingInvite.expiresAt
            ? new Date(existingInvite.expiresAt)
            : null;

          return (
            <Stack gap="md">
              <Text size="sm" c="dimmed">
                {t("myOrganization.workshops.inviteDescription")}
              </Text>

              <Stack gap="xs">
                <Text size="sm" fw={500}>
                  {t("myOrganization.workshops.inviteLink")}
                </Text>
                <Group gap="xs">
                  <Code
                    style={{
                      flex: 1,
                      padding: "8px 12px",
                      wordBreak: "break-all",
                    }}
                  >
                    {inviteLink}
                  </Code>
                  <CopyButton value={inviteLink}>
                    {({ copied, copy }) => (
                      <Tooltip label={copied ? t("copied") : t("copy")}>
                        <ActionIcon
                          color={copied ? "teal" : "gray"}
                          onClick={copy}
                          size="lg"
                        >
                          {copied ? (
                            <IconCheck size={18} />
                          ) : (
                            <IconCopy size={18} />
                          )}
                        </ActionIcon>
                      </Tooltip>
                    )}
                  </CopyButton>
                </Group>
              </Stack>

              <Group gap="xl">
                {createdAt && (
                  <Stack gap={2}>
                    <Text size="xs" c="dimmed">
                      {t("myOrganization.workshops.inviteCreatedAt")}
                    </Text>
                    <Group gap="xs">
                      <IconCalendar size={14} />
                      <Text size="sm">{createdAt.toLocaleDateString()}</Text>
                    </Group>
                  </Stack>
                )}
                {expiresAt && (
                  <Stack gap={2}>
                    <Text size="xs" c="dimmed">
                      {t("myOrganization.workshops.inviteExpiresAt")}
                    </Text>
                    <Group gap="xs">
                      <IconClock size={14} />
                      <Text size="sm">{expiresAt.toLocaleDateString()}</Text>
                    </Group>
                  </Stack>
                )}
                {existingInvite.usesCount !== undefined &&
                  existingInvite.usesCount > 0 && (
                    <Stack gap={2}>
                      <Text size="xs" c="dimmed">
                        {t("myOrganization.workshops.inviteUsage")}
                      </Text>
                      <Badge size="sm" variant="light">
                        {t("myOrganization.workshops.usedCount", {
                          count: existingInvite.usesCount,
                        })}
                      </Badge>
                    </Stack>
                  )}
              </Group>

              <Group justify="space-between" mt="md">
                <DangerButton
                  onClick={() => {
                    const invite =
                      newlyCreatedInvite ||
                      selectedWorkshop?.invites?.find(
                        (inv) => inv.status === "pending" && inv.inviteToken,
                      );
                    if (invite?.id) handleRevokeInviteAndClose(invite.id);
                  }}
                  loading={revokeInvite.isPending}
                >
                  {t("myOrganization.workshops.revokeInvite")}
                </DangerButton>
                <TextButton onClick={closeInviteLinkModal}>
                  {t("close")}
                </TextButton>
              </Group>
            </Stack>
          );
        })()}
      </Modal>

      {/* Remove Participant Confirmation Modal */}
      <ConfirmationModal
        opened={!!participantToRemove}
        onClose={() => setParticipantToRemove(null)}
        onConfirm={handleConfirmRemoveParticipant}
        title={t("myOrganization.workshops.removeParticipantTitle")}
        message={t("myOrganization.workshops.removeParticipantConfirm", {
          name:
            participantToRemove?.name ||
            t("myOrganization.workshops.anonymousParticipant"),
        })}
        confirmIcon={<IconTrash size={16} />}
        confirmColor="red"
        isLoading={removeParticipant.isPending}
      />
    </>
  );
}

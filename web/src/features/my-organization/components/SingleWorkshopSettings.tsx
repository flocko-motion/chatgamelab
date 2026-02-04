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
  Code,
  Table,
  Select,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { notifications } from "@mantine/notifications";
import {
  IconTrash,
  IconCopy,
  IconCheck,
  IconAlertCircle,
  IconUser,
  IconCalendar,
  IconKey,
  IconLink,
  IconClock,
  IconPlayerPlay,
  IconPencil,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useResponsiveDesign } from "@/common/hooks/useResponsiveDesign";
import {
  useWorkshop,
  useUpdateWorkshop,
  useCreateWorkshopInvite,
  useRevokeInvite,
  useInstitutionApiKeys,
  useSetWorkshopApiKey,
  useUpdateParticipant,
  useRemoveParticipant,
  useGetParticipantToken,
  usePlatforms,
} from "@/api/hooks";
import { TextButton } from "@/common/components/buttons/TextButton";
import { buildShareUrl } from "@/common/lib/url";
import { DangerButton } from "@/common/components/buttons/DangerButton";
import { ConfirmationModal } from "./ConfirmationModal";
import { ObjRole, type ObjWorkshopParticipant } from "@/api/generated";

interface SingleWorkshopSettingsProps {
  workshopId: string;
  institutionId: string;
}

export function SingleWorkshopSettings({
  workshopId,
  institutionId,
}: SingleWorkshopSettingsProps) {
  const { t } = useTranslation("common");
  const { isMobile } = useResponsiveDesign();

  const [
    inviteLinkModalOpened,
    { open: openInviteLinkModal, close: closeInviteLinkModal },
  ] = useDisclosure(false);

  const [newlyCreatedInvite, setNewlyCreatedInvite] = useState<{
    id?: string;
    inviteToken?: string;
    expiresAt?: string;
    usesCount?: number;
    meta?: { createdAt?: string };
  } | null>(null);

  // Participant editing state
  const [editingParticipant, setEditingParticipant] =
    useState<ObjWorkshopParticipant | null>(null);
  const [participantNewName, setParticipantNewName] = useState("");
  const [participantToRemove, setParticipantToRemove] =
    useState<ObjWorkshopParticipant | null>(null);

  const { data: workshop, isLoading, isError } = useWorkshop(workshopId);

  const { data: institutionApiKeys } = useInstitutionApiKeys(institutionId);
  const updateWorkshop = useUpdateWorkshop();
  const createInvite = useCreateWorkshopInvite();
  const revokeInvite = useRevokeInvite();
  const setWorkshopApiKey = useSetWorkshopApiKey();
  const updateParticipant = useUpdateParticipant();
  const removeParticipant = useRemoveParticipant();
  const getParticipantToken = useGetParticipantToken();

  const { data: platforms } = usePlatforms();

  // Build API key options for select - only institution-shared keys
  const apiKeyOptions = [
    { value: "", label: t("myOrganization.workshops.noDefaultApiKey") },
    ...(institutionApiKeys?.map((key) => ({
      value: key.id || "",
      label: key.apiKey?.name || key.apiKey?.platform || "Unknown",
    })) || []),
  ];

  // Get platform for the selected workshop API key to show available models
  const selectedApiKey = institutionApiKeys?.find(
    (k) => k.id === workshop?.defaultApiKeyShareId,
  );
  const selectedPlatform = platforms?.find(
    (p) => p.id === selectedApiKey?.apiKey?.platform,
  );
  const modelOptions = [
    { value: "", label: t("myOrganization.workshops.noDefaultModel") },
    ...(selectedPlatform?.models?.map((model) => ({
      value: model.id || "",
      label: model.name || model.id || "",
    })) || []),
  ];

  const handleCreateAndViewInvite = async () => {
    if (!workshopId) return;
    const invite = await createInvite.mutateAsync({ workshopId });
    setNewlyCreatedInvite(invite);
    openInviteLinkModal();
  };

  const handleViewInviteLink = () => {
    setNewlyCreatedInvite(null);
    openInviteLinkModal();
  };

  const handleRevokeInviteAndClose = async (inviteId: string) => {
    await revokeInvite.mutateAsync(inviteId);
    setNewlyCreatedInvite(null);
    closeInviteLinkModal();
  };

  const handleSetApiKey = async (apiKeyShareId: string | null) => {
    await setWorkshopApiKey.mutateAsync({ workshopId, apiKeyShareId });
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

  const handleGetParticipantShareLink = async (participantId: string) => {
    try {
      const result = await getParticipantToken.mutateAsync(participantId);
      if (result?.token) {
        const shareUrl = buildShareUrl(`/invites/participant/${result.token}`);
        // Copy to clipboard directly
        await navigator.clipboard.writeText(shareUrl);
        notifications.show({
          title: t("myOrganization.workshops.linkCopied"),
          message: t("myOrganization.workshops.linkCopiedMessage"),
          color: "green",
        });
      }
    } catch {
      // Participant doesn't have a token (not an anonymous participant)
      notifications.show({
        title: t("error"),
        message: t("myOrganization.workshops.noParticipantToken"),
        color: "red",
      });
    }
  };

  // Workshop settings handler
  const handleUpdateWorkshopSettings = async (
    settings: Partial<{
      showAiModelSelector: boolean;
      showPublicGames: boolean;
      showOtherParticipantsGames: boolean;
      useSpecificAiModel: string | null;
    }>,
  ) => {
    if (!workshop?.id) return;
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

  if (isError || !workshop) {
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

  const existingInvite = workshop.invites?.find(
    (inv) => inv.status === "pending" && inv.inviteToken,
  );
  const inviteLink = existingInvite?.inviteToken
    ? buildShareUrl(`/invites/${existingInvite.inviteToken}/accept`)
    : newlyCreatedInvite?.inviteToken
      ? buildShareUrl(`/invites/${newlyCreatedInvite.inviteToken}/accept`)
      : null;

  return (
    <>
      <Card shadow="sm" padding="md" radius="md" withBorder>
        {/* Workshop header info */}
        <Group justify="space-between" wrap="nowrap" gap="sm" mb="md">
          <Group gap="sm" style={{ flex: 1, minWidth: 0 }}>
            <Stack gap={2} style={{ minWidth: 0 }}>
              <Text size="lg" fw={600} truncate>
                {workshop.name}
              </Text>
              <Group gap="xs">
                <Badge
                  color={workshop.active ? "green" : "gray"}
                  variant="light"
                  size="sm"
                >
                  {workshop.active
                    ? t("myOrganization.workshops.active")
                    : t("myOrganization.workshops.inactive")}
                </Badge>
                <Group gap={4}>
                  <IconUser size={14} color="gray" />
                  <Text size="sm" c="dimmed">
                    {workshop.participants?.length || 0}{" "}
                    {t("myOrganization.workshops.participants").toLowerCase()}
                  </Text>
                </Group>
              </Group>
            </Stack>
          </Group>
          <Group gap="xs" wrap="nowrap">
            {/* Invite Link button */}
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
                    ? handleViewInviteLink()
                    : handleCreateAndViewInvite()
                }
                loading={createInvite.isPending}
              >
                <IconLink size={18} />
              </ActionIcon>
            </Tooltip>
          </Group>
        </Group>

        {/* Settings content - always visible, not collapsible */}
        <Stack
          gap="md"
          pt="md"
          style={{
            borderTop: "1px solid var(--mantine-color-gray-3)",
          }}
        >
          {/* Default API Key Section */}
          <Stack gap="xs">
            <Group gap="xs">
              <IconKey size={14} />
              <Text size="sm" fw={500}>
                {t("myOrganization.workshops.defaultApiKey")}
              </Text>
            </Group>
            <Select
              size="sm"
              data={apiKeyOptions}
              value={workshop.defaultApiKeyShareId || ""}
              onChange={(value: string | null) =>
                handleSetApiKey(value || null)
              }
              placeholder={t("myOrganization.workshops.selectApiKey")}
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
              size="sm"
              label={t("myOrganization.workshops.showAiModelSelector")}
              checked={workshop.showAiModelSelector || false}
              onChange={(e) =>
                handleUpdateWorkshopSettings({
                  showAiModelSelector: e.currentTarget.checked,
                })
              }
            />
            {/* Default AI Model - only show when API key is selected and model selector is disabled */}
            {workshop.defaultApiKeyShareId && !workshop.showAiModelSelector && (
              <Select
                size="sm"
                label={t("myOrganization.workshops.defaultAiModel")}
                description={t("myOrganization.workshops.defaultAiModelHint")}
                data={modelOptions}
                value={workshop.useSpecificAiModel || ""}
                onChange={(value) =>
                  handleUpdateWorkshopSettings({
                    useSpecificAiModel: value || null,
                  })
                }
                disabled={updateWorkshop.isPending}
              />
            )}
            <Switch
              size="sm"
              label={t("myOrganization.workshops.showPublicGames")}
              checked={workshop.showPublicGames || false}
              onChange={(e) =>
                handleUpdateWorkshopSettings({
                  showPublicGames: e.currentTarget.checked,
                })
              }
            />
            <Switch
              size="sm"
              label={t("myOrganization.workshops.showOtherParticipantsGames")}
              checked={workshop.showOtherParticipantsGames !== false}
              onChange={(e) =>
                handleUpdateWorkshopSettings({
                  showOtherParticipantsGames: e.currentTarget.checked,
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
            {workshop.participants && workshop.participants.length > 0 ? (
              isMobile ? (
                <Stack gap="sm">
                  {workshop.participants.map((participant) => {
                    const joinedDate = participant.meta?.createdAt
                      ? new Date(
                          participant.meta.createdAt,
                        ).toLocaleDateString()
                      : null;
                    const isEditing = editingParticipant?.id === participant.id;

                    return (
                      <Card
                        key={participant.id}
                        padding="xs"
                        radius="sm"
                        withBorder
                      >
                        <Group justify="space-between" wrap="nowrap">
                          <Group gap="xs" style={{ flex: 1, minWidth: 0 }}>
                            <IconUser size={14} color="gray" />
                            {isEditing ? (
                              <TextInput
                                size="xs"
                                value={participantNewName}
                                onChange={(e) =>
                                  setParticipantNewName(e.currentTarget.value)
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
                                      <IconCalendar size={10} color="gray" />
                                      <Text size="xs" c="dimmed">
                                        {t(
                                          "myOrganization.workshops.participantJoined",
                                          {
                                            date: joinedDate,
                                          },
                                        )}
                                      </Text>
                                    </Group>
                                  )}
                                  <Group gap={4}>
                                    <IconPlayerPlay size={10} color="gray" />
                                    <Text size="xs" c="dimmed">
                                      {t(
                                        "myOrganization.workshops.participantGames",
                                        {
                                          count: participant.gamesCount || 0,
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
                                    onClick={handleSaveParticipantName}
                                    loading={updateParticipant.isPending}
                                  >
                                    <IconCheck size={14} />
                                  </ActionIcon>
                                </Tooltip>
                                <Tooltip label={t("cancel")}>
                                  <ActionIcon
                                    variant="subtle"
                                    color="gray"
                                    size="sm"
                                    onClick={handleCancelEditParticipant}
                                  >
                                    <IconAlertCircle size={14} />
                                  </ActionIcon>
                                </Tooltip>
                              </>
                            ) : (
                              <>
                                {participant.role ===
                                  ObjRole.RoleParticipant && (
                                  <Tooltip
                                    label={t(
                                      "myOrganization.workshops.shareParticipantLink",
                                    )}
                                  >
                                    <ActionIcon
                                      variant="subtle"
                                      color="blue"
                                      size="sm"
                                      loading={getParticipantToken.isPending}
                                      onClick={(e) => {
                                        e.stopPropagation();
                                        if (participant.id) {
                                          handleGetParticipantShareLink(
                                            participant.id,
                                          );
                                        }
                                      }}
                                    >
                                      <IconLink size={14} />
                                    </ActionIcon>
                                  </Tooltip>
                                )}
                                <Tooltip
                                  label={t(
                                    "myOrganization.workshops.editParticipant",
                                  )}
                                >
                                  <ActionIcon
                                    variant="subtle"
                                    color="gray"
                                    size="sm"
                                    onClick={(e) => {
                                      e.stopPropagation();
                                      handleEditParticipant(participant);
                                    }}
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
                                      setParticipantToRemove(participant)
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
                <Table striped highlightOnHover>
                  <Table.Thead>
                    <Table.Tr>
                      <Table.Th>
                        {t("myOrganization.workshops.participantName")}
                      </Table.Th>
                      <Table.Th>
                        {t("myOrganization.workshops.joined")}
                      </Table.Th>
                      <Table.Th>{t("myOrganization.workshops.games")}</Table.Th>
                      <Table.Th style={{ width: 100 }}>{t("actions")}</Table.Th>
                    </Table.Tr>
                  </Table.Thead>
                  <Table.Tbody>
                    {workshop.participants.map((participant) => {
                      const joinedDate = participant.meta?.createdAt
                        ? new Date(
                            participant.meta.createdAt,
                          ).toLocaleDateString()
                        : null;
                      const isEditing =
                        editingParticipant?.id === participant.id;

                      return (
                        <Table.Tr key={participant.id}>
                          <Table.Td>
                            {isEditing ? (
                              <TextInput
                                size="xs"
                                value={participantNewName}
                                onChange={(e) =>
                                  setParticipantNewName(e.currentTarget.value)
                                }
                                placeholder={t(
                                  "myOrganization.workshops.participantName",
                                )}
                                autoFocus
                                onKeyDown={(e) => {
                                  if (e.key === "Enter")
                                    handleSaveParticipantName();
                                  if (e.key === "Escape")
                                    handleCancelEditParticipant();
                                }}
                              />
                            ) : (
                              <Group gap="xs">
                                <IconUser size={14} color="gray" />
                                <Text size="sm">
                                  {participant.name ||
                                    t(
                                      "myOrganization.workshops.anonymousParticipant",
                                    )}
                                </Text>
                              </Group>
                            )}
                          </Table.Td>
                          <Table.Td>
                            <Text size="sm" c="dimmed">
                              {joinedDate || "-"}
                            </Text>
                          </Table.Td>
                          <Table.Td>
                            <Text size="sm">{participant.gamesCount || 0}</Text>
                          </Table.Td>
                          <Table.Td>
                            <Group gap="xs" wrap="nowrap">
                              {isEditing ? (
                                <>
                                  <Tooltip label={t("save")}>
                                    <ActionIcon
                                      variant="subtle"
                                      color="green"
                                      size="sm"
                                      onClick={handleSaveParticipantName}
                                      loading={updateParticipant.isPending}
                                    >
                                      <IconCheck size={14} />
                                    </ActionIcon>
                                  </Tooltip>
                                  <Tooltip label={t("cancel")}>
                                    <ActionIcon
                                      variant="subtle"
                                      color="gray"
                                      size="sm"
                                      onClick={handleCancelEditParticipant}
                                    >
                                      <IconAlertCircle size={14} />
                                    </ActionIcon>
                                  </Tooltip>
                                </>
                              ) : (
                                <>
                                  {participant.role ===
                                    ObjRole.RoleParticipant && (
                                    <Tooltip
                                      label={t(
                                        "myOrganization.workshops.shareParticipantLink",
                                      )}
                                    >
                                      <ActionIcon
                                        variant="subtle"
                                        color="blue"
                                        size="sm"
                                        loading={getParticipantToken.isPending}
                                        onClick={(e) => {
                                          e.stopPropagation();
                                          if (participant.id) {
                                            handleGetParticipantShareLink(
                                              participant.id,
                                            );
                                          }
                                        }}
                                      >
                                        <IconLink size={14} />
                                      </ActionIcon>
                                    </Tooltip>
                                  )}
                                  <Tooltip
                                    label={t(
                                      "myOrganization.workshops.editParticipant",
                                    )}
                                  >
                                    <ActionIcon
                                      variant="subtle"
                                      color="gray"
                                      size="sm"
                                      onClick={(e) => {
                                        e.stopPropagation();
                                        handleEditParticipant(participant);
                                      }}
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
                                        setParticipantToRemove(participant)
                                      }
                                    >
                                      <IconTrash size={14} />
                                    </ActionIcon>
                                  </Tooltip>
                                </>
                              )}
                            </Group>
                          </Table.Td>
                        </Table.Tr>
                      );
                    })}
                  </Table.Tbody>
                </Table>
              )
            ) : (
              <Text size="sm" c="dimmed">
                {t("myOrganization.workshops.noParticipants")}
              </Text>
            )}
          </Stack>
        </Stack>
      </Card>

      {/* Invite Link Modal */}
      <Modal
        opened={inviteLinkModalOpened}
        onClose={closeInviteLinkModal}
        title={t("myOrganization.workshops.inviteLinkTitle")}
        size="md"
      >
        <Stack gap="md">
          {inviteLink ? (
            <>
              <Text size="sm">
                {t("myOrganization.workshops.inviteLinkDescription")}
              </Text>
              <Group gap="xs">
                <Code style={{ flex: 1, wordBreak: "break-all" }}>
                  {inviteLink}
                </Code>
                <CopyButton value={inviteLink}>
                  {({ copied, copy }) => (
                    <Tooltip label={copied ? t("copied") : t("copy")}>
                      <ActionIcon
                        variant="subtle"
                        color={copied ? "green" : "gray"}
                        onClick={copy}
                      >
                        {copied ? (
                          <IconCheck size={16} />
                        ) : (
                          <IconCopy size={16} />
                        )}
                      </ActionIcon>
                    </Tooltip>
                  )}
                </CopyButton>
              </Group>
              {(existingInvite?.expiresAt || newlyCreatedInvite?.expiresAt) && (
                <Group gap="xs">
                  <IconClock size={14} color="gray" />
                  <Text size="xs" c="dimmed">
                    {t("myOrganization.workshops.inviteExpires", {
                      date: new Date(
                        existingInvite?.expiresAt ||
                          newlyCreatedInvite?.expiresAt ||
                          "",
                      ).toLocaleDateString(),
                    })}
                  </Text>
                </Group>
              )}
              <Group justify="space-between" mt="md">
                <DangerButton
                  onClick={() =>
                    handleRevokeInviteAndClose(
                      existingInvite?.id || newlyCreatedInvite?.id || "",
                    )
                  }
                  loading={revokeInvite.isPending}
                >
                  {t("myOrganization.workshops.revokeInvite")}
                </DangerButton>
                <TextButton onClick={closeInviteLinkModal}>
                  {t("close")}
                </TextButton>
              </Group>
            </>
          ) : (
            <Text size="sm" c="dimmed">
              {t("myOrganization.workshops.noInviteLink")}
            </Text>
          )}
        </Stack>
      </Modal>

      {/* Remove Participant Confirmation Modal */}
      <ConfirmationModal
        opened={!!participantToRemove}
        onClose={() => setParticipantToRemove(null)}
        onConfirm={handleConfirmRemoveParticipant}
        title={t("myOrganization.workshops.removeParticipantTitle")}
        message={t("myOrganization.workshops.removeParticipantMessage", {
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

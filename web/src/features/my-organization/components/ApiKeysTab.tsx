import { useState, useMemo } from "react";
import {
  Stack,
  Text,
  Alert,
  Card,
  Group,
  Badge,
  Table,
  ActionIcon,
  Loader,
  Center,
  Modal,
  Select,
  TextInput,
  Button,
  Tooltip,
  NumberInput,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import {
  IconKey,
  IconInfoCircle,
  IconTrash,
  IconPlus,
  IconSearch,
  IconChevronDown,
  IconChevronRight,
  IconLink,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useResponsiveDesign } from "@/common/hooks/useResponsiveDesign";
import { useAuth } from "@/providers/AuthProvider";
import { ActionButton, PlusIconButton, DeleteIconButton, EditIconButton } from "@/common/components/buttons";
import { CancelButton } from "@/common/components/buttons";
import {
  ExpandableSearch,
  FilterSegmentedControl,
} from "@/common/components/controls";
import {
  useInstitutionApiKeys,
  useApiKeys,
  useShareApiKeyWithInstitution,
  useRemoveInstitutionApiKeyShare,
  useSetInstitutionFreeUseKey,
  useApiKeyGameShares,
  useRevokePrivateShare,
  useUpdateGameShare,
} from "@/api/hooks";
import type { ObjApiKeyShare, RoutesEnrichedGameShare } from "@/api/generated";
import { AutoShareConfirmModal } from "./AutoShareConfirmModal";
import { useOrgKeyOptions } from "../hooks/useOrgKeyOptions";

interface ApiKeysTabProps {
  institutionId: string;
  institutionName?: string;
  freeUseApiKeyShareId?: string;
}

export function ApiKeysTab({
  institutionId,
  institutionName,
  freeUseApiKeyShareId,
}: ApiKeysTabProps) {
  const { t } = useTranslation("common");
  const { isMobile } = useResponsiveDesign();
  const { backendUser } = useAuth();
  const [shareModalOpened, { open: openShareModal, close: closeShareModal }] =
    useDisclosure(false);
  const [selectedKeyId, setSelectedKeyId] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [platformFilter, setPlatformFilter] = useState<string | null>(null);

  // Fetch institution API keys
  const {
    data: institutionKeys,
    isLoading,
    isError,
  } = useInstitutionApiKeys(institutionId);

  // Fetch user's own keys (for sharing)
  const { data: userKeys } = useApiKeys();

  // Mutations
  const shareApiKey = useShareApiKeyWithInstitution();
  const removeShare = useRemoveInstitutionApiKeyShare();
  const setFreeUseKey = useSetInstitutionFreeUseKey();
  // Combined org + personal key options for free-use key selector
  const {
    options: freeUseKeyOptions,
    personalKeyIds: freeUsePersonalKeyIds,
    personalSelfShareIds: freeUsePersonalSelfShareIds,
    personalKeyNames: freeUsePersonalKeyNames,
  } = useOrgKeyOptions(institutionId);

  // Auto-share confirmation state for free-use key
  const [freeUseAutoSharePending, setFreeUseAutoSharePending] = useState<string | null>(null);
  const [freeUseAutoShareError, setFreeUseAutoShareError] = useState<string | null>(null);

  // Get unique platforms for filter
  const platforms = useMemo(() => {
    if (!institutionKeys) return [];
    const unique = new Set(
      institutionKeys.map((s) => s.apiKey?.platform).filter(Boolean),
    );
    return Array.from(unique) as string[];
  }, [institutionKeys]);

  // Filter institution keys by search and platform
  const filteredKeys = useMemo(() => {
    if (!institutionKeys) return [];
    return institutionKeys.filter((share) => {
      const matchesSearch =
        !searchQuery ||
        share.apiKey?.name?.toLowerCase().includes(searchQuery.toLowerCase()) ||
        share.apiKey?.userName
          ?.toLowerCase()
          .includes(searchQuery.toLowerCase());
      const matchesPlatform =
        !platformFilter || share.apiKey?.platform === platformFilter;
      return matchesSearch && matchesPlatform;
    });
  }, [institutionKeys, searchQuery, platformFilter]);

  // Filter user's own keys that aren't already shared with the institution
  const ownKeys = userKeys?.apiKeys ?? [];
  const ownShares = userKeys?.shares ?? [];
  const availableKeysToShare = ownKeys.filter((key) => {
    // Only show keys the user owns
    if (key.userId !== backendUser?.id) return false;
    // Don't show keys already shared with this institution
    const alreadyShared = institutionKeys?.some(
      (instShare) => instShare.apiKeyId === key.id,
    );
    return !alreadyShared;
  });

  const handleShareKey = async () => {
    if (!selectedKeyId) return;

    // Find the user's self-share for this API key (backend expects share ID, not key ID)
    const selfShare = ownShares.find(
      (s) =>
        s.apiKeyId === selectedKeyId &&
        s.user &&
        !s.institution &&
        !s.workshop &&
        !s.game,
    );
    if (!selfShare?.id) return;

    await shareApiKey.mutateAsync({
      shareId: selfShare.id,
      institutionId,
    });

    closeShareModal();
    setSelectedKeyId(null);
  };

  const [expandedShareId, setExpandedShareId] = useState<string | null>(null);

  const handleRemoveShare = async (share: ObjApiKeyShare) => {
    if (!share.id) return;
    await removeShare.mutateAsync({
      shareId: share.id,
      institutionId,
    });
  };

  const handleFreeUseAutoShareConfirm = async () => {
    if (!freeUseAutoSharePending) return;
    const selfShareId = freeUsePersonalSelfShareIds.get(freeUseAutoSharePending);
    if (!selfShareId) return;

    try {
      // Step 1: Share the personal key with the institution
      const newShare = await shareApiKey.mutateAsync({
        shareId: selfShareId,
        institutionId,
      });
      // Step 2: Set the new institution share as the free-use key
      if (newShare?.id) {
        await setFreeUseKey.mutateAsync({
          institutionId,
          shareId: newShare.id,
        });
      }
      setFreeUseAutoSharePending(null);
      setFreeUseAutoShareError(null);
    } catch {
      setFreeUseAutoShareError(t("myOrganization.apiKeys.loadError"));
    }
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
      <Alert color="red" title={t("error")}>
        {t("myOrganization.apiKeys.loadError")}
      </Alert>
    );
  }

  const hasKeys = institutionKeys && institutionKeys.length > 0;

  return (
    <Stack gap="lg">
      {/* Info block */}
      <Alert
        icon={<IconInfoCircle size={16} />}
        title={t("myOrganization.apiKeys.aboutTitle")}
        color="cyan"
      >
        <Text size="sm">{t("myOrganization.apiKeys.aboutDescription")}</Text>
      </Alert>

      {/* Actions and filters */}
      {isMobile ? (
        <Group gap="sm" wrap="nowrap">
          <Tooltip label={t("myOrganization.apiKeys.shareKey")} withArrow>
            <PlusIconButton
              onClick={openShareModal}
              variant="filled"
              disabled={availableKeysToShare.length === 0}
              aria-label={t("myOrganization.apiKeys.shareKey")}
            />
          </Tooltip>
          <ExpandableSearch
            value={searchQuery}
            onChange={setSearchQuery}
            placeholder={t("myOrganization.apiKeys.searchPlaceholder")}
          />
          {hasKeys && platforms.length > 1 && (
            <FilterSegmentedControl
              value={platformFilter || "all"}
              onChange={(v) => setPlatformFilter(v === "all" ? null : v)}
              options={[
                {
                  value: "all",
                  label: t("myOrganization.apiKeys.allPlatforms", "All"),
                },
                ...platforms.map((p) => ({ value: p, label: p })),
              ]}
            />
          )}
        </Group>
      ) : (
        <Group justify="space-between" wrap="wrap" gap="sm">
          <ActionButton
            leftSection={<IconPlus size={16} />}
            onClick={openShareModal}
            disabled={availableKeysToShare.length === 0}
          >
            {t("myOrganization.apiKeys.shareKey")}
          </ActionButton>

          {hasKeys && (
            <Group gap="sm" wrap="wrap">
              <TextInput
                placeholder={t("myOrganization.apiKeys.searchPlaceholder")}
                leftSection={<IconSearch size={16} />}
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.currentTarget.value)}
                size="sm"
                style={{ minWidth: 200 }}
              />
              <Select
                placeholder={t("myOrganization.apiKeys.filterByPlatform")}
                data={platforms.map((p) => ({ value: p, label: p }))}
                value={platformFilter}
                onChange={setPlatformFilter}
                clearable
                size="sm"
                style={{ minWidth: 150 }}
              />
            </Group>
          )}
        </Group>
      )}

      {/* Keys list */}
      {hasKeys ? (
        <Card
          shadow="sm"
          padding={isMobile ? "xs" : "lg"}
          radius="md"
          withBorder
        >
          {filteredKeys.length > 0 ? (
            isMobile ? (
              <Stack gap="xs">
                {filteredKeys.map((share) => (
                  <OrgKeyCardMobile
                    key={share.id}
                    share={share}
                    expanded={expandedShareId === share.id}
                    onToggle={() =>
                      setExpandedShareId((prev) =>
                        prev === share.id ? null : (share.id ?? null),
                      )
                    }
                    onRemove={() => handleRemoveShare(share)}
                    isRemoving={removeShare.isPending}
                  />
                ))}
              </Stack>
            ) : (
              <Table striped highlightOnHover>
                <Table.Thead>
                  <Table.Tr>
                    <Table.Th>{t("myOrganization.apiKeys.keyName")}</Table.Th>
                    <Table.Th>{t("myOrganization.apiKeys.owner")}</Table.Th>
                    <Table.Th>{t("myOrganization.apiKeys.platform")}</Table.Th>
                    <Table.Th>{t("myOrganization.actions")}</Table.Th>
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {filteredKeys.map((share) => (
                    <OrgKeyRow
                      key={share.id}
                      share={share}
                      expanded={expandedShareId === share.id}
                      onToggle={() =>
                        setExpandedShareId((prev) =>
                          prev === share.id ? null : (share.id ?? null),
                        )
                      }
                      onRemove={() => handleRemoveShare(share)}
                      isRemoving={removeShare.isPending}
                    />
                  ))}
                </Table.Tbody>
              </Table>
            )
          ) : (
            <Stack gap="md" align="center" py="xl">
              <IconSearch size={48} color="var(--mantine-color-dimmed)" />
              <Text c="dimmed" ta="center">
                {t("myOrganization.apiKeys.noResults")}
              </Text>
            </Stack>
          )}
        </Card>
      ) : (
        <Card
          shadow="sm"
          padding={isMobile ? "md" : "lg"}
          radius="md"
          withBorder
        >
          <Stack gap="md" align="center" py="xl">
            <IconKey size={48} color="var(--mantine-color-dimmed)" />
            <Text c="dimmed" ta="center">
              {t("myOrganization.apiKeys.noKeys")}
            </Text>
            <Badge color="cyan" variant="light">
              {institutionName || institutionId}
            </Badge>
          </Stack>
        </Card>
      )}

      {/* Free-Use Key Section */}
      <Card shadow="sm" padding={isMobile ? "md" : "lg"} radius="md" withBorder>
        <Stack gap="md">
          <Group gap="xs">
            <IconKey size={20} />
            <Text fw={600} size="sm">
              {t("myOrganization.apiKeys.freeUseKey.title")}
            </Text>
          </Group>
          <Text size="sm" c="dimmed">
            {t("myOrganization.apiKeys.freeUseKey.description")}
          </Text>

          {freeUseApiKeyShareId ? (
            (() => {
              const currentFreeUseShare = institutionKeys?.find(
                (s) => s.id === freeUseApiKeyShareId,
              );
              return (
                <Group gap="sm" wrap="wrap">
                  <Badge color="cyan" variant="light" size="lg">
                    {currentFreeUseShare?.apiKey?.name ||
                      t("myOrganization.apiKeys.freeUseKey.unknownKey")}
                    {currentFreeUseShare?.apiKey?.platform &&
                      ` (${currentFreeUseShare.apiKey.platform})`}
                  </Badge>
                  <Button
                    variant="subtle"
                    color="red"
                    size="xs"
                    onClick={() =>
                      setFreeUseKey.mutate({
                        institutionId,
                        shareId: null,
                      })
                    }
                    loading={setFreeUseKey.isPending}
                  >
                    {t("myOrganization.apiKeys.freeUseKey.remove")}
                  </Button>
                </Group>
              );
            })()
          ) : (
            <Select
              placeholder={t(
                "myOrganization.apiKeys.freeUseKey.selectPlaceholder",
              )}
              data={freeUseKeyOptions}
              onChange={(value) => {
                if (!value) return;
                // If user selected a personal key, open confirmation modal
                if (freeUsePersonalKeyIds.has(value)) {
                  setFreeUseAutoSharePending(value);
                  setFreeUseAutoShareError(null);
                  return;
                }
                setFreeUseKey.mutate({
                  institutionId,
                  shareId: value,
                });
              }}
              disabled={setFreeUseKey.isPending}
              clearable={false}
              size="sm"
              style={{ maxWidth: 400 }}
            />
          )}
        </Stack>
      </Card>

      {/* Share Key Modal */}
      <Modal
        opened={shareModalOpened}
        onClose={closeShareModal}
        title={t("myOrganization.apiKeys.shareKeyTitle")}
      >
        <Stack gap="md">
          <Select
            label={t("myOrganization.apiKeys.selectKey")}
            placeholder={t("myOrganization.apiKeys.selectKeyPlaceholder")}
            data={availableKeysToShare.map((key) => ({
              value: key.id || "",
              label: `${key.name || t("unnamed")} (${key.platform})`,
            }))}
            value={selectedKeyId}
            onChange={setSelectedKeyId}
          />

          <Group justify="flex-end" mt="md">
            <Button variant="default" onClick={closeShareModal}>
              {t("cancel")}
            </Button>
            <Button
              onClick={handleShareKey}
              loading={shareApiKey.isPending}
              disabled={!selectedKeyId}
            >
              {t("myOrganization.apiKeys.share")}
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Auto-share personal key confirmation for free-use key */}
      <AutoShareConfirmModal
        opened={!!freeUseAutoSharePending}
        onClose={() => {
          setFreeUseAutoSharePending(null);
          setFreeUseAutoShareError(null);
        }}
        onConfirm={handleFreeUseAutoShareConfirm}
        keyName={
          freeUseAutoSharePending
            ? (freeUsePersonalKeyNames.get(freeUseAutoSharePending) ?? "")
            : ""
        }
        orgName={institutionName ?? institutionId}
        isLoading={shareApiKey.isPending || setFreeUseKey.isPending}
        error={freeUseAutoShareError}
      />
    </Stack>
  );
}

// --- Mobile card for org API key ---

interface OrgKeyRowProps {
  share: ObjApiKeyShare;
  expanded: boolean;
  onToggle: () => void;
  onRemove: () => void;
  isRemoving: boolean;
}

function OrgKeyCardMobile({
  share,
  expanded,
  onToggle,
  onRemove,
  isRemoving,
}: OrgKeyRowProps) {
  const { t } = useTranslation("common");

  return (
    <Card withBorder radius="sm" padding="sm">
      <Group
        gap="xs"
        justify="space-between"
        wrap="nowrap"
        onClick={onToggle}
        style={{ cursor: "pointer" }}
      >
        <Group gap="xs" wrap="nowrap" style={{ minWidth: 0, flex: 1 }}>
          {expanded ? (
            <IconChevronDown size={14} color="var(--mantine-color-dimmed)" style={{ flexShrink: 0 }} />
          ) : (
            <IconChevronRight size={14} color="var(--mantine-color-dimmed)" style={{ flexShrink: 0 }} />
          )}
          <IconKey size={16} style={{ flexShrink: 0 }} />
          <Stack gap={0} style={{ minWidth: 0 }}>
            <Text size="sm" fw={500} truncate>
              {share.apiKey?.name || t("unnamed")}
            </Text>
            <Group gap={4}>
              <Text size="xs" c="dimmed" truncate>
                {share.apiKey?.userName}
              </Text>
              {share.apiKey?.platform && (
                <Badge size="xs" variant="light" color="gray">
                  {share.apiKey.platform}
                </Badge>
              )}
            </Group>
          </Stack>
        </Group>
        <ActionIcon
          color="red"
          variant="subtle"
          size="sm"
          onClick={(e) => {
            e.stopPropagation();
            onRemove();
          }}
          loading={isRemoving}
        >
          <IconTrash size={14} />
        </ActionIcon>
      </Group>
      {expanded && <OrgKeyDetails shareId={share.id ?? ""} />}
    </Card>
  );
}

// --- Expandable row for org API key table (desktop) ---

function OrgKeyRow({
  share,
  expanded,
  onToggle,
  onRemove,
  isRemoving,
}: OrgKeyRowProps) {
  const { t } = useTranslation("common");

  return (
    <>
      <Table.Tr
        style={{ cursor: "pointer" }}
        onClick={onToggle}
      >
        <Table.Td>
          <Group gap="xs">
            {expanded ? (
              <IconChevronDown size={14} color="var(--mantine-color-dimmed)" />
            ) : (
              <IconChevronRight size={14} color="var(--mantine-color-dimmed)" />
            )}
            <IconKey size={16} />
            <Text size="sm">{share.apiKey?.name || t("unnamed")}</Text>
          </Group>
        </Table.Td>
        <Table.Td>
          <Text size="sm">{share.apiKey?.userName}</Text>
        </Table.Td>
        <Table.Td>
          <Text size="sm">{share.apiKey?.platform}</Text>
        </Table.Td>
        <Table.Td>
          <ActionIcon
            color="red"
            variant="subtle"
            onClick={(e) => {
              e.stopPropagation();
              onRemove();
            }}
            loading={isRemoving}
          >
            <IconTrash size={16} />
          </ActionIcon>
        </Table.Td>
      </Table.Tr>
      {expanded && (
        <Table.Tr>
          <Table.Td colSpan={4} style={{ padding: 0 }}>
            <OrgKeyDetails shareId={share.id ?? ""} />
          </Table.Td>
        </Table.Tr>
      )}
    </>
  );
}

function OrgKeyDetails({ shareId }: { shareId: string }) {
  const { t } = useTranslation("common");
  const { data: gameShares, isLoading } = useApiKeyGameShares(
    shareId,
    "organization",
  );

  if (isLoading) {
    return (
      <Center py="sm">
        <Loader size="sm" />
      </Center>
    );
  }

  if (!gameShares || gameShares.length === 0) {
    return (
      <Text size="sm" c="dimmed" px="md" py="sm">
        {t("apiKeys.shares.noSponsorships")}
      </Text>
    );
  }

  return (
    <Stack gap="sm" px="md" py="sm">
      {gameShares.length > 0 && (
        <Stack gap={4}>
          <Group gap={8} align="center">
            <IconLink size={18} color="var(--mantine-color-accent-5)" />
            <Text size="sm" fw={600} c="dimmed" tt="uppercase">
              {t("apiKeys.shares.gameShares")} ({gameShares.length})
            </Text>
          </Group>
          <Stack gap={6} pl="md">
            {gameShares.map((gs) => (
              <OrgGameShareRow key={gs.id} gameShare={gs} />
            ))}
          </Stack>
        </Stack>
      )}
    </Stack>
  );
}

function OrgGameShareRow({ gameShare }: { gameShare: RoutesEnrichedGameShare }) {
  const { t } = useTranslation("common");
  const { isMobile } = useResponsiveDesign();
  const revokeShare = useRevokePrivateShare();
  const updateShare = useUpdateGameShare();
  const [editModalOpened, { open: openEditModal, close: closeEditModal }] =
    useDisclosure(false);
  const [editMaxSessions, setEditMaxSessions] = useState<number | string>(
    gameShare.remaining ?? "",
  );

  const getContextLabel = (): string => {
    if (gameShare.source === "workshop") {
      return gameShare.workshopName
        ? `${t("apiKeys.shares.context.workshop")}: ${gameShare.workshopName}`
        : t("apiKeys.shares.context.workshop");
    }
    if (gameShare.source === "organization") {
      return t("apiKeys.shares.context.organization");
    }
    return t("apiKeys.shares.context.personal");
  };

  const getRemainingLabel = (): string => {
    if (gameShare.remaining == null) return t("apiKeys.shares.unlimited");
    return t("apiKeys.shares.remaining", { count: gameShare.remaining });
  };

  const handleEditSubmit = async () => {
    if (!gameShare.gameId || !gameShare.id) return;
    try {
      await updateShare.mutateAsync({
        gameId: gameShare.gameId,
        shareId: gameShare.id,
        maxSessions:
          typeof editMaxSessions === "number" && editMaxSessions > 0
            ? editMaxSessions
            : null,
      });
      closeEditModal();
    } catch {
      // Error handled by mutation
    }
  };

  const handleOpenEdit = () => {
    setEditMaxSessions(gameShare.remaining ?? "");
    openEditModal();
  };

  return (
    <>
      <Group gap="sm" align="center" justify="space-between" wrap="nowrap">
        <Stack gap={2} style={{ minWidth: 0, flex: 1 }}>
          <Group gap="xs" wrap="wrap">
            <Text size="sm" fw={500} truncate>
              {gameShare.gameName || t("apiKeys.shares.unknownGame")}
            </Text>
            <Badge size="xs" variant="light" color="gray">
              {getContextLabel()}
            </Badge>
          </Group>
          <Text size="xs" c="dimmed">
            {getRemainingLabel()}
          </Text>
        </Stack>
        <Group gap={4} wrap="nowrap" style={{ flexShrink: 0 }}>
          <EditIconButton
            size="sm"
            onClick={handleOpenEdit}
            aria-label={t("apiKeys.shares.editRemaining")}
          />
          <DeleteIconButton
            size="sm"
            onClick={() => {
              if (gameShare.gameId && gameShare.id) {
                revokeShare.mutate({
                  gameId: gameShare.gameId,
                  shareId: gameShare.id,
                });
              }
            }}
            aria-label={t("apiKeys.shares.revokeGameShare")}
          />
        </Group>
      </Group>

      <Modal
        opened={editModalOpened}
        onClose={closeEditModal}
        title={t("apiKeys.shares.editRemaining")}
        size="sm"
        fullScreen={isMobile}
        centered={!isMobile}
      >
        <Stack gap="md">
          <NumberInput
            label={t("games.privateShare.maxSessions")}
            description={t("games.privateShare.maxSessionsDescription")}
            placeholder={t("apiKeys.shares.unlimited")}
            value={editMaxSessions}
            onChange={(v) => setEditMaxSessions(v === "" ? "" : v)}
            min={1}
            allowNegative={false}
            allowDecimal={false}
          />
          <Group justify="flex-end" gap="sm">
            <CancelButton onClick={closeEditModal}>
              {t("cancel")}
            </CancelButton>
            <ActionButton
              onClick={handleEditSubmit}
              loading={updateShare.isPending}
              size="sm"
            >
              {t("save")}
            </ActionButton>
          </Group>
        </Stack>
      </Modal>
    </>
  );
}

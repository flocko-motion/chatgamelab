import {
  Stack,
  Group,
  Text,
  Modal,
  Select,
  Alert,
  NumberInput,
  CopyButton,
  ActionIcon,
  Tooltip,
  Badge,
} from "@mantine/core";
import {
  IconAlertCircle,
  IconLink,
  IconCheck,
  IconCopy,
  IconSettings,
} from "@tabler/icons-react";
import { useMediaQuery, useDisclosure } from "@mantine/hooks";
import { useTranslation } from "react-i18next";
import { useState } from "react";
import { ActionButton, CancelButton, DangerButton } from "@components/buttons";
import {
  useApiKeys,
  useInstitutionApiKeys,
  usePrivateShareStatus,
  useCreateGameShare,
  useRevokePrivateShare,
  useUpdateGameShare,
  type EnrichedGameShare,
} from "@/api/hooks";
import { useAuth } from "@/providers/AuthProvider";
import { getAiQualityTierOptions } from "@/common/lib/aiQualityTier";

interface PrivateShareSectionProps {
  gameId: string;
  workshopId?: string;
}

export function PrivateShareSection({
  gameId,
  workshopId,
}: PrivateShareSectionProps) {
  const { t } = useTranslation("common");
  const isMobile = useMediaQuery("(max-width: 48em)");
  const { backendUser } = useAuth();
  const isWorkshopMode = !!workshopId;

  const institutionId = backendUser?.role?.institution?.id;
  const { data: apiKeys, isLoading: personalKeysLoading } = useApiKeys();
  const { data: institutionKeys, isLoading: instKeysLoading } =
    useInstitutionApiKeys(institutionId ?? "");
  const keysLoading = personalKeysLoading || instKeysLoading;
  const { data: shareStatus, isLoading: statusLoading } =
    usePrivateShareStatus(gameId);
  const createGameShare = useCreateGameShare();
  const revokeShare = useRevokePrivateShare();
  const updateShare = useUpdateGameShare();

  const [modalOpened, { open: openModal, close: closeModal }] =
    useDisclosure(false);
  const [editModalOpened, { open: openEditModal, close: closeEditModal }] =
    useDisclosure(false);
  const [selectedShareId, setSelectedShareId] = useState<string | null>(null);
  const [maxSessions, setMaxSessions] = useState<number | string>("");
  const [selectedTier, setSelectedTier] = useState<string | null>(null);
  const [editingShare, setEditingShare] = useState<EnrichedGameShare | null>(null);
  const [editMaxSessions, setEditMaxSessions] = useState<number | string>("");
  const [editTier, setEditTier] = useState<string | null>(null);

  const shares = shareStatus?.shares ?? [];

  // Eligible keys for personal/org share creation — grouped like SponsorGameModal
  const keys = apiKeys?.apiKeys ?? [];
  const allShares = apiKeys?.shares ?? [];
  const eligibleKeys: { id: string; name: string; platform: string; isOrg: boolean }[] = [];
  if (!isWorkshopMode) {
    const seenShareIds = new Set<string>();
    for (const share of allShares) {
      if (!share.id || seenShareIds.has(share.id)) continue;
      const apiKey = keys.find((k) => k.id === share.apiKeyId);
      if (apiKey?.lastUsageSuccess === false) continue;
      // Only include pure self-shares (no institution, workshop, or game context)
      if (share.institution || share.workshop || share.game) continue;
      seenShareIds.add(share.id);
      eligibleKeys.push({
        id: share.id,
        name: apiKey?.name ?? "Unknown",
        platform: apiKey?.platform ?? "?",
        isOrg: false,
      });
    }
    for (const share of institutionKeys ?? []) {
      if (!share.id || seenShareIds.has(share.id)) continue;
      if (share.apiKey?.lastUsageSuccess === false) continue;
      seenShareIds.add(share.id);
      eligibleKeys.push({
        id: share.id,
        name: share.apiKey?.name ?? "Unknown",
        platform: share.apiKey?.platform ?? "?",
        isOrg: true,
      });
    }
  }

  const personalItems = eligibleKeys
    .filter((k) => !k.isOrg)
    .map((k) => ({ value: k.id, label: `${k.name} (${k.platform})` }));
  const orgItems = eligibleKeys
    .filter((k) => k.isOrg)
    .map((k) => ({ value: k.id, label: `${k.name} (${k.platform})` }));
  const selectData = [
    ...(personalItems.length > 0
      ? [{ group: t("games.privateShare.sourcePersonal"), items: personalItems }]
      : []),
    ...(orgItems.length > 0
      ? [{ group: t("games.privateShare.sourceOrg"), items: orgItems }]
      : []),
  ];

  const handleOpenCreate = () => {
    setSelectedShareId(null);
    setMaxSessions("");
    setSelectedTier(null);
    openModal();
  };

  const handleSubmit = async () => {
    if (!isWorkshopMode && !selectedShareId) return;
    try {
      await createGameShare.mutateAsync({
        gameId,
        workshopId: isWorkshopMode ? workshopId : undefined,
        sponsorKeyShareId: isWorkshopMode ? undefined : selectedShareId!,
        maxSessions:
          typeof maxSessions === "number" && maxSessions > 0
            ? maxSessions
            : null,
        aiQualityTier: !isWorkshopMode && selectedTier ? selectedTier : null,
      });
      closeModal();
    } catch {
      // Error handled by mutation
    }
  };

  const handleRevoke = async (shareId: string) => {
    try {
      await revokeShare.mutateAsync({ gameId, shareId });
    } catch {
      // Error handled by mutation
    }
  };

  const handleOpenEdit = (share: EnrichedGameShare) => {
    setEditingShare(share);
    setEditMaxSessions(share.remaining ?? "");
    setEditTier(share.aiQualityTier ?? null);
    openEditModal();
  };

  const handleEditSubmit = async () => {
    if (!editingShare) return;
    try {
      await updateShare.mutateAsync({
        gameId,
        shareId: editingShare.id,
        maxSessions:
          typeof editMaxSessions === "number" && editMaxSessions > 0
            ? editMaxSessions
            : null,
        aiQualityTier: editingShare.source !== "workshop" && editTier ? editTier : null,
      });
      closeEditModal();
    } catch {
      // Error handled by mutation
    }
  };

  const isSubmitting = createGameShare.isPending;
  const canSubmit = isWorkshopMode || !!selectedShareId;

  if (statusLoading) {
    return (
      <Text size="sm" c="dimmed">
        {t("games.privateShare.loading")}
      </Text>
    );
  }

  const workshopShares = shares.filter((s) => s.source === "workshop");
  const personalOrgShares = shares.filter((s) => s.source !== "workshop");

  return (
    <>
      <Stack gap="md">
        {/* Workshop share subsection */}
        {(isWorkshopMode || workshopShares.length > 0) && (
          <Stack gap="xs">
            <Text size="sm" fw={600}>
              {t("games.privateShare.workshopSubsection")}
            </Text>
            {workshopShares.length > 0 ? (
              workshopShares.map((share) => (
                <ShareCard
                  key={share.id}
                  share={share}
                  onRevoke={() => handleRevoke(share.id)}
                  onEdit={() => handleOpenEdit(share)}
                  isRevoking={revokeShare.isPending}
                  t={t}
                />
              ))
            ) : isWorkshopMode ? (
              <>
                <Text size="sm" c="dimmed">
                  {t("games.privateShare.workshopCreateHint")}
                </Text>
                <div>
                  <ActionButton
                    onClick={handleOpenCreate}
                    size="sm"
                    leftSection={<IconLink size={16} />}
                  >
                    {t("games.privateShare.enable")}
                  </ActionButton>
                </div>
              </>
            ) : null}
          </Stack>
        )}

        {/* Personal / org share subsection */}
        {(personalOrgShares.length > 0 || !isWorkshopMode) && (
          <Stack gap="xs">
            <Text size="sm" fw={600}>
              {t("games.privateShare.personalOrgSubsection")}
            </Text>
            {personalOrgShares.length > 0 ? (
              personalOrgShares.map((share) => (
                <ShareCard
                  key={share.id}
                  share={share}
                  onRevoke={() => handleRevoke(share.id)}
                  onEdit={() => handleOpenEdit(share)}
                  isRevoking={revokeShare.isPending}
                  t={t}
                />
              ))
            ) : !isWorkshopMode ? (
              <>
                <Text size="sm" c="dimmed">
                  {t("games.privateShare.personalCreateHint")}
                </Text>
                <div>
                  <ActionButton
                    onClick={handleOpenCreate}
                    size="sm"
                    leftSection={<IconLink size={16} />}
                  >
                    {t("games.privateShare.enable")}
                  </ActionButton>
                </div>
              </>
            ) : null}
            {isWorkshopMode && personalOrgShares.length === 0 && (
              <Text size="sm" c="dimmed">
                {t("games.privateShare.noPersonalShare")}
              </Text>
            )}
          </Stack>
        )}
      </Stack>

      <Modal
        opened={modalOpened}
        onClose={closeModal}
        title={t("games.privateShare.createModalTitle")}
        size="sm"
        fullScreen={isMobile}
        centered={!isMobile}
      >
        <Stack gap="md">
          <Text size="sm" c="dimmed">
            {isWorkshopMode
              ? t("games.privateShare.workshopDescription")
              : t("games.privateShare.description")}
          </Text>

          {!isWorkshopMode && (
            <>
              {keysLoading ? (
                <Text size="sm" c="dimmed">
                  {t("games.privateShare.loadingKeys")}
                </Text>
              ) : selectData.length === 0 ? (
                <Alert
                  icon={<IconAlertCircle size={16} />}
                  color="yellow"
                  variant="light"
                >
                  {t("games.privateShare.noEligibleKeys")}
                </Alert>
              ) : (
                <Select
                  label={t("games.privateShare.selectKey")}
                  placeholder={t("games.privateShare.selectKeyPlaceholder")}
                  data={selectData}
                  value={selectedShareId}
                  onChange={setSelectedShareId}
                  searchable={eligibleKeys.length > 5}
                />
              )}
            </>
          )}

          {!isWorkshopMode && selectData.length > 0 && (
            <Select
              label={t("games.privateShare.aiQualityTier")}
              description={t("games.privateShare.aiQualityTierDescription")}
              data={getAiQualityTierOptions(t, { includeEmpty: true })}
              value={selectedTier ?? ""}
              onChange={(v) => setSelectedTier(v || null)}
              allowDeselect={false}
            />
          )}

          {(isWorkshopMode || selectData.length > 0) && (
            <NumberInput
              label={t("games.privateShare.maxSessions")}
              description={t("games.privateShare.maxSessionsDescription")}
              placeholder={t("games.privateShare.unlimited")}
              value={maxSessions}
              onChange={(v) => setMaxSessions(v === "" ? "" : v)}
              min={1}
              allowNegative={false}
              allowDecimal={false}
            />
          )}

          <Group justify="flex-end" gap="sm">
            <CancelButton onClick={closeModal}>{t("cancel")}</CancelButton>
            <ActionButton
              onClick={handleSubmit}
              disabled={!canSubmit}
              loading={isSubmitting}
              size="sm"
              leftSection={<IconLink size={16} />}
            >
              {t("games.privateShare.enable")}
            </ActionButton>
          </Group>
        </Stack>
      </Modal>

      <Modal
        opened={editModalOpened}
        onClose={closeEditModal}
        title={t("games.privateShare.editModalTitle")}
        size="sm"
        fullScreen={isMobile}
        centered={!isMobile}
      >
        <Stack gap="md">
          {editingShare && editingShare.source !== "workshop" && (
            <Select
              label={t("games.privateShare.aiQualityTier")}
              description={t("games.privateShare.aiQualityTierDescription")}
              data={getAiQualityTierOptions(t, { includeEmpty: true })}
              value={editTier ?? ""}
              onChange={(v) => setEditTier(v || null)}
              allowDeselect={false}
            />
          )}

          <NumberInput
            label={t("games.privateShare.maxSessions")}
            description={t("games.privateShare.maxSessionsDescription")}
            placeholder={t("games.privateShare.unlimited")}
            value={editMaxSessions}
            onChange={(v) => setEditMaxSessions(v === "" ? "" : v)}
            min={1}
            allowNegative={false}
            allowDecimal={false}
          />

          <Group justify="flex-end" gap="sm">
            <CancelButton onClick={closeEditModal}>{t("cancel")}</CancelButton>
            <ActionButton
              onClick={handleEditSubmit}
              loading={updateShare.isPending}
              size="sm"
            >
              {t("games.privateShare.update")}
            </ActionButton>
          </Group>
        </Stack>
      </Modal>
    </>
  );
}

function ShareCard({
  share,
  onRevoke,
  onEdit,
  isRevoking,
  t,
}: {
  share: EnrichedGameShare;
  onRevoke: () => void;
  onEdit: () => void;
  isRevoking: boolean;
  t: (key: string, options?: Record<string, unknown>) => string;
}) {
  const shareUrl =
    typeof window !== "undefined"
      ? `${window.location.origin}${share.shareUrl}`
      : "";

  const sourceLabel =
    share.source === "workshop"
      ? t("games.privateShare.workshopShareDescription", {
          name: share.workshopName ?? "",
        })
      : share.source === "organization"
        ? t("games.privateShare.orgShareDescription")
        : t("games.privateShare.personalShareDescription");

  const sourceBadge =
    share.source === "workshop"
      ? t("games.privateShare.sourceWorkshop")
      : share.source === "organization"
        ? t("games.privateShare.sourceOrg")
        : t("games.privateShare.sourcePersonal");

  const remainingLabel =
    share.remaining != null
      ? t("games.privateShare.remainingSessions", { count: share.remaining })
      : t("games.privateShare.unlimitedSessions");

  return (
    <Stack gap="xs">
      <Group gap="xs">
        <Badge size="xs" variant="light" color={
          share.source === "workshop" ? "blue" :
          share.source === "organization" ? "grape" : "teal"
        }>
          {sourceBadge}
        </Badge>
        {share.aiQualityTier && (
          <Badge size="xs" variant="light" color="violet">
            {t(`aiQualityTier.${share.aiQualityTier}`)}
          </Badge>
        )}
        <Text size="xs" c="green" fw={500}>
          {remainingLabel}
        </Text>
      </Group>

      <Text size="sm" c="dimmed">
        {sourceLabel}
      </Text>

      {shareUrl && (
        <Group gap="xs" wrap="nowrap">
          <Text
            size="xs"
            style={{
              wordBreak: "break-all",
              flex: 1,
              fontFamily: "monospace",
            }}
          >
            {shareUrl}
          </Text>
          <CopyButton value={shareUrl}>
            {({ copied, copy }) => (
              <Tooltip
                label={
                  copied ? t("copied") : t("games.privateShare.copyLink")
                }
              >
                <ActionIcon
                  color={copied ? "teal" : "gray"}
                  onClick={copy}
                  variant="subtle"
                  size="sm"
                >
                  {copied ? (
                    <IconCheck size={14} />
                  ) : (
                    <IconCopy size={14} />
                  )}
                </ActionIcon>
              </Tooltip>
            )}
          </CopyButton>
        </Group>
      )}

      <Group gap="sm" wrap="wrap">
        <CopyButton value={shareUrl}>
          {({ copied, copy }) => (
            <ActionButton
              onClick={copy}
              size="xs"
              leftSection={
                copied ? <IconCheck size={14} /> : <IconCopy size={14} />
              }
            >
              {copied ? t("copied") : t("games.privateShare.copyLink")}
            </ActionButton>
          )}
        </CopyButton>
        <ActionButton
          onClick={onEdit}
          size="xs"
          leftSection={<IconSettings size={14} />}
        >
          {t("games.privateShare.editSettings")}
        </ActionButton>
        <DangerButton
          onClick={onRevoke}
          loading={isRevoking}
          size="xs"
        >
          {t("games.privateShare.revoke")}
        </DangerButton>
      </Group>
    </Stack>
  );
}

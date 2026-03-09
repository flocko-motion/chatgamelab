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
} from "@mantine/core";
import {
  IconAlertCircle,
  IconLink,
  IconCheck,
  IconCopy,
  IconEdit,
} from "@tabler/icons-react";
import { useMediaQuery, useDisclosure } from "@mantine/hooks";
import { useTranslation } from "react-i18next";
import { useState } from "react";
import { ActionButton, CancelButton, DangerButton } from "@components/buttons";
import {
  useApiKeys,
  usePrivateShareStatus,
  useEnablePrivateShare,
  useCreateGameShare,
  useRevokePrivateShare,
} from "@/api/hooks";

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
  const isWorkshopMode = !!workshopId;

  const { data: apiKeys, isLoading: keysLoading } = useApiKeys();
  const { data: shareStatus, isLoading: statusLoading } =
    usePrivateShareStatus(gameId);
  const enableShare = useEnablePrivateShare();
  const createGameShare = useCreateGameShare();
  const revokeShare = useRevokePrivateShare();

  const [modalOpened, { open: openModal, close: closeModal }] =
    useDisclosure(false);
  const [modalMode, setModalMode] = useState<"create" | "edit">("create");
  const [selectedShareId, setSelectedShareId] = useState<string | null>(null);
  const [maxSessions, setMaxSessions] = useState<number | string>("");

  const isEnabled = shareStatus?.enabled ?? false;

  // Eligible keys (personal mode only)
  const keys = apiKeys?.apiKeys ?? [];
  const shares = apiKeys?.shares ?? [];
  const selectData: { value: string; label: string }[] = [];
  if (!isWorkshopMode) {
    const seenKeyIds = new Set<string>();
    for (const share of shares) {
      if (!share.apiKeyId || seenKeyIds.has(share.apiKeyId)) continue;
      const apiKey = keys.find((k) => k.id === share.apiKeyId);
      if (apiKey?.lastUsageSuccess === false) continue;
      seenKeyIds.add(share.apiKeyId);
      selectData.push({
        value: share.id!,
        label: `${apiKey?.name ?? "Unknown"} (${apiKey?.platform ?? "?"})`,
      });
    }
  }

  const handleOpenCreate = () => {
    setModalMode("create");
    setSelectedShareId(null);
    setMaxSessions("");
    openModal();
  };

  const handleOpenEdit = () => {
    setModalMode("edit");
    setSelectedShareId(null);
    setMaxSessions(shareStatus?.remaining ?? "");
    openModal();
  };

  const handleSubmit = async () => {
    try {
      if (isWorkshopMode) {
        await createGameShare.mutateAsync({
          gameId,
          workshopId,
          maxSessions:
            typeof maxSessions === "number" && maxSessions > 0
              ? maxSessions
              : null,
        });
      } else {
        if (!selectedShareId) return;
        await enableShare.mutateAsync({
          gameId,
          sponsorKeyShareId: selectedShareId,
          maxSessions:
            typeof maxSessions === "number" && maxSessions > 0
              ? maxSessions
              : null,
        });
      }
      closeModal();
    } catch {
      // Error handled by mutation
    }
  };

  const handleRevoke = async () => {
    try {
      await revokeShare.mutateAsync(gameId);
      closeModal();
    } catch {
      // Error handled by mutation
    }
  };

  const shareUrl =
    shareStatus?.token && typeof window !== "undefined"
      ? `${window.location.origin}/play/${shareStatus.token}`
      : "";

  const isSubmitting = enableShare.isPending || createGameShare.isPending;
  const canSubmit = isWorkshopMode || !!selectedShareId;

  const sourceLabel = isWorkshopMode
    ? t("games.privateShare.workshopShare")
    : t("games.privateShare.personalShare");

  const remainingLabel =
    shareStatus?.remaining != null
      ? t("games.privateShare.remainingSessions", {
          count: shareStatus.remaining,
        })
      : t("games.privateShare.unlimitedSessions");

  // Loading
  if (statusLoading) {
    return (
      <Text size="sm" c="dimmed">
        {t("games.privateShare.loading")}
      </Text>
    );
  }

  // Active share — inline display
  if (isEnabled) {
    return (
      <>
        <Stack gap="xs">
          <Group gap="xs">
            <Text size="xs" c="green" fw={500}>
              {sourceLabel}
            </Text>
            <Text size="xs" c="dimmed">
              — {remainingLabel}
            </Text>
          </Group>

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
                  {copied
                    ? t("copied")
                    : t("games.privateShare.copyLink")}
                </ActionButton>
              )}
            </CopyButton>
            <ActionButton
              onClick={handleOpenEdit}
              size="xs"
              leftSection={<IconEdit size={14} />}
            >
              {t("games.privateShare.edit")}
            </ActionButton>
            <DangerButton
              onClick={handleRevoke}
              loading={revokeShare.isPending}
              size="xs"
            >
              {t("games.privateShare.revoke")}
            </DangerButton>
          </Group>
        </Stack>

        <ShareFormModal
          opened={modalOpened}
          onClose={closeModal}
          mode={modalMode}
          isMobile={isMobile}
          isWorkshopMode={isWorkshopMode}
          keysLoading={keysLoading}
          selectData={selectData}
          selectedShareId={selectedShareId}
          onSelectShareId={setSelectedShareId}
          maxSessions={maxSessions}
          onMaxSessionsChange={setMaxSessions}
          canSubmit={canSubmit}
          isSubmitting={isSubmitting}
          onSubmit={handleSubmit}
          onRevoke={handleRevoke}
          isRevoking={revokeShare.isPending}
          t={t}
        />
      </>
    );
  }

  // No share — compact create button
  return (
    <>
      <div>
        <ActionButton
          onClick={handleOpenCreate}
          size="xs"
          leftSection={<IconLink size={14} />}
        >
          {t("games.privateShare.enable")}
        </ActionButton>
      </div>

      <ShareFormModal
        opened={modalOpened}
        onClose={closeModal}
        mode={modalMode}
        isMobile={isMobile}
        isWorkshopMode={isWorkshopMode}
        keysLoading={keysLoading}
        selectData={selectData}
        selectedShareId={selectedShareId}
        onSelectShareId={setSelectedShareId}
        maxSessions={maxSessions}
        onMaxSessionsChange={(v) => setMaxSessions(v)}
        canSubmit={canSubmit}
        isSubmitting={isSubmitting}
        onSubmit={handleSubmit}
        onRevoke={handleRevoke}
        isRevoking={revokeShare.isPending}
        t={t}
      />
    </>
  );
}

function ShareFormModal({
  opened,
  onClose,
  mode,
  isMobile,
  isWorkshopMode,
  keysLoading,
  selectData,
  selectedShareId,
  onSelectShareId,
  maxSessions,
  onMaxSessionsChange,
  canSubmit,
  isSubmitting,
  onSubmit,
  onRevoke,
  isRevoking,
  t,
}: {
  opened: boolean;
  onClose: () => void;
  mode: "create" | "edit";
  isMobile?: boolean;
  isWorkshopMode: boolean;
  keysLoading: boolean;
  selectData: { value: string; label: string }[];
  selectedShareId: string | null;
  onSelectShareId: (v: string | null) => void;
  maxSessions: number | string;
  onMaxSessionsChange: (v: number | string) => void;
  canSubmit: boolean;
  isSubmitting: boolean;
  onSubmit: () => void;
  onRevoke: () => void;
  isRevoking: boolean;
  t: (key: string, options?: Record<string, unknown>) => string;
}) {
  const isCreate = mode === "create";

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={
        isCreate
          ? t("games.privateShare.createModalTitle")
          : t("games.privateShare.editModalTitle")
      }
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
                onChange={onSelectShareId}
                searchable={selectData.length > 5}
              />
            )}
          </>
        )}

        {(isWorkshopMode || selectData.length > 0) && (
          <NumberInput
            label={t("games.privateShare.maxSessions")}
            description={t("games.privateShare.maxSessionsDescription")}
            placeholder={t("games.privateShare.unlimited")}
            value={maxSessions}
            onChange={(v) => onMaxSessionsChange(v === "" ? "" : v)}
            min={1}
            allowNegative={false}
            allowDecimal={false}
          />
        )}

        <Group justify="flex-end" gap="sm">
          {!isCreate && (
            <DangerButton
              onClick={onRevoke}
              loading={isRevoking}
              size="xs"
              style={{ marginRight: "auto" }}
            >
              {t("games.privateShare.revoke")}
            </DangerButton>
          )}
          <CancelButton onClick={onClose}>{t("cancel")}</CancelButton>
          <ActionButton
            onClick={onSubmit}
            disabled={!canSubmit}
            loading={isSubmitting}
            size="sm"
            leftSection={<IconLink size={16} />}
          >
            {isCreate
              ? t("games.privateShare.enable")
              : t("games.privateShare.update")}
          </ActionButton>
        </Group>
      </Stack>
    </Modal>
  );
}

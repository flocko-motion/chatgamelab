import {
  Stack,
  Group,
  Text,
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
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useState } from "react";
import { ActionButton, CancelButton, DangerButton } from "@components/buttons";
import { HelperText } from "@components/typography";
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
  const isWorkshopMode = !!workshopId;

  const { data: apiKeys, isLoading: keysLoading } = useApiKeys();
  const { data: shareStatus, isLoading: statusLoading } =
    usePrivateShareStatus(gameId);
  const enableShare = useEnablePrivateShare();
  const createGameShare = useCreateGameShare();
  const revokeShare = useRevokePrivateShare();

  const [selectedShareId, setSelectedShareId] = useState<string | null>(null);
  const [maxSessions, setMaxSessions] = useState<number | string>("");
  const [isCreating, setIsCreating] = useState(false);
  const [isEditing, setIsEditing] = useState(false);

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

  const handleEnable = async () => {
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
      setIsCreating(false);
      setIsEditing(false);
    } catch {
      // Error handled by mutation
    }
  };

  const handleRevoke = async () => {
    try {
      await revokeShare.mutateAsync(gameId);
      setIsEditing(false);
    } catch {
      // Error handled by mutation
    }
  };

  const handleStartEdit = () => {
    setSelectedShareId(null);
    setMaxSessions(shareStatus?.remaining ?? "");
    setIsEditing(true);
  };

  const handleStartCreate = () => {
    setSelectedShareId(null);
    setMaxSessions("");
    setIsCreating(true);
  };

  const shareUrl =
    shareStatus?.token && typeof window !== "undefined"
      ? `${window.location.origin}/play/${shareStatus.token}`
      : "";

  const isEnabling = enableShare.isPending || createGameShare.isPending;
  const canEnable = isWorkshopMode || !!selectedShareId;

  // Key selector (personal mode only)
  const keySelector = isWorkshopMode ? null : (
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
          searchable={selectData.length > 5}
        />
      )}
    </>
  );

  const sessionLimitField = (
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
  );

  const formFields = (
    <>
      {keySelector}
      {(isWorkshopMode || selectData.length > 0) && sessionLimitField}
    </>
  );

  // Loading
  if (statusLoading) {
    return (
      <Text size="sm" c="dimmed">
        {t("games.privateShare.loading")}
      </Text>
    );
  }

  // Active share — display mode
  if (isEnabled && !isEditing) {
    return (
      <Stack gap="sm">
        <Alert icon={<IconCheck size={16} />} color="green" variant="light">
          <Stack gap="xs">
            <Text size="sm">{t("games.privateShare.active")}</Text>
            {shareStatus?.remaining != null && (
              <HelperText>
                {t("games.privateShare.remainingSessions", {
                  count: shareStatus.remaining,
                })}
              </HelperText>
            )}
          </Stack>
        </Alert>

        {shareUrl && (
          <Group gap="xs" wrap="nowrap">
            <Text
              size="sm"
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
        )}

        <Group gap="sm" wrap="wrap">
          <CopyButton value={shareUrl}>
            {({ copied, copy }) => (
              <ActionButton
                onClick={copy}
                size="sm"
                leftSection={
                  copied ? <IconCheck size={16} /> : <IconCopy size={16} />
                }
              >
                {copied
                  ? t("copied")
                  : t("games.privateShare.copyLink")}
              </ActionButton>
            )}
          </CopyButton>
          <ActionButton
            onClick={handleStartEdit}
            size="sm"
            leftSection={<IconLink size={16} />}
          >
            {t("games.privateShare.edit")}
          </ActionButton>
          <DangerButton
            onClick={handleRevoke}
            loading={revokeShare.isPending}
          >
            {t("games.privateShare.revoke")}
          </DangerButton>
        </Group>
      </Stack>
    );
  }

  // Editing existing share
  if (isEditing) {
    return (
      <Stack gap="sm">
        <Text size="sm" fw={500} c="dimmed">
          {t("games.privateShare.editSettings")}
        </Text>
        {formFields}
        <Group gap="sm" wrap="wrap">
          <CancelButton onClick={() => setIsEditing(false)}>
            {t("cancel")}
          </CancelButton>
          <ActionButton
            onClick={handleEnable}
            disabled={!canEnable}
            loading={isEnabling}
            size="sm"
            leftSection={<IconLink size={16} />}
          >
            {t("games.privateShare.update")}
          </ActionButton>
          <DangerButton
            onClick={handleRevoke}
            loading={revokeShare.isPending}
          >
            {t("games.privateShare.revoke")}
          </DangerButton>
        </Group>
      </Stack>
    );
  }

  // Creating new share (expanded form)
  if (isCreating) {
    return (
      <Stack gap="sm">
        <Text size="sm" c="dimmed">
          {isWorkshopMode
            ? t("games.privateShare.workshopDescription")
            : t("games.privateShare.description")}
        </Text>
        {formFields}
        <Group gap="sm" wrap="wrap">
          <CancelButton onClick={() => setIsCreating(false)}>
            {t("cancel")}
          </CancelButton>
          <ActionButton
            onClick={handleEnable}
            disabled={!canEnable}
            loading={isEnabling}
            size="sm"
            leftSection={<IconLink size={16} />}
          >
            {t("games.privateShare.enable")}
          </ActionButton>
        </Group>
      </Stack>
    );
  }

  // Default: no share — show create button
  return (
    <ActionButton
      onClick={handleStartCreate}
      size="sm"
      leftSection={<IconLink size={16} />}
    >
      {t("games.privateShare.enable")}
    </ActionButton>
  );
}

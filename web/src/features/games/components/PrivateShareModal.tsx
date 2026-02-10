import {
  Modal,
  Stack,
  Group,
  Text,
  Select,
  Alert,
  NumberInput,
  CopyButton,
  ActionIcon,
  Tooltip,
  rem,
} from "@mantine/core";
import {
  IconAlertCircle,
  IconLink,
  IconCheck,
  IconCopy,
} from "@tabler/icons-react";
import { useMediaQuery } from "@mantine/hooks";
import { useTranslation } from "react-i18next";
import { useState } from "react";
import { ActionButton, CancelButton, DangerButton } from "@components/buttons";
import { HelperText } from "@components/typography";
import {
  useApiKeys,
  usePrivateShareStatus,
  useEnablePrivateShare,
  useRevokePrivateShare,
} from "@/api/hooks";
import type { ObjGame, ObjApiKeyShare } from "@/api/generated";

interface PrivateShareModalProps {
  game: ObjGame | null;
  opened: boolean;
  onClose: () => void;
}

export function PrivateShareModal({
  game,
  opened,
  onClose,
}: PrivateShareModalProps) {
  const { t } = useTranslation("common");
  const isMobile = useMediaQuery("(max-width: 48em)");
  const { data: apiKeys, isLoading: keysLoading } = useApiKeys();
  const { data: shareStatus, isLoading: statusLoading } = usePrivateShareStatus(
    game?.id,
  );
  const enableShare = useEnablePrivateShare();
  const revokeShare = useRevokePrivateShare();

  const [selectedShareId, setSelectedShareId] = useState<string | null>(null);
  const [maxSessions, setMaxSessions] = useState<number | string>("");

  const isEnabled = shareStatus?.enabled ?? false;

  const handleClose = () => {
    setSelectedShareId(null);
    setMaxSessions("");
    onClose();
  };

  // Build eligible key list (same logic as SponsorGameModal)
  const keys = apiKeys?.apiKeys ?? [];
  const shares = apiKeys?.shares ?? [];
  const eligibleKeys: (ObjApiKeyShare & {
    apiKey?: (typeof keys)[number];
  })[] = [];
  const seenKeyIds = new Set<string>();
  for (const share of shares) {
    if (!share.allowPublicGameSponsoring || !share.apiKeyId) continue;
    if (seenKeyIds.has(share.apiKeyId)) continue;
    const apiKey = keys.find((k) => k.id === share.apiKeyId);
    if (apiKey?.lastUsageSuccess === false) continue;
    seenKeyIds.add(share.apiKeyId);
    eligibleKeys.push({ ...share, apiKey });
  }

  const selectData = eligibleKeys.map((share) => ({
    value: share.id!,
    label: `${share.apiKey?.name ?? "Unknown"} (${share.apiKey?.platform ?? "?"})`,
  }));

  const handleEnable = async () => {
    if (!game?.id || !selectedShareId) return;
    try {
      await enableShare.mutateAsync({
        gameId: game.id,
        sponsorKeyShareId: selectedShareId,
        maxSessions:
          typeof maxSessions === "number" && maxSessions > 0
            ? maxSessions
            : null,
      });
    } catch {
      // Error handled by mutation
    }
  };

  const handleRevoke = async () => {
    if (!game?.id) return;
    try {
      await revokeShare.mutateAsync(game.id);
    } catch {
      // Error handled by mutation
    }
  };

  const shareUrl =
    shareStatus?.token && typeof window !== "undefined"
      ? `${window.location.origin}/play/${shareStatus.token}`
      : "";

  return (
    <Modal
      opened={opened}
      onClose={handleClose}
      title={t("games.privateShare.title")}
      size={isMobile ? "100%" : rem(500)}
      fullScreen={isMobile}
      centered={!isMobile}
    >
      <Stack gap="md">
        {statusLoading ? (
          <Text size="sm" c="dimmed">
            {t("games.privateShare.loading")}
          </Text>
        ) : isEnabled ? (
          <>
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

            <Group justify="flex-end" mt="md" gap="sm">
              <CancelButton onClick={handleClose}>{t("close")}</CancelButton>
              <DangerButton
                onClick={handleRevoke}
                loading={revokeShare.isPending}
              >
                {t("games.privateShare.revoke")}
              </DangerButton>
            </Group>
          </>
        ) : (
          <>
            <Text size="sm" c="dimmed">
              {t("games.privateShare.description")}
            </Text>

            {keysLoading ? (
              <Text size="sm" c="dimmed">
                {t("games.privateShare.loadingKeys")}
              </Text>
            ) : eligibleKeys.length === 0 ? (
              <Alert
                icon={<IconAlertCircle size={16} />}
                color="yellow"
                variant="light"
              >
                {t("games.privateShare.noEligibleKeys")}
              </Alert>
            ) : (
              <>
                <Select
                  label={t("games.privateShare.selectKey")}
                  placeholder={t("games.privateShare.selectKeyPlaceholder")}
                  data={selectData}
                  value={selectedShareId}
                  onChange={setSelectedShareId}
                  searchable={selectData.length > 5}
                />

                <NumberInput
                  label={t("games.privateShare.maxSessions")}
                  description={t("games.privateShare.maxSessionsDescription")}
                  placeholder={t("games.privateShare.unlimited")}
                  value={maxSessions}
                  onChange={setMaxSessions}
                  min={1}
                  allowNegative={false}
                  allowDecimal={false}
                />
              </>
            )}

            <Group justify="flex-end" mt="md" gap="sm">
              <CancelButton onClick={handleClose}>{t("cancel")}</CancelButton>
              <ActionButton
                onClick={handleEnable}
                disabled={!selectedShareId}
                loading={enableShare.isPending}
                size="sm"
                leftSection={<IconLink size={16} />}
              >
                {t("games.privateShare.enable")}
              </ActionButton>
            </Group>
          </>
        )}
      </Stack>
    </Modal>
  );
}

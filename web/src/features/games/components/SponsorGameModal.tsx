import {
  Modal,
  Stack,
  Group,
  Text,
  Select,
  Alert,
  Badge,
  rem,
} from "@mantine/core";
import { IconAlertCircle, IconCoin, IconCheck } from "@tabler/icons-react";
import { useMediaQuery } from "@mantine/hooks";
import { useTranslation } from "react-i18next";
import { useState } from "react";
import { ActionButton, CancelButton, DangerButton } from "@components/buttons";
import { useApiKeys, useSponsorGame, useRemoveGameSponsor } from "@/api/hooks";
import type { ObjGame, ObjApiKeyShare } from "@/api/generated";

interface SponsorGameModalProps {
  game: ObjGame | null;
  opened: boolean;
  onClose: () => void;
}

export function SponsorGameModal({
  game,
  opened,
  onClose,
}: SponsorGameModalProps) {
  const { t } = useTranslation("common");
  const isMobile = useMediaQuery("(max-width: 48em)");
  const { data: apiKeys, isLoading: keysLoading } = useApiKeys();
  const sponsorGame = useSponsorGame();
  const removeSponsor = useRemoveGameSponsor();

  const [selectedShareId, setSelectedShareId] = useState<string | null>(null);
  const [step, setStep] = useState<"select" | "confirm">("select");

  const isSponsored = !!game?.publicSponsoredApiKeyShareId;

  // Filter to only show shares that allow public sponsoring, enriched with key info
  const keys = apiKeys?.apiKeys ?? [];
  const shares = apiKeys?.shares ?? [];
  const eligibleKeys: (ObjApiKeyShare & { apiKey?: (typeof keys)[number] })[] =
    shares
      .filter((share) => share.allowPublicGameSponsoring)
      .map((share) => ({
        ...share,
        apiKey: keys.find((k) => k.id === share.apiKeyId),
      }))
      .filter((share) => share.apiKey?.lastUsageSuccess !== false);

  const selectedShare = eligibleKeys.find((s) => s.id === selectedShareId);

  const handleClose = () => {
    setSelectedShareId(null);
    setStep("select");
    onClose();
  };

  const handleNext = () => {
    if (selectedShareId) {
      setStep("confirm");
    }
  };

  const handleBack = () => {
    setStep("select");
  };

  const handleConfirm = async () => {
    if (!game?.id || !selectedShareId) return;
    try {
      await sponsorGame.mutateAsync({
        gameId: game.id,
        shareId: selectedShareId,
      });
      handleClose();
    } catch {
      // Error handled by mutation
    }
  };

  const handleRemoveSponsor = async () => {
    if (!game?.id) return;
    try {
      await removeSponsor.mutateAsync(game.id);
      handleClose();
    } catch {
      // Error handled by mutation
    }
  };

  const selectData = eligibleKeys.map((share) => ({
    value: share.id!,
    label: `${share.apiKey?.name ?? "Unknown"} (${share.apiKey?.platform ?? "?"})`,
  }));

  return (
    <Modal
      opened={opened}
      onClose={handleClose}
      title={t("games.sponsor.title")}
      size={isMobile ? "100%" : rem(500)}
      fullScreen={isMobile}
      centered={!isMobile}
    >
      <Stack gap="md">
        {isSponsored && step === "select" && (
          <Alert icon={<IconCheck size={16} />} color="green" variant="light">
            <Stack gap="xs">
              <Text size="sm">{t("games.sponsor.currentlySponsored")}</Text>
              <DangerButton
                onClick={handleRemoveSponsor}
                loading={removeSponsor.isPending}
              >
                {t("games.sponsor.removeSponsor")}
              </DangerButton>
            </Stack>
          </Alert>
        )}

        {step === "select" && (
          <>
            <Text size="sm" c="dimmed">
              {t("games.sponsor.description")}
            </Text>

            {keysLoading ? (
              <Text size="sm" c="dimmed">
                {t("games.sponsor.loadingKeys")}
              </Text>
            ) : eligibleKeys.length === 0 ? (
              <Alert
                icon={<IconAlertCircle size={16} />}
                color="yellow"
                variant="light"
              >
                {t("games.sponsor.noEligibleKeys")}
              </Alert>
            ) : (
              <Select
                label={t("games.sponsor.selectKey")}
                placeholder={t("games.sponsor.selectKeyPlaceholder")}
                data={selectData}
                value={selectedShareId}
                onChange={setSelectedShareId}
                searchable={selectData.length > 5}
              />
            )}

            <Group justify="flex-end" mt="md" gap="sm">
              <CancelButton onClick={handleClose}>{t("cancel")}</CancelButton>
              <ActionButton
                onClick={handleNext}
                disabled={!selectedShareId}
                size="sm"
                leftSection={<IconCoin size={16} />}
              >
                {t("games.sponsor.next")}
              </ActionButton>
            </Group>
          </>
        )}

        {step === "confirm" && selectedShare && (
          <>
            <Alert
              icon={<IconAlertCircle size={16} />}
              color="blue"
              variant="light"
            >
              {t("games.sponsor.confirmExplanation")}
            </Alert>

            <Stack gap="xs">
              <Group gap="xs">
                <Text size="sm" fw={600}>
                  {t("games.sponsor.gameName")}
                </Text>
                <Text size="sm">{game?.name}</Text>
              </Group>
              <Group gap="xs">
                <Text size="sm" fw={600}>
                  {t("games.sponsor.apiKey")}
                </Text>
                <Text size="sm">{selectedShare.apiKey?.name}</Text>
                <Badge size="sm" variant="light">
                  {selectedShare.apiKey?.platform}
                </Badge>
              </Group>
            </Stack>

            <Text size="sm" c="dimmed">
              {t("games.sponsor.confirmNote")}
            </Text>

            <Group justify="flex-end" mt="md" gap="sm">
              <CancelButton onClick={handleBack}>
                {t("games.sponsor.back")}
              </CancelButton>
              <ActionButton
                onClick={handleConfirm}
                loading={sponsorGame.isPending}
                size="sm"
                leftSection={<IconCoin size={16} />}
              >
                {t("games.sponsor.confirm")}
              </ActionButton>
            </Group>
          </>
        )}
      </Stack>
    </Modal>
  );
}

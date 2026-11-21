import { Badge, Divider, Group, Stack, Switch, Text } from "@mantine/core";
import { IconHeartFilled, IconShare } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { ActionButton } from "@components/buttons";
import { PrivateShareSection } from "./PrivateShareSection";

/** The editor's sharing and visibility: public switch, share link, sponsoring. */
export function SharingSection({
  game,
  gameId,
  isPublic,
  setIsPublic,
  mayEditCard,
  mayPublish,
  wasPublic,
  mayShare,
  onSponsor,
  showShareSection,
  workshopId,
}: {
  game?: {
    publicSponsoredApiKeyShareId?: string;
    workshopId?: string;
  } | null;
  gameId?: string;
  isPublic: boolean;
  setIsPublic: (v: boolean) => void;
  /** Making the game private again is a card change. */
  mayEditCard: boolean;
  /** The caller may make the game public (the server's flag). */
  mayPublish: boolean;
  /** The game is public as loaded. */
  wasPublic: boolean;
  /** The caller may share the game where they are now (the server's flag). */
  mayShare: boolean;
  onSponsor?: () => void;
  showShareSection?: boolean;
  workshopId?: string;
}) {
  const { t } = useTranslation("common");

  const hasSponsoring = !!game?.publicSponsoredApiKeyShareId;

  const canShowShareInContext = showShareSection && mayShare;

  // Switching on needs mayPublish, switching a public game off needs
  // mayEditCard; taking back an unsaved switch is always possible.
  const canSwitch = isPublic ? mayEditCard || !wasPublic : mayPublish;

  // Only show public sponsoring when game is public
  const canShowSponsoring = onSponsor && isPublic;

  return (
    <>
      <Divider my="xs" />
      <Group gap="xs" align="center">
        <IconShare size={22} />
        <Text size="lg" fw={700}>
          {t("games.sharing.sectionTitle")}
        </Text>
      </Group>

      <Stack gap="lg">
        <Stack gap={4}>
          <Switch
            label={t("games.createModal.publicLabel")}
            checked={isPublic}
            onChange={(e) => setIsPublic(e.currentTarget.checked)}
            disabled={!canSwitch}
          />
          <Text size="sm" c="dimmed">
            {t("games.createModal.publicDescription")}
          </Text>
          {mayPublish && !isPublic && (
            <Text size="xs" c="orange.8">
              {t("games.privacyHint")}
            </Text>
          )}
        </Stack>
        {!isPublic && game?.publicSponsoredApiKeyShareId && (
          <Text size="xs" c="orange" fw={500}>
            {t("games.sponsor.privateWarning")}
          </Text>
        )}

        {canShowShareInContext && (
          <Stack gap="xs">
            <Divider />
            <Text size="md" fw={600}>
              {t("games.sharing.privateShareLink")}
            </Text>
            <Text size="sm" c="dimmed">
              {t("games.sharing.privateShareLinkDescription")}
            </Text>
            <PrivateShareSection gameId={gameId!} workshopId={workshopId} />
          </Stack>
        )}

        {canShowSponsoring && (
          <Stack gap="xs">
            <Divider />
            <Text size="md" fw={600}>
              {t("games.sharing.publicSponsoring")}
            </Text>
            <Text size="sm" c="dimmed">
              {t("games.sharing.publicSponsoringDescription")}
            </Text>
            <Group gap="sm" align="center">
              <ActionButton
                onClick={onSponsor}
                size="xs"
                leftSection={<IconHeartFilled size={16} />}
              >
                {hasSponsoring
                  ? t("games.sponsor.manageSponsor")
                  : t("games.sponsor.sponsorGame")}
              </ActionButton>
              {hasSponsoring && (
                <Badge
                  size="xs"
                  color="pink"
                  variant="light"
                  leftSection={<IconHeartFilled size={10} />}
                >
                  {t("games.sponsor.sponsored")}
                </Badge>
              )}
            </Group>
          </Stack>
        )}
      </Stack>
    </>
  );
}

import { Badge, Divider, Group, Stack, Switch } from "@mantine/core";
import { IconHeartFilled } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { GameView } from "@/api/types";
import { ActionButton } from "@components/buttons";
import { CardTitle, HelperText } from "@components/typography";
import type { GameEditor } from "../../hooks/useGameEditor";
import { PrivateShareSection } from "../PrivateShareSection";
import { DownloadSection } from "./DownloadSection";
import { EmbedSection } from "./EmbedSection";

interface SharingSectionProps {
  editor: GameEditor;
  game: GameView;
  /** Whether the list the editor opened from offers share links. */
  showShareLinks: boolean;
  workshopId?: string;
  onSponsor?: () => void;
}

/**
 * Sharing, for a saved game: public, share links and sponsoring, embedding,
 * and the game as a file. It shows what the server allows the caller: making
 * public by mayPublish, share links and embedding by mayShare; the download
 * always.
 */
export function SharingSection({ editor, game, showShareLinks, workshopId, onSponsor }: SharingSectionProps) {
  const { t } = useTranslation("common");
  const isPublic = editor.values.public;
  const sponsored = !!game.publicSponsoredApiKeyShareId;

  return (
    <Stack gap="lg">
      {editor.publicSwitch.shown && (
        <Stack gap={4}>
          <Switch
            label={t("games.createModal.publicLabel")}
            checked={isPublic}
            onChange={(e) => editor.set("public", e.currentTarget.checked)}
            disabled={!editor.publicSwitch.enabled}
          />
          <HelperText>{t("games.createModal.publicDescription")}</HelperText>
          {game.mayPublish && !isPublic && <HelperText c="orange.8">{t("games.privacyHint")}</HelperText>}
          {!isPublic && sponsored && <HelperText c="orange">{t("games.sponsor.privateWarning")}</HelperText>}
        </Stack>
      )}

      {showShareLinks && game.mayShare && (
        <Stack gap="xs">
          <Divider />
          <CardTitle>{t("games.sharing.privateShareLink")}</CardTitle>
          <HelperText>{t("games.sharing.privateShareLinkDescription")}</HelperText>
          <PrivateShareSection gameId={game.id} workshopId={workshopId} />
        </Stack>
      )}

      {onSponsor && isPublic && (
        <Stack gap="xs">
          <Divider />
          <CardTitle>{t("games.sharing.publicSponsoring")}</CardTitle>
          <HelperText>{t("games.sharing.publicSponsoringDescription")}</HelperText>
          <Group gap="sm" align="center">
            <ActionButton onClick={onSponsor} size="xs" leftSection={<IconHeartFilled size={16} />}>
              {sponsored ? t("games.sponsor.manageSponsor") : t("games.sponsor.sponsorGame")}
            </ActionButton>
            {sponsored && (
              <Badge size="xs" color="pink" variant="light" leftSection={<IconHeartFilled size={10} />}>
                {t("games.sponsor.sponsored")}
              </Badge>
            )}
          </Group>
        </Stack>
      )}

      <EmbedSection game={game} />

      <Divider />
      <DownloadSection game={game} />
    </Stack>
  );
}

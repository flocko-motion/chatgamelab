import { Badge, Divider, Group, Stack, Switch } from "@mantine/core";
import { IconHeartFilled } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { GameView } from "@/api/types";
import { ActionButton } from "@components/buttons";
import { HelperText } from "@components/typography";
import type { GameEditor } from "../../hooks/useGameEditor";
import { useEmbedPanel } from "../../hooks/useEmbedPanel";
import { PrivateShareSection } from "../PrivateShareSection";
import { DownloadSection } from "./DownloadSection";
import { EmbedSection } from "./EmbedSection";
import { SharingBlock } from "./SharingBlock";

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
 * always. Each is one block, headed the same way, with a line between them.
 */
export function SharingSection({ editor, game, showShareLinks, workshopId, onSponsor }: SharingSectionProps) {
  const { t } = useTranslation("common");
  const isPublic = editor.values.public;
  const sponsored = !!game.publicSponsoredApiKeyShareId;
  const embed = useEmbedPanel(game);

  const blocks = [
    editor.publicSwitch.shown && (
      <SharingBlock
        key="public"
        title={t("games.createModal.publicLabel")}
        description={t("games.createModal.publicDescription")}
        control={
          <Switch
            aria-label={t("games.createModal.publicLabel")}
            checked={isPublic}
            onChange={(e) => editor.set("public", e.currentTarget.checked)}
            disabled={!editor.publicSwitch.enabled}
          />
        }
      >
        {game.mayPublish && !isPublic && <HelperText c="orange.8">{t("games.privacyHint")}</HelperText>}
        {!isPublic && sponsored && <HelperText c="orange">{t("games.sponsor.privateWarning")}</HelperText>}
      </SharingBlock>
    ),

    showShareLinks && game.mayShare && (
      <SharingBlock
        key="links"
        title={t("games.sharing.privateShareLink")}
        description={t("games.sharing.privateShareLinkDescription")}
      >
        <PrivateShareSection gameId={game.id} workshopId={workshopId} />
      </SharingBlock>
    ),

    onSponsor && isPublic && (
      <SharingBlock
        key="sponsor"
        title={t("games.sharing.publicSponsoring")}
        description={t("games.sharing.publicSponsoringDescription")}
      >
        <Group gap="sm" align="center">
          <ActionButton onClick={onSponsor} size="sm" leftSection={<IconHeartFilled size={16} />}>
            {sponsored ? t("games.sponsor.manageSponsor") : t("games.sponsor.sponsorGame")}
          </ActionButton>
          {sponsored && (
            <Badge size="xs" color="pink" variant="light" leftSection={<IconHeartFilled size={10} />}>
              {t("games.sponsor.sponsored")}
            </Badge>
          )}
        </Group>
      </SharingBlock>
    ),

    embed.shown && <EmbedSection key="embed" panel={embed} />,

    <DownloadSection key="download" game={game} />,
  ].filter(Boolean);

  return (
    <Stack gap="lg">
      {blocks.flatMap((block, i) => (i === 0 ? [block] : [<Divider key={`divider-${i}`} />, block]))}
    </Stack>
  );
}

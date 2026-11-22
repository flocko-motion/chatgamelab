import { Alert, Group, Stack, TextInput } from "@mantine/core";
import { IconAlertCircle, IconPhoto, IconRefresh, IconSparkles } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { ActionButton } from "@components/buttons";
import { HelperText } from "@components/typography";
import { GameCover } from "@components/GameCover";
import type { GameView } from "@/api/types";
import { COVER_HINT_MAX, useGameCover } from "../hooks";

interface GameCoverSectionProps {
  /** The game as saved; a new game has none yet. */
  game: GameView | undefined;
  /** The caller may change the card, which holds the cover. */
  mayEdit: boolean;
}

/** The editor's cover: the picture, and drawing a new one with a hint. */
export function GameCoverSection({ game, mayEdit }: GameCoverSectionProps) {
  const { t } = useTranslation("common");
  const cover = useGameCover(game?.id);

  if (!game) {
    return (
      <Group gap="xs" align="center" wrap="nowrap">
        <IconPhoto size={18} aria-hidden />
        <HelperText>{t("games.cover.drawnOnSave")}</HelperText>
      </Group>
    );
  }

  return (
    <Stack gap="sm">
      <Group align="flex-start" gap="md" wrap="wrap">
        <div style={{ flex: "1 1 220px", maxWidth: 360 }}>
          <GameCover game={game} busy={cover.drawing} />
        </div>
        {mayEdit && (
          <Stack gap="sm" style={{ flex: "1 1 260px" }}>
            <TextInput
              label={t("games.cover.hint")}
              description={t("games.cover.hintDescription")}
              placeholder={t("games.cover.hintPlaceholder")}
              value={cover.hint}
              onChange={(e) => cover.setHint(e.currentTarget.value)}
              maxLength={COVER_HINT_MAX}
              disabled={cover.drawing}
            />
            <Group>
              <ActionButton
                size="sm"
                onClick={cover.draw}
                loading={cover.drawing}
                leftSection={game.coverAt ? <IconRefresh size={16} /> : <IconSparkles size={16} />}
              >
                {game.coverAt ? t("games.cover.drawAgain") : t("games.cover.draw")}
              </ActionButton>
            </Group>
          </Stack>
        )}
      </Group>
      {cover.errorKey && (
        <Alert icon={<IconAlertCircle size={16} />} color="orange" variant="light">
          {t(cover.errorKey)}
        </Alert>
      )}
    </Stack>
  );
}

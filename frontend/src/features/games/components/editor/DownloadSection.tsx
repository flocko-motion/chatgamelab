import { Group, Stack } from "@mantine/core";
import { IconDownload } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { GameView } from "@/api/types";
import { GenericButtonWithText } from "@components/buttons";
import { CardTitle, HelperText } from "@components/typography";
import { useDownloadGame } from "../../hooks/useDownloadGame";

/** The game as a file: to keep, or to import into another ChatGameLab server. */
export function DownloadSection({ game }: { game: GameView }) {
  const { t } = useTranslation("common");
  const download = useDownloadGame();
  return (
    <Group justify="space-between" wrap="nowrap" align="flex-start">
      <Stack gap={2} style={{ flex: 1, minWidth: 0 }}>
        <CardTitle>{t("games.editor.download.title")}</CardTitle>
        <HelperText>{t("games.editor.download.description")}</HelperText>
      </Stack>
      <GenericButtonWithText
        icon={<IconDownload size={16} />}
        variant="default"
        size="sm"
        style={{ flexShrink: 0 }}
        loading={download.downloading}
        onClick={() => download.run(game)}
      >
        {t("games.editor.download.button")}
      </GenericButtonWithText>
    </Group>
  );
}

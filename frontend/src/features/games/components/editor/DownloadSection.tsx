import { IconDownload } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { GameView } from "@/api/types";
import { GenericButtonWithText } from "@components/buttons";
import { useDownloadGame } from "../../hooks/useDownloadGame";
import { SharingBlock } from "./SharingBlock";

/** The game as a file: to keep, or to import into another ChatGameLab server. */
export function DownloadSection({ game }: { game: GameView }) {
  const { t } = useTranslation("common");
  const download = useDownloadGame();
  return (
    <SharingBlock
      title={t("games.editor.download.title")}
      description={t("games.editor.download.description")}
      control={
        <GenericButtonWithText
          icon={<IconDownload size={16} />}
          variant="default"
          size="sm"
          loading={download.downloading}
          onClick={() => download.run(game)}
        >
          {t("games.editor.download.button")}
        </GenericButtonWithText>
      }
    />
  );
}

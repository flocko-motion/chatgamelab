import { Stack } from "@mantine/core";
import { useTranslation } from "react-i18next";
import { PageTitle } from "@components/typography";
import { GamesTable } from "./GamesTable";

interface AllGamesProps {
  /** Open this game's modal on mount (/games/<id>). */
  initialGameId?: string;
  /** Open the create modal on mount (/games/create). */
  initialCreate?: boolean;
  /** What the new game is about, filled in on create (/games/create?about=…). */
  initialAbout?: string;
  /** Open the import file dialog on mount (/games?action=import). */
  autoImport?: boolean;
  /** Called when a modal opened on mount is closed. */
  onModalClose?: () => void;
}

/** Every game the caller can see, narrowed by toggles (own, sponsored, …). */
export function AllGames({ initialGameId, initialCreate, initialAbout, autoImport, onModalClose }: AllGamesProps = {}) {
  const { t } = useTranslation("common");

  return (
    <Stack gap="lg">
      <PageTitle>{t("allGames.title")}</PageTitle>
      <GamesTable
        preset="all"
        initialGameId={initialGameId}
        initialCreate={initialCreate}
        initialAbout={initialAbout}
        autoImport={autoImport}
        onModalClose={onModalClose}
      />
    </Stack>
  );
}

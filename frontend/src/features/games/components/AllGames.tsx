import { Button, Group, Stack } from "@mantine/core";
import { IconFlag } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { PageTitle } from "@components/typography";
import { HELP_LINKS } from "@/config/helpLinks";
import { GamesTable } from "./GamesTable";

interface AllGamesProps {
  /** Open this game's modal on mount (/games/<id>). */
  initialGameId?: string;
  /** Open the create modal on mount (/games/create). */
  initialCreate?: boolean;
  /** Open the import file dialog on mount (/games?action=import). */
  autoImport?: boolean;
  /** Called when a modal opened on mount is closed. */
  onModalClose?: () => void;
}

/** Every game the caller can see, narrowed by toggles (own, sponsored, …). */
export function AllGames({ initialGameId, initialCreate, autoImport, onModalClose }: AllGamesProps = {}) {
  const { t } = useTranslation("common");

  return (
    <Stack gap="lg">
      <Group justify="space-between" align="center">
        <PageTitle>{t("allGames.title")}</PageTitle>
        <Button
          component="a"
          href={HELP_LINKS.REPORT_GAME}
          target="_blank"
          rel="noopener noreferrer"
          variant="light"
          color="gray"
          size="xs"
          leftSection={<IconFlag size={14} color="var(--mantine-color-red-6)" />}
        >
          {t("allGames.reportGame")}
        </Button>
      </Group>
      <GamesTable
        preset="all"
        initialGameId={initialGameId}
        initialCreate={initialCreate}
        autoImport={autoImport}
        onModalClose={onModalClose}
      />
    </Stack>
  );
}

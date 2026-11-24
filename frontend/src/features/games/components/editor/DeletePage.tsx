import { Alert, Group, Paper, Stack } from "@mantine/core";
import { IconAlertTriangle, IconTrash } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { GameView } from "@/api/types";
import { GameCover } from "@components/GameCover";
import { DangerButton } from "@components/buttons";
import { BodyText, HelperText } from "@components/typography";
import { DownloadSection } from "./DownloadSection";
import classes from "./GameEditor.module.css";

interface DeletePageProps {
  game: GameView;
  name: string;
  deleting: boolean;
  onDelete: () => void;
  /** Someone else's game of the caller's workshop (a fact, for the warning). */
  othersWorkshopGame: boolean;
  /** An admin deleting a game that is not theirs (a fact, for the warning). */
  adminAction: boolean;
}

/**
 * Deleting is a page of its own: the game as it fades, what goes for good, a
 * way to keep a copy, and the button in the middle of the page, away from
 * the corner where Done and Save sit. The safe way out, Keep it, is the
 * footer's.
 */
export function DeletePage({ game, name, deleting, onDelete, othersWorkshopGame, adminAction }: DeletePageProps) {
  const { t } = useTranslation("common");
  return (
    <Stack gap="lg">
      {othersWorkshopGame && (
        <Alert color="red" icon={<IconAlertTriangle size={16} />} title={t("games.deleteModal.moderatorActionTitle")}>
          {t("games.deleteModal.moderatorActionWarning")}
        </Alert>
      )}
      {adminAction && !othersWorkshopGame && (
        <Alert color="red" icon={<IconAlertTriangle size={16} />} title={t("games.deleteModal.adminActionTitle")}>
          {t("games.deleteModal.adminActionWarning")}
        </Alert>
      )}
      <div className={classes.fading}>
        <GameCover game={game} />
      </div>
      <BodyText>{t("games.editor.delete.whatGoes", { name })}</BodyText>
      <Paper withBorder radius="md" p="md">
        <DownloadSection game={game} />
      </Paper>
      <HelperText>{t("games.editor.delete.keepFile")}</HelperText>
      <Group justify="center" mt="md">
        <DangerButton onClick={onDelete} loading={deleting} leftSection={<IconTrash size={18} />}>
          {t("games.editor.delete.confirm")}
        </DangerButton>
      </Group>
    </Stack>
  );
}

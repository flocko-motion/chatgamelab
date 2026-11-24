import type { ReactNode } from "react";
import { Alert, Badge, Group, Paper, SimpleGrid, Stack, ThemeIcon, UnstyledButton } from "@mantine/core";
import {
  IconAlertCircle,
  IconChevronRight,
  IconCircleCheckFilled,
  IconCircleDashed,
  IconTrash,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { GameCover } from "@components/GameCover";
import { CardTitle, HelperText, Label, SectionTitle } from "@components/typography";
import { pageSummary, type EditorPage } from "../../lib/editorPages";
import type { GameEditor } from "../../hooks/useGameEditor";
import type { EditorNavigation } from "../../hooks/useEditorNavigation";
import type { VoicePreview } from "../../hooks/useVoicePreview";
import { PAGE_ICON } from "./pageIcons";
import { VoiceTile } from "./VoiceTile";
import classes from "./GameEditor.module.css";

interface EditorOverviewProps {
  editor: GameEditor;
  nav: EditorNavigation;
  preview: VoicePreview;
  phone: boolean;
}

/**
 * The editor's start: one entry per section, each saying what it holds and
 * whether it is done, opened in any order. On a desktop a grid of cards,
 * with the cover and the voice to play; on a phone a compact list, like the
 * phone's own settings. Deleting is the last entry, plain until it is open.
 */
export function EditorOverview({ editor, nav, preview, phone }: EditorOverviewProps) {
  const { t } = useTranslation("common");
  const summary = (page: EditorPage) => {
    const s = pageSummary(page, editor.values);
    return "text" in s ? s.text : t(s.key);
  };
  const label = (page: EditorPage) => t(`games.editor.pages.${page}.label`);

  let hint: ReactNode = null;
  if (editor.readOnly) {
    hint = (
      <Alert icon={<IconAlertCircle size={16} />} color="blue" variant="light">
        {t("games.editor.othersGame")}
      </Alert>
    );
  } else if (editor.creating) {
    hint = <HelperText>{nav.allDone ? t("games.editor.allSet") : t("games.editor.fillIn")}</HelperText>;
  }

  return (
    <Stack gap="md">
      <Stack gap={4}>
        <SectionTitle>{editor.values.name.trim() || t("games.editor.untitled")}</SectionTitle>
        <Group gap="xs">
          <Badge variant="light" size="lg">
            {t(`games.genres.${editor.values.genre}`, { defaultValue: editor.values.genre })}
          </Badge>
        </Group>
      </Stack>
      {hint}

      {phone ? (
        <Paper withBorder radius="lg" className={classes.list}>
          {nav.pages.map((page) => (
            <Row
              key={page}
              icon={PAGE_ICON[page]}
              label={label(page)}
              summary={summary(page)}
              done={nav.done(page)}
              next={page === nav.next}
              onOpen={() => nav.open(page)}
            />
          ))}
          {nav.mayDelete && (
            <Row
              icon={<IconTrash size={18} />}
              plain
              label={t("games.editor.delete.label")}
              summary={t("games.editor.delete.intro")}
              onOpen={nav.openDelete}
            />
          )}
        </Paper>
      ) : (
        <SimpleGrid cols={2} spacing="md">
          {nav.pages.map((page) => (
            <Card
              key={page}
              icon={PAGE_ICON[page]}
              label={label(page)}
              summary={summary(page)}
              done={nav.done(page)}
              next={page === nav.next}
              onOpen={() => nav.open(page)}
            >
              {page === "look" && editor.game && (
                <div className={classes.thumb}>
                  <GameCover game={editor.game} />
                </div>
              )}
              {page === "sound" && (
                <div className={classes.thumb}>
                  <VoiceTile preview={preview} style={editor.values.voiceStyle.trim()} />
                </div>
              )}
            </Card>
          ))}
          {nav.mayDelete && (
            <Card
              icon={<IconTrash size={18} />}
              plain
              label={t("games.editor.delete.label")}
              summary={t("games.editor.delete.intro")}
              onOpen={nav.openDelete}
            />
          )}
        </SimpleGrid>
      )}
    </Stack>
  );
}

interface EntryProps {
  icon: ReactNode;
  label: string;
  summary: string;
  /** The section's green check; undefined for an entry without one. */
  done?: boolean;
  next?: boolean;
  /** Grey, with no check: the delete entry. */
  plain?: boolean;
  onOpen: () => void;
  children?: ReactNode;
}

function Check({ done, size }: { done: boolean; size: number }) {
  const { t } = useTranslation("common");
  return done ? (
    <IconCircleCheckFilled size={size} color="var(--mantine-color-green-6)" aria-label={t("games.editor.done")} />
  ) : (
    <IconCircleDashed size={size} color="var(--mantine-color-gray-5)" aria-label={t("games.editor.notDone")} />
  );
}

function Card({ icon, label, summary, done, next, plain, onOpen, children }: EntryProps) {
  const { t } = useTranslation("common");
  return (
    <UnstyledButton component="div" role="button" tabIndex={0} onClick={onOpen} onKeyDown={(e) => e.key === "Enter" && onOpen()}>
      <Paper withBorder radius="lg" p="lg" className={`${classes.card} ${next ? classes.cardNext : ""}`}>
        <Group wrap="nowrap" align="flex-start" gap="md">
          <ThemeIcon size={44} radius="md" variant="light" color={plain ? "gray" : undefined}>
            {icon}
          </ThemeIcon>
          <Stack gap={4} style={{ flex: 1, minWidth: 0 }}>
            <Group gap="xs">
              <CardTitle>{label}</CardTitle>
              {!plain && done !== undefined && <Check done={done} size={16} />}
              {next && <Badge size="xs">{t("games.editor.next")}</Badge>}
            </Group>
            <HelperText lineClamp={2}>{summary}</HelperText>
            {children}
          </Stack>
          <IconChevronRight size={18} color="var(--mantine-color-gray-5)" aria-hidden />
        </Group>
      </Paper>
    </UnstyledButton>
  );
}

function Row({ icon, label, summary, done, next, plain, onOpen }: EntryProps) {
  return (
    <UnstyledButton
      component="div"
      role="button"
      tabIndex={0}
      onClick={onOpen}
      onKeyDown={(e) => e.key === "Enter" && onOpen()}
      className={`${classes.row} ${next ? classes.rowNext : ""}`}
    >
      <Group wrap="nowrap" gap="sm">
        <ThemeIcon size={32} radius="md" variant="light" color={plain ? "gray" : undefined}>
          {icon}
        </ThemeIcon>
        <Stack gap={0} style={{ flex: 1, minWidth: 0 }}>
          <Label>{label}</Label>
          <HelperText size="xs" truncate>
            {summary}
          </HelperText>
        </Stack>
        {!plain && done !== undefined && <Check done={done} size={20} />}
        <IconChevronRight size={18} color="var(--mantine-color-gray-5)" aria-hidden />
      </Group>
    </UnstyledButton>
  );
}

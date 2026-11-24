import { Badge, Center, Loader, Paper, SimpleGrid, Stack, ThemeIcon, UnstyledButton } from "@mantine/core";
import { IconCompass, IconMessageCircle, IconSparkles, IconUsersGroup } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { CardTitle, HelperText, SectionTitle } from "@components/typography";
import { genreWording } from "../../lib/editorPages";
import { useGenreChoices } from "../../hooks/useEditorChoices";
import classes from "./GameEditor.module.css";

const GENRE_ICON: Record<string, React.ReactNode> = {
  adventure: <IconCompass size={36} />,
  npc: <IconMessageCircle size={36} />,
  discussion: <IconUsersGroup size={36} />,
};

/**
 * The first screen of a new game: one question, what to make, with a big
 * card per genre the engine plays. Picking one is the whole answer. It shows
 * once the form has its template, so a genre picked is never overwritten.
 */
export function GenreQuestion({ loading, onPick }: { loading: boolean; onPick: (genre: string) => void }) {
  const { t } = useTranslation("common");
  const genres = useGenreChoices();

  return (
    <Stack gap="xl" py="lg">
      <Center>
        <SectionTitle>{t("games.editor.genre.question")}</SectionTitle>
      </Center>
      {loading || genres.loading ? (
        <Center py="xl">
          <Loader />
        </Center>
      ) : (
        <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="lg">
          {genres.choices.map((genre) => (
            <UnstyledButton key={genre.id} onClick={() => onPick(genre.id)} style={{ height: "100%" }}>
              <Paper withBorder radius="lg" p="xl" className={classes.genreCard}>
                <Stack gap="sm" align="flex-start">
                  <ThemeIcon size={64} radius="xl" variant="light">
                    {GENRE_ICON[genre.id] ?? <IconSparkles size={36} />}
                  </ThemeIcon>
                  <CardTitle>{genre.name}</CardTitle>
                  <HelperText>{t(`games.editor.genre.pitch.${genreWording(genre.id)}`)}</HelperText>
                  {genre.needs && (
                    <Badge variant="light" color="orange">
                      {genre.needs}
                    </Badge>
                  )}
                </Stack>
              </Paper>
            </UnstyledButton>
          ))}
        </SimpleGrid>
      )}
    </Stack>
  );
}

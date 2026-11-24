import { Select, Stack, Textarea, TextInput } from "@mantine/core";
import { useTranslation } from "react-i18next";
import { Label } from "@components/typography";
import { genreWording, joinDuo, splitDuo, STATUS_FIELDS_MAX } from "../../lib/editorPages";
import { NAME_MAX, type GameEditor } from "../../hooks/useGameEditor";
import { useThemeOptions } from "../../hooks/useEditorChoices";
import { StatusFieldsEditor } from "../StatusFieldsEditor";
import { GameCoverSection } from "../GameCoverSection";

interface SectionProps {
  editor: GameEditor;
  /** Whether the genre reads a game field. */
  shows: (field: "statusFields" | "imageStyle" | "voiceStyle") => boolean;
}

/** Card: what players see in the gallery, the title and a description. */
export function CardSection({ editor }: SectionProps) {
  const { t } = useTranslation("common");
  const { values, set, rights } = editor;
  const wording = genreWording(values.genre);
  return (
    <Stack gap="lg">
      <TextInput
        label={t("games.editor.card.title")}
        placeholder={t(`games.editor.card.${wording}.titlePlaceholder`)}
        description={
          rights.card ? t("games.editor.card.titleHint", { length: values.name.length, max: NAME_MAX }) : undefined
        }
        value={values.name}
        onChange={(e) => set("name", e.target.value)}
        error={editor.nameError ? t(editor.nameError) : undefined}
        maxLength={NAME_MAX}
        readOnly={!rights.card}
        required
        data-autofocus
      />
      <Textarea
        label={t("games.editor.card.description")}
        description={t("games.editor.card.descriptionHint")}
        placeholder={t(`games.editor.card.${wording}.descriptionPlaceholder`)}
        value={values.description}
        onChange={(e) => set("description", e.target.value)}
        minRows={2}
        autosize
        readOnly={!rights.card}
      />
    </Stack>
  );
}

/**
 * Story: the deeper part. An Adventure says what it is about and how it
 * begins, then the values it tracks; an Encounter who the character is and
 * how they greet the player; a Duo who its two characters are, in two fields
 * kept in the game's one scenario, and the scene they are in.
 */
export function StorySection({ editor, shows }: SectionProps) {
  const { t } = useTranslation("common");
  const { values, set, rights } = editor;
  const wording = genreWording(values.genre);
  const [first, second] = splitDuo(values.systemMessageScenario);
  return (
    <Stack gap="lg">
      {wording === "duo" ? (
        <>
          <Textarea
            label={t("games.editor.story.duo.firstCharacter")}
            description={t("games.editor.story.duo.firstCharacterHint")}
            placeholder={t("games.editor.story.duo.firstCharacterPlaceholder")}
            value={first}
            onChange={(e) => set("systemMessageScenario", joinDuo(e.target.value, second))}
            minRows={4}
            autosize
            readOnly={!rights.content}
            required
          />
          <Textarea
            label={t("games.editor.story.duo.secondCharacter")}
            description={t("games.editor.story.duo.secondCharacterHint")}
            placeholder={t("games.editor.story.duo.secondCharacterPlaceholder")}
            value={second}
            onChange={(e) => set("systemMessageScenario", joinDuo(first, e.target.value))}
            minRows={4}
            autosize
            readOnly={!rights.content}
            required
          />
        </>
      ) : (
        <Textarea
          label={t(`games.editor.story.${wording}.scenario`)}
          description={t(`games.editor.story.${wording}.scenarioHint`)}
          placeholder={t(`games.editor.story.${wording}.scenarioPlaceholder`)}
          value={values.systemMessageScenario}
          onChange={(e) => set("systemMessageScenario", e.target.value)}
          minRows={5}
          autosize
          readOnly={!rights.content}
          required
        />
      )}
      <Textarea
        label={t(`games.editor.story.${wording}.gameStart`)}
        description={t(`games.editor.story.${wording}.gameStartHint`)}
        placeholder={t(`games.editor.story.${wording}.gameStartPlaceholder`)}
        value={values.systemMessageGameStart}
        onChange={(e) => set("systemMessageGameStart", e.target.value)}
        minRows={3}
        autosize
        readOnly={!rights.content}
        required
      />
      {shows("statusFields") && (
        <StatusFieldsEditor
          value={values.statusFields}
          onChange={(next) => set("statusFields", next)}
          readOnly={!rights.content}
          max={STATUS_FIELDS_MAX}
          maxHint={t("games.statusFieldsEditor.maxHint")}
        />
      )}
    </Stack>
  );
}

/** Look: the pictures' style, the cover, and the page the game is played in. */
export function LookSection({ editor, shows }: SectionProps) {
  const { t } = useTranslation("common");
  const { values, set, rights } = editor;
  const themeOptions = useThemeOptions(values.theme);
  return (
    <Stack gap="lg">
      {shows("imageStyle") && (
        <TextInput
          label={t("games.editor.look.imageStyle")}
          description={t("games.editor.look.imageStyleHint")}
          placeholder={t("games.editor.look.imageStylePlaceholder")}
          value={values.imageStyle}
          onChange={(e) => set("imageStyle", e.target.value)}
          readOnly={!rights.content}
          required
        />
      )}
      <Stack gap={4}>
        <Label>{t("games.cover.title")}</Label>
        <GameCoverSection game={editor.creating ? undefined : editor.game} mayEdit={rights.card} />
      </Stack>
      <Select
        label={t("games.editFields.theme")}
        description={t("games.editFields.themeHint")}
        placeholder={t("games.editFields.themeDefault")}
        data={themeOptions}
        value={values.theme || null}
        onChange={(next) => set("theme", next ?? "")}
        clearable
        searchable
        readOnly={!rights.card}
      />
    </Stack>
  );
}

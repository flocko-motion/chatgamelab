import { useTranslation } from "react-i18next";
import { useGenres, useThemes } from "@/api/hooks";

/**
 * The genres a new game can be, as the editor offers them: each with its
 * name and, when it needs features of a key, which ones, in a sentence.
 */
export function useGenreChoices() {
  const { t, i18n } = useTranslation("common");
  const { data: genres, isLoading } = useGenres();
  const list = new Intl.ListFormat(i18n.language, { type: "conjunction" });
  return {
    loading: isLoading || !genres,
    choices: (genres ?? []).map((genre) => {
      const needs = genre.needs ?? [];
      return {
        id: genre.id,
        name: t(`games.genres.${genre.id}`, { defaultValue: genre.id }),
        needs:
          needs.length > 0
            ? t("games.editFields.genreNeeds", {
                features: list.format(needs.map((need) => t(`games.features.${need}`, { defaultValue: need }))),
              })
            : "",
      };
    }),
  };
}

/** The playback-engine's looks as the editor offers them; a look the game has stays offered. */
export function useThemeOptions(current: string) {
  const { t } = useTranslation("common");
  const { data: themes } = useThemes();
  const label = (id: string) => t(`games.theme.presets.${id}`, { defaultValue: id });
  const options = (themes ?? [])
    .map((theme) => theme.id ?? "")
    .filter((id) => id !== "")
    .map((id) => ({ value: id, label: label(id) }));
  if (current && !options.some((o) => o.value === current)) {
    options.push({ value: current, label: label(current) });
  }
  return options;
}

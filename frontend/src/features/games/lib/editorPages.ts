import type { GameField } from "@/api/types";
import type { GameFormFields } from "./gameUpdate";

/**
 * The editor's sections, one page each, in the order a game is designed:
 * its card in the gallery, its story, how it looks, how it sounds, and who
 * gets to play it. The genre is chosen once, before them, when a game is
 * made; deleting is an entry of its own, after them.
 */
export type EditorPage = "card" | "story" | "look" | "sound" | "sharing";

const ORDER: EditorPage[] = ["card", "story", "look", "sound", "sharing"];

/**
 * The sections the editor offers: Sound when the genre reads a voice style,
 * Sharing once the game is saved.
 */
export function editorPages(saved: boolean, shows: (field: GameField) => boolean): EditorPage[] {
  return ORDER.filter((page) => {
    switch (page) {
      case "sound":
        return shows("voiceStyle");
      case "sharing":
        return saved;
      default:
        return true;
    }
  });
}

const filled = (s: string) => s.trim() !== "";

/**
 * Whether a section is done (its green check): every field it requires is
 * filled in. A section that requires nothing (Sound, Sharing) is done once
 * it has been opened, or when the game is saved already.
 */
export function pageDone(
  page: EditorPage,
  values: GameFormFields,
  shows: (field: GameField) => boolean,
  opened: boolean,
  saved: boolean,
): boolean {
  switch (page) {
    case "card":
      return filled(values.name);
    case "story":
      return filled(values.systemMessageScenario) && filled(values.systemMessageGameStart);
    case "look":
      return !shows("imageStyle") || filled(values.imageStyle);
    case "sound":
    case "sharing":
      return opened || saved;
  }
}

/** The section that holds a field of the game, as the review names it. */
export function pageOfField(field: string): EditorPage | null {
  switch (field) {
    case "name":
    case "description":
      return "card";
    case "systemMessageScenario":
    case "systemMessageGameStart":
    case "statusFields":
      return "story";
    case "imageStyle":
    case "theme":
      return "look";
    case "voiceStyle":
      return "sound";
    case "public":
      return "sharing";
  }
  return null;
}

/** The most game values the editor lets an author add; more brought along stay. */
export const STATUS_FIELDS_MAX = 5;

/**
 * The key segment of a genre's own wording: an Encounter talks of a
 * character, an Adventure (and a genre without words of its own) of a world.
 */
export function genreWording(genre: string): "adventure" | "encounter" {
  return genre === "npc-live" ? "encounter" : "adventure";
}

/**
 * A section's one line on the overview: what it holds, as written, or a
 * translation key when it is the editor's to say (nothing filled in yet, a
 * built-in voice, public or private).
 */
export function pageSummary(page: EditorPage, values: GameFormFields): { text: string } | { key: string } {
  const written = (text: string) => (text.trim() ? { text: text.trim() } : null);
  switch (page) {
    case "card":
      return written([values.name.trim(), values.description.trim()].filter(Boolean).join(" — ")) ?? {
        key: "games.editor.pages.card.intro",
      };
    case "story":
      return written(values.systemMessageScenario) ?? { key: "games.editor.pages.story.intro" };
    case "look":
      return written(values.imageStyle) ?? { key: "games.editor.pages.look.intro" };
    case "sound":
      return written(values.voiceStyle) ?? { key: "games.editor.sound.builtInVoice" };
    case "sharing":
      return { key: values.public ? "games.visibility.public" : "games.visibility.private" };
  }
}

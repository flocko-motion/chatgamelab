import type { CreateGameReq, UpdateGameReq } from "@/api/types";

/**
 * A game update: only the fields that change. A field left out stays as it
 * is; "" clears a text field. The server refuses the whole update when it
 * touches a section the caller may not edit.
 */
export type GameChanges = UpdateGameReq;

/** The fields the edit dialog shows, as the dialog holds them. */
export type GameFormFields = Required<
  Pick<
    UpdateGameReq,
    | "name"
    | "description"
    | "public"
    | "systemMessageScenario"
    | "systemMessageGameStart"
    | "imageStyle"
    | "voiceStyle"
    | "statusFields"
    | "genre"
    | "theme"
  >
>;

/** The fields that differ from what was loaded: what the dialog sends. */
export function gameChanges(loaded: GameFormFields, edited: GameFormFields): GameChanges {
  const changes: GameChanges = {};
  for (const key of Object.keys(edited) as (keyof GameFormFields)[]) {
    if (edited[key] !== loaded[key]) {
      (changes as Record<string, unknown>)[key] = edited[key];
    }
  }
  return changes;
}

/**
 * A new game: the name, and the fields that differ from the template it
 * starts from. What is left out keeps the template's value.
 */
export function gameCreate(template: GameFormFields, edited: GameFormFields): CreateGameReq {
  return { ...gameChanges(template, edited), name: edited.name };
}

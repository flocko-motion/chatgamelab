import { useMemo, useState } from "react";
import { useGame, useGenres } from "@/api/hooks";
import type { Mode } from "@/api/types";

/** One of a genre's prompts as the advanced settings show it. */
export interface LaunchPrompt {
  readonly name: string;
  /** The text the session will start with. */
  readonly text: string;
  /** The genre's own text. */
  readonly original: string;
  readonly edited: boolean;
}

/**
 * The prompts of a game's genre, to adapt before a session starts: each
 * starts as the genre's own, and only the changed ones go to the server.
 * Prompts belong to a half of the genre, live or rounds; for a genre with
 * both, the starter picks the half first and adapts its prompts.
 */
export function useLaunchPrompts(gameId: string) {
  const game = useGame(gameId);
  const genres = useGenres();
  const [chosen, setChosen] = useState<Mode | null>(null);
  const [texts, setTexts] = useState<Record<string, string>>({});

  const halves = useMemo(
    () => genres.data?.find((g) => g.id === game.data?.genre)?.subgenres ?? [],
    [genres.data, game.data?.genre],
  );
  const half = halves.find((h) => h.mode === chosen) ?? halves[0];
  const mode = half?.mode;
  /** Each half keeps its own changes, so switching back finds them. */
  const keyOf = (name: string) => `${mode}:${name}`;

  const prompts: LaunchPrompt[] = (half?.prompts ?? []).map(
    ({ name, text }) => {
      const current = texts[keyOf(name)] ?? text;
      return { name, text: current, original: text, edited: current !== text };
    },
  );

  return {
    prompts,
    /** The halves the genre has, live first: a choice only where there are two. */
    modes: halves.map((h) => h.mode),
    mode,
    /** The mode to start in: the half chosen, where the genre has two. */
    startMode: halves.length > 1 ? mode : undefined,
    setMode: setChosen,
    loading: game.isLoading || genres.isLoading,
    failed: game.isError || genres.isError,
    setText: (name: string, text: string) =>
      setTexts((all) => ({ ...all, [keyOf(name)]: text })),
    reset: (name: string) =>
      setTexts((all) => {
        const rest = { ...all };
        delete rest[keyOf(name)];
        return rest;
      }),
    /** The changed prompts of the half chosen, by name: what the session's spec replaces. */
    overrides: (): Record<string, string> =>
      Object.fromEntries(
        prompts.filter((p) => p.edited).map((p) => [p.name, p.text]),
      ),
  };
}

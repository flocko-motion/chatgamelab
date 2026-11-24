import { useMemo, useState } from "react";
import { useGame, useGenres } from "@/api/hooks";

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
 */
export function useLaunchPrompts(gameId: string) {
  const game = useGame(gameId);
  const genres = useGenres();
  const [texts, setTexts] = useState<Record<string, string>>({});

  const originals = useMemo(
    () => genres.data?.find((g) => g.id === game.data?.genre)?.prompts ?? [],
    [genres.data, game.data?.genre],
  );

  const prompts: LaunchPrompt[] = originals.map(({ name, text }) => {
    const current = texts[name] ?? text;
    return { name, text: current, original: text, edited: current !== text };
  });

  return {
    prompts,
    loading: game.isLoading || genres.isLoading,
    failed: game.isError || genres.isError,
    setText: (name: string, text: string) =>
      setTexts((all) => ({ ...all, [name]: text })),
    reset: (name: string) =>
      setTexts((all) => {
        const rest = { ...all };
        delete rest[name];
        return rest;
      }),
    /** The changed prompts, by name: what the session's spec replaces. */
    overrides: (): Record<string, string> =>
      Object.fromEntries(prompts.filter((p) => p.edited).map((p) => [p.name, p.text])),
  };
}

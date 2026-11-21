/**
 * What the playback-engine says to the player, in the session's language. The
 * texts are the platform's own, under `playbackEngine` in its language files,
 * so they are written and translated with everything else. English is
 * bundled; the bundle step puts each other language beside the page as
 * locales/<lang>.json, and a text a language lacks is said in English.
 */
import { playbackEngine as english } from "../../../../frontend/src/i18n/locales/en.json";

type Texts = typeof english;
type Group = keyof Texts;

/** A text by its place in the language file, such as "problems.keyRejected". */
export type TextKey = { [G in Group]: `${G}.${keyof Texts[G] & string}` }[Group];

let current: Partial<Record<Group, Partial<Record<string, string>>>> = english;

/** The text for a key in the current language. */
export function text(key: TextKey): string {
  const [group, name] = key.split(".") as [Group, string];
  return current[group]?.[name] ?? (english[group] as Record<string, string>)[name] ?? key;
}

/**
 * What a block is called in the pipeline view, by its name in the wiring: a
 * name like "expand" says nothing to a player, "Narrate" does. A block with no
 * text keeps its name.
 */
export function blockLabel(name: string): string {
  const own = current.blocks as Partial<Record<string, string>> | undefined;
  return own?.[name] ?? (english.blocks as Record<string, string>)[name] ?? name;
}

/**
 * What a wire is called in the pipeline view, by the blocks it joins: "style"
 * says more than its kind "props". A wire with no text is called by its kind.
 */
export function wireLabel(from: string, to: string, kind: string): string {
  const key = `${from}>${to}`;
  const own = current.wires as Partial<Record<string, string>> | undefined;
  return own?.[key] ?? (english.wires as Record<string, string>)[key] ?? kind;
}

/**
 * Switches to a language by its BCP 47 tag, read by its primary subtag. A
 * language with no file beside the page, or one that fails to load, leaves
 * English in place.
 */
export async function useLanguage(tag: string, page: string | URL): Promise<void> {
  const code = tag.toLowerCase().split("-")[0] ?? "en";
  if (code === "en" || !/^[a-z]{2,3}$/.test(code)) {
    current = english;
    return;
  }
  try {
    const response = await fetch(new URL(`locales/${code}.json`, page));
    if (response.ok) current = (await response.json()) as typeof current;
  } catch {
    // Unreachable is not untranslatable: English still says it.
  }
}

/** Something to tell the player: a text by its key, and what went wrong, for whoever debugs. */
export interface Notice {
  readonly key: TextKey;
  readonly detail?: string;
}

/** A notice, keeping the error it came from as its detail. */
export function notice(key: TextKey, cause?: unknown): Notice {
  if (cause === undefined) return { key };
  if (cause instanceof NoticeError) return cause.notice;
  return { key, detail: cause instanceof Error ? cause.message : String(cause) };
}

/** A notice in a log line: its key and its detail. */
export function describe(said: Notice): string {
  return said.detail ? `${said.key}: ${said.detail}` : said.key;
}

/** An error that already knows what to tell the player. */
export class NoticeError extends Error {
  constructor(readonly notice: Notice) {
    super(describe(notice));
  }
}

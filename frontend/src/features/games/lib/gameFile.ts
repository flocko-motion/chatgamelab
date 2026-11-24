import { unzipSync } from "fflate";
import type { CreateGameFormData } from "../types";
import { parseGameYaml } from "./parseGameYaml";

/**
 * The largest game file taken, zip or YAML, as the server takes a cover: a
 * stored cover is tens of kilobytes and a game file a few, so a megabyte is
 * ample and turns away what is no game file.
 */
export const MAX_GAME_FILE_BYTES = 1024 * 1024;

/**
 * A game file read: the form's fields, and the cover a zip brought along
 * with the signature the exporting server gave it (cover.sig), which the
 * server asks for to take the cover back.
 */
export interface GameFileContents {
  data: Partial<CreateGameFormData>;
  cover?: ImportedCover;
}

/** A cover from a game's zip and its signature; empty when the zip had none. */
export interface ImportedCover {
  picture: Blob;
  signature: string;
}

/** Why a game file could not be read, as a translation key. */
export class GameFileError extends Error {
  readonly key: string;
  constructor(key: string) {
    super(key);
    this.key = key;
  }
}

/** A zip starts with a local file header, whatever the file is called. */
function isZip(bytes: Uint8Array): boolean {
  return bytes.length >= 4 && bytes[0] === 0x50 && bytes[1] === 0x4b && bytes[2] === 0x03 && bytes[3] === 0x04;
}

/** A picture's type, read from its first bytes: PNG or JPEG, else none. */
function pictureType(bytes: Uint8Array): string | null {
  if (bytes[0] === 0xff && bytes[1] === 0xd8 && bytes[2] === 0xff) return "image/jpeg";
  if (bytes[0] === 0x89 && bytes[1] === 0x50 && bytes[2] === 0x4e && bytes[3] === 0x47) return "image/png";
  return null;
}

/** The entry named first in names, else the first whose name ends in one of the suffixes. */
function pick(entries: Record<string, Uint8Array>, names: string[], suffixes: string[]): Uint8Array | undefined {
  // Folders and macOS's resource forks are no files of the game.
  const files = Object.keys(entries).filter((n) => !n.endsWith("/") && !n.startsWith("__MACOSX/"));
  const base = (n: string) => n.split("/").pop()!.toLowerCase();
  const named = files.find((n) => names.includes(base(n)));
  const found = named ?? files.find((n) => suffixes.some((s) => base(n).endsWith(s)));
  return found ? entries[found] : undefined;
}

/**
 * Reads a game file: a zip holding the game's YAML and its cover, as the
 * export writes it, or the YAML alone, as older exports are.
 */
export async function readGameFile(file: File): Promise<GameFileContents> {
  if (file.size > MAX_GAME_FILE_BYTES) {
    throw new GameFileError("games.importExport.errors.tooLarge");
  }
  const bytes = new Uint8Array(await file.arrayBuffer());
  if (!isZip(bytes)) {
    return { data: parseGameYaml(new TextDecoder().decode(bytes)) };
  }

  let entries: Record<string, Uint8Array>;
  try {
    entries = unzipSync(bytes);
  } catch {
    throw new GameFileError("games.importExport.errors.unreadable");
  }
  const yaml = pick(entries, ["game.yaml", "game.yml"], [".yaml", ".yml"]);
  if (!yaml) {
    throw new GameFileError("games.importExport.errors.noGameInZip");
  }
  const data = parseGameYaml(new TextDecoder().decode(yaml));
  const picture = pick(entries, ["cover.jpg", "cover.jpeg", "cover.png"], [".jpg", ".jpeg", ".png"]);
  const type = picture ? pictureType(picture) : null;
  if (!picture || !type) {
    return { data };
  }
  const signature = entries["cover.sig"] ? new TextDecoder().decode(entries["cover.sig"]).trim() : "";
  return { data, cover: { picture: new Blob([picture as BlobPart], { type }), signature } };
}

import yaml from "js-yaml";
import type { CreateGameFormData } from "../types";

/**
 * Shape of the backend Game YAML export (snake_case yaml tags from obj.Game).
 */
interface GameYaml {
  name?: string;
  description?: string;
  system_message_scenario?: string;
  system_message_game_start?: string;
  image_style?: string;
  voice_style?: string;
  voices?: Record<string, unknown>;
  status_fields?: string;
  genre?: string;
  theme?: string;
}

/**
 * Parse a YAML string (as exported by the backend) into CreateGameFormData.
 * Uses js-yaml for robust parsing of all YAML features.
 * Ignores unknown keys (id, css, etc.).
 */
export function parseGameYaml(content: string): Partial<CreateGameFormData> {
  const parsed = yaml.load(content) as GameYaml | null;
  if (!parsed || typeof parsed !== "object") {
    return { name: "", description: "", isPublic: false };
  }

  const str = (v: unknown): string | undefined =>
    typeof v === "string" && v.trim() ? v.trim() : undefined;
  // The choice for each voice, by its name; anything else is left out.
  const voices = (v: unknown): Record<string, string> | undefined => {
    if (!v || typeof v !== "object") return undefined;
    const out: Record<string, string> = {};
    for (const [name, choice] of Object.entries(v)) {
      if (typeof choice === "string") out[name] = choice;
    }
    return out;
  };

  return {
    name: str(parsed.name) ?? "",
    description: str(parsed.description) ?? "",
    isPublic: false,
    systemMessageScenario: str(parsed.system_message_scenario),
    systemMessageGameStart: str(parsed.system_message_game_start),
    imageStyle: str(parsed.image_style),
    voiceStyle: str(parsed.voice_style),
    voices: voices(parsed.voices),
    statusFields: str(parsed.status_fields),
    genre: str(parsed.genre),
    theme: str(parsed.theme),
  };
}

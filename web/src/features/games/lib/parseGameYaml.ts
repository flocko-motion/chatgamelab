import type { CreateGameFormData } from "../types";

/**
 * Known YAML keys from the backend Game struct (snake_case yaml tags).
 * Maps yaml key → CreateGameFormData field.
 */
const YAML_KEY_MAP: Record<string, keyof CreateGameFormData> = {
  name: "name",
  description: "description",
  system_message_scenario: "systemMessageScenario",
  system_message_game_start: "systemMessageGameStart",
  image_style: "imageStyle",
  status_fields: "statusFields",
};

/**
 * Parse a simple flat YAML string (as exported by the backend) into CreateGameFormData.
 *
 * Handles:
 * - Simple key: value pairs
 * - Quoted values (single/double)
 * - Block scalars (| and >)
 * - Multi-line continuation (indented lines)
 *
 * Ignores unknown keys (id, css, etc.).
 */
export function parseGameYaml(yaml: string): Partial<CreateGameFormData> {
  const result: Record<string, string> = {};
  const lines = yaml.split("\n");

  let currentKey: string | null = null;
  let currentValue: string[] = [];
  let isBlockScalar = false;
  let blockIndent: number | null = null;

  const flushCurrent = () => {
    if (currentKey && currentKey in YAML_KEY_MAP) {
      const formKey = YAML_KEY_MAP[currentKey];
      result[formKey] = currentValue.join("\n").trim();
    }
    currentKey = null;
    currentValue = [];
    isBlockScalar = false;
    blockIndent = null;
  };

  for (const line of lines) {
    // Check if this is a top-level key (no leading whitespace, contains colon)
    const keyMatch = line.match(/^([a-z_]+)\s*:\s*(.*)/);

    if (keyMatch && !isBlockScalar) {
      flushCurrent();
      const [, key, rawValue] = keyMatch;
      currentKey = key;

      const trimmed = rawValue.trim();
      if (trimmed === "|" || trimmed === ">") {
        // Block scalar — collect following indented lines
        isBlockScalar = true;
        blockIndent = null;
      } else {
        // Inline value — strip optional quotes
        const unquoted = trimmed.replace(/^["']|["']$/g, "");
        currentValue = [unquoted];
      }
    } else if (isBlockScalar && currentKey) {
      // Inside a block scalar
      if (line.trim() === "" && blockIndent === null) {
        // Empty line before any content — skip
        continue;
      }
      if (blockIndent === null) {
        // Detect indent from first content line
        const indentMatch = line.match(/^(\s+)/);
        blockIndent = indentMatch ? indentMatch[1].length : 0;
      }
      if (line.trim() === "" || line.match(/^\s/)) {
        // Still in the block — strip the block indent
        const stripped =
          blockIndent > 0 ? line.slice(blockIndent) : line;
        currentValue.push(stripped);
      } else {
        // Non-indented line means block ended — re-process this line
        flushCurrent();
        const newKeyMatch = line.match(/^([a-z_]+)\s*:\s*(.*)/);
        if (newKeyMatch) {
          const [, key, rawValue] = newKeyMatch;
          currentKey = key;
          const trimmed = rawValue.trim();
          if (trimmed === "|" || trimmed === ">") {
            isBlockScalar = true;
            blockIndent = null;
          } else {
            const unquoted = trimmed.replace(/^["']|["']$/g, "");
            currentValue = [unquoted];
          }
        }
      }
    } else if (currentKey && line.match(/^\s/) && !isBlockScalar) {
      // Continuation line for a non-block value (indented)
      currentValue.push(line.trim());
    }
  }
  flushCurrent();

  return {
    name: result.name || "",
    description: result.description || "",
    isPublic: false,
    systemMessageScenario: result.systemMessageScenario || undefined,
    systemMessageGameStart: result.systemMessageGameStart || undefined,
    imageStyle: result.imageStyle || undefined,
    statusFields: result.statusFields || undefined,
  };
}

import type {
  ObjGameSessionMessage,
  ObjStatusField,
  ObjGameTheme,
} from "@/api/generated";
import { PRESET_THEMES } from "./theme/defaults";
import type {
  PartialGameTheme,
  BackgroundAnimation as BackgroundAnimationType,
} from "./theme/types";

// ============================================================================
// Core Types
// ============================================================================

export type MessageType = "system" | "player" | "game";

export type ImageStatus = "none" | "generating" | "complete" | "error";

export interface SceneMessage {
  id: string;
  type: MessageType;
  text: string;
  statusFields?: ObjStatusField[];
  imagePrompt?: string;
  isStreaming?: boolean;
  isImageLoading?: boolean;
  imageStatus?: ImageStatus;
  imageHash?: string;
  imageErrorCode?: string;
  timestamp: Date;
  seq?: number;
  /** Set when a player message failed (AI error) — shows red with retry */
  error?: string;
  /** Machine-readable error code for i18n */
  errorCode?: string;
}

export interface StreamChunk {
  text?: string;
  textDone?: boolean;
  imageData?: string; // Base64-encoded partial/WIP image data
  imageDone?: boolean;
  error?: string; // Error message from the backend
  errorCode?: string; // Machine-readable error code (maps to frontend i18n)
}

/** Response from GET /messages/{id}/status — unified polling endpoint */
export interface MessageStatus {
  text: string;
  textDone: boolean;
  imageStatus: ImageStatus;
  imageHash?: string;
  imageError?: string;
  statusFields?: ObjStatusField[];
  error?: string;
  errorCode?: string;
}

export interface GameSessionConfig {
  model?: string;
}

export interface GameInfo {
  id?: string;
  name?: string;
  description?: string;
}

// ============================================================================
// Player State
// ============================================================================

export type GamePhase =
  | "idle"
  | "starting"
  | "playing"
  | "error"
  | "needs-api-key";

export interface StreamError {
  code: string | null;
  message: string;
}

export interface GamePlayerState {
  phase: GamePhase;
  sessionId: string | null;
  gameInfo: GameInfo | null;
  messages: SceneMessage[];
  statusFields: ObjStatusField[];
  isWaitingForResponse: boolean;
  error: string | null;
  /** Full error object for error code extraction */
  errorObject: unknown;
  /** Recoverable error from SSE stream (mid-game AI errors) — player can dismiss and retry */
  streamError: StreamError | null;
  /** AI-generated visual theme from the session */
  theme: ObjGameTheme | null;
}

// ============================================================================
// Utility Functions
// ============================================================================

export function mapApiMessageToScene(msg: ObjGameSessionMessage): SceneMessage {
  // Determine image status for non-streaming messages loaded from DB
  let imageStatus: ImageStatus | undefined;
  let imageHash: string | undefined;
  if (msg.imagePrompt) {
    if (msg.stream) {
      // Still streaming — polling will determine actual status
      imageStatus = "generating";
    } else {
      // Completed message with image prompt — image is persisted
      imageStatus = "complete";
      imageHash = "persisted";
    }
  }

  return {
    id: msg.id || crypto.randomUUID(),
    type: (msg.type as MessageType) || "game",
    text: msg.message || "",
    statusFields: msg.statusFields,
    imagePrompt: msg.imagePrompt,
    isStreaming: msg.stream,
    isImageLoading: msg.stream && !!msg.imagePrompt,
    imageStatus,
    imageHash,
    timestamp: msg.meta?.createdAt ? new Date(msg.meta.createdAt) : new Date(),
    seq: msg.seq,
  };
}

/** Maps API theme to frontend PartialGameTheme.
 *
 * Simplified structure: { preset, animation?, thinkingText?, statusEmojis? }
 * 1. Load the preset
 * 2. Apply animation override (if specified)
 * 3. Apply thinkingText override (if specified)
 * 4. Apply statusEmojis
 */
export function mapApiThemeToPartial(
  apiTheme: ObjGameTheme | null | undefined,
): PartialGameTheme | undefined {
  if (!apiTheme) return undefined;

  // Load preset (deep clone to avoid mutating the original)
  const preset = apiTheme.preset ? PRESET_THEMES[apiTheme.preset] : undefined;
  const result: PartialGameTheme = preset
    ? JSON.parse(JSON.stringify(preset))
    : {};

  // Apply animation override
  if (apiTheme.animation) {
    result.background = {
      ...result.background,
      animation: apiTheme.animation as BackgroundAnimationType,
    };
  }

  // Apply thinkingText override
  if (apiTheme.thinkingText) {
    result.thinking = {
      ...result.thinking,
      text: apiTheme.thinkingText,
    };
  }

  // Apply statusEmojis
  if (apiTheme.statusEmojis) {
    result.statusEmojis = { ...result.statusEmojis, ...apiTheme.statusEmojis };
  }

  return result;
}

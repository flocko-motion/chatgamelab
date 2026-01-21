import type { ObjGameSessionMessage, ObjStatusField, ObjApiKeyShare, ObjAiModel, ObjGameTheme } from '@/api/generated';
import { config } from '@/config/env';
import { PRESET_THEMES } from './theme/defaults';
import type { PartialGameTheme, CornerStyle, ThemeColor, BackgroundTint, PlayerIndicator, CardBgColor, FontColor, CardBorderThickness, ThinkingStyle, MessageFont, DividerStyle } from './theme/types';

// ============================================================================
// Core Types
// ============================================================================

export type MessageType = 'system' | 'player' | 'game';

export interface SceneMessage {
  id: string;
  type: MessageType;
  text: string;
  statusFields?: ObjStatusField[];
  imageUrl?: string;
  imagePrompt?: string;
  isStreaming?: boolean;
  isImageLoading?: boolean;
  timestamp: Date;
  seq?: number;
}

export interface StreamChunk {
  text?: string;
  textDone?: boolean;
  imageDone?: boolean;
}

export interface GameSessionConfig {
  shareId: string;
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

export type GamePhase = 'selecting-key' | 'starting' | 'playing' | 'error';

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
  /** AI-generated visual theme from the session */
  theme: ObjGameTheme | null;
}

// ============================================================================
// Theme System
// ============================================================================

export type ThemePreset = 'classic' | 'modern' | 'dark' | 'playful' | 'minimal';

export interface GameTheme {
  preset: ThemePreset;
  // Future: custom overrides
}

export const DEFAULT_THEME: GameTheme = {
  preset: 'modern',
};

// ============================================================================
// Utility Functions
// ============================================================================

export function mapApiMessageToScene(msg: ObjGameSessionMessage): SceneMessage {
  const hasImage = msg.imagePrompt && msg.id;
  
  return {
    id: msg.id || crypto.randomUUID(),
    type: (msg.type as MessageType) || 'game',
    text: msg.message || '',
    statusFields: msg.statusFields,
    imagePrompt: msg.imagePrompt,
    imageUrl: hasImage ? `${config.API_BASE_URL}/messages/${msg.id}/image` : undefined,
    isStreaming: msg.stream,
    timestamp: msg.meta?.createdAt ? new Date(msg.meta.createdAt) : new Date(),
    seq: msg.seq,
  };
}

export function getDefaultApiKey(apiKeys: ObjApiKeyShare[]): ObjApiKeyShare | undefined {
  return apiKeys.find(k => k.isUserDefault) || apiKeys[0];
}

export function getModelsForApiKey(
  apiKey: ObjApiKeyShare | undefined, 
  platforms: { id?: string; models?: ObjAiModel[] }[]
): ObjAiModel[] {
  if (!apiKey?.apiKey?.platform) return [];
  const platform = platforms.find(p => p.id === apiKey.apiKey?.platform);
  return platform?.models || [];
}

/** Maps API theme type to frontend PartialGameTheme (type-safe conversion)
 * 
 * New structure: { preset: "space", override: { ...fields to override... } }
 * 1. Load the preset (if specified)
 * 2. Apply any overrides on top
 */
export function mapApiThemeToPartial(apiTheme: ObjGameTheme | null | undefined): PartialGameTheme | undefined {
  if (!apiTheme) return undefined;
  
  // Start with preset if specified
  let result: PartialGameTheme = {};
  
  if (apiTheme.preset && apiTheme.preset !== 'custom') {
    const preset = PRESET_THEMES[apiTheme.preset];
    if (preset) {
      // Deep clone preset
      result = JSON.parse(JSON.stringify(preset));
    }
  }
  
  // Apply overrides if present
  const override = apiTheme.override;
  if (override) {
    if (override.corners) {
      result.corners = { ...result.corners, ...mapCorners(override.corners) };
    }
    if (override.background) {
      result.background = { ...result.background, tint: override.background.tint as BackgroundTint };
    }
    if (override.player) {
      result.player = { ...result.player, ...mapPlayer(override.player) };
    }
    if (override.gameMessage) {
      result.gameMessage = { ...result.gameMessage, ...mapGameMessage(override.gameMessage) };
    }
    if (override.cards) {
      result.cards = { ...result.cards, borderThickness: override.cards.borderThickness as CardBorderThickness };
    }
    if (override.thinking) {
      result.thinking = { ...result.thinking, ...mapThinking(override.thinking) };
    }
    if (override.typography) {
      result.typography = { ...result.typography, messages: override.typography.messages as MessageFont };
    }
    if (override.statusFields) {
      result.statusFields = { ...result.statusFields, ...mapStatusFields(override.statusFields) };
    }
    if (override.header) {
      result.header = { ...result.header, ...mapHeader(override.header) };
    }
    if (override.divider) {
      result.divider = { ...result.divider, ...mapDivider(override.divider) };
    }
    if (override.statusEmojis) {
      result.statusEmojis = { ...result.statusEmojis, ...override.statusEmojis };
    }
  }
  
  return result;
}

// Helper functions for type-safe mapping
function mapCorners(c: NonNullable<ObjGameTheme['override']>['corners']): Partial<PartialGameTheme['corners']> {
  return {
    style: c?.style as CornerStyle,
    color: c?.color as ThemeColor,
  };
}

function mapPlayer(p: NonNullable<ObjGameTheme['override']>['player']): Partial<PartialGameTheme['player']> {
  return {
    color: p?.color as ThemeColor,
    indicator: p?.indicator as PlayerIndicator,
    indicatorBlink: p?.indicatorBlink,
    bgColor: p?.bgColor as CardBgColor,
    fontColor: p?.fontColor as FontColor,
    borderColor: p?.borderColor as ThemeColor,
  };
}

function mapGameMessage(g: NonNullable<ObjGameTheme['override']>['gameMessage']): Partial<PartialGameTheme['gameMessage']> {
  return {
    dropCap: g?.dropCap,
    dropCapColor: g?.dropCapColor as ThemeColor,
    bgColor: g?.bgColor as CardBgColor,
    fontColor: g?.fontColor as FontColor,
    borderColor: g?.borderColor as ThemeColor,
  };
}

function mapThinking(t: NonNullable<ObjGameTheme['override']>['thinking']): Partial<PartialGameTheme['thinking']> {
  return {
    text: t?.text,
    style: t?.style as ThinkingStyle,
  };
}

function mapStatusFields(s: NonNullable<ObjGameTheme['override']>['statusFields']): Partial<PartialGameTheme['statusFields']> {
  return {
    bgColor: s?.bgColor as CardBgColor,
    accentColor: s?.accentColor as ThemeColor,
    borderColor: s?.borderColor as ThemeColor,
    fontColor: s?.fontColor as FontColor,
  };
}

function mapHeader(h: NonNullable<ObjGameTheme['override']>['header']): Partial<PartialGameTheme['header']> {
  return {
    bgColor: h?.bgColor as CardBgColor,
    fontColor: h?.fontColor as FontColor,
    accentColor: h?.accentColor as ThemeColor,
  };
}

function mapDivider(d: NonNullable<ObjGameTheme['override']>['divider']): Partial<PartialGameTheme['divider']> {
  return {
    style: d?.style as DividerStyle,
    color: d?.color as ThemeColor,
  };
}

import type { ObjGameSessionMessage, ObjStatusField, ObjApiKeyShare, ObjAiModel } from '@/api/generated';
import { config } from '@/config/env';

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

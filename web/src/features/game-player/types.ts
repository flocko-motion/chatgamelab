import type { ObjGameSessionMessage, ObjStatusField, ObjApiKeyShare, ObjAiModel } from '@/api/generated';
import { config } from '@/config/env';

export type MessageType = 'system' | 'player' | 'game';

export interface ChatMessage {
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

export interface GamePlayerState {
  phase: 'selecting-key' | 'starting' | 'playing' | 'error';
  sessionId: string | null;
  gameInfo: GameInfo | null;
  messages: ChatMessage[];
  statusFields: ObjStatusField[];
  isWaitingForResponse: boolean;
  error: string | null;
}

export function mapApiMessageToChat(msg: ObjGameSessionMessage): ChatMessage {
  // For messages with images, use the image endpoint URL
  // The actual image data is fetched from /api/messages/{id}/image
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

export function getModelsForApiKey(apiKey: ObjApiKeyShare | undefined, platforms: { id?: string; models?: ObjAiModel[] }[]): ObjAiModel[] {
  if (!apiKey?.apiKey?.platform) return [];
  const platform = platforms.find(p => p.id === apiKey.apiKey?.platform);
  return platform?.models || [];
}

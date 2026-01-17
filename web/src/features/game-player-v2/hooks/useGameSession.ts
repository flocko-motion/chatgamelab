import { useState, useCallback, useRef, useEffect } from 'react';
import { useRequiredAuthenticatedApi } from '@/api/useAuthenticatedApi';
import { useAuth } from '@/providers/AuthProvider';
import { config } from '@/config/env';
import type { RoutesSessionResponse } from '@/api/generated';
import type { SceneMessage, StreamChunk, GameSessionConfig, GamePlayerState } from '../types';
import { mapApiMessageToScene } from '../types';

const INITIAL_STATE: GamePlayerState = {
  phase: 'selecting-key',
  sessionId: null,
  gameInfo: null,
  messages: [],
  statusFields: [],
  isWaitingForResponse: false,
  error: null,
};

export function useGameSession(gameId: string) {
  const api = useRequiredAuthenticatedApi();
  const { getAccessToken } = useAuth();
  const [state, setState] = useState<GamePlayerState>(INITIAL_STATE);
  const abortControllerRef = useRef<AbortController | null>(null);

  const updateMessage = useCallback((messageId: string, update: Partial<SceneMessage>) => {
    setState(prev => ({
      ...prev,
      messages: prev.messages.map(msg =>
        msg.id === messageId ? { ...msg, ...update } : msg
      ),
    }));
  }, []);

  const appendTextToMessage = useCallback((messageId: string, text: string) => {
    setState(prev => ({
      ...prev,
      messages: prev.messages.map(msg =>
        msg.id === messageId ? { ...msg, text: msg.text + text } : msg
      ),
    }));
  }, []);

  const connectToStream = useCallback(async (messageId: string) => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }

    const controller = new AbortController();
    abortControllerRef.current = controller;

    try {
      const token = await getAccessToken();
      const streamUrl = `${config.API_BASE_URL}/messages/${messageId}/stream`;
      
      const response = await fetch(streamUrl, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Accept': 'text/event-stream',
        },
        signal: controller.signal,
      });

      if (!response.ok) {
        throw new Error(`Stream request failed: ${response.status}`);
      }

      const reader = response.body?.getReader();
      if (!reader) {
        throw new Error('No response body');
      }

      const decoder = new TextDecoder();
      let buffer = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const data = line.slice(6);
              const chunk: StreamChunk = JSON.parse(data);

              if (chunk.text) {
                appendTextToMessage(messageId, chunk.text);
              }

              if (chunk.textDone) {
                const imageUrl = `${config.API_BASE_URL}/messages/${messageId}/image`;
                updateMessage(messageId, { 
                  isStreaming: false,
                  imageUrl,
                });
                setState(prev => ({ ...prev, isWaitingForResponse: false }));
              }

              if (chunk.imageDone) {
                updateMessage(messageId, { isImageLoading: false });
                return;
              }
            } catch (e) {
              console.error('Failed to parse stream chunk:', e);
            }
          }
        }
      }

      setState(prev => ({ ...prev, isWaitingForResponse: false }));
    } catch (error) {
      if ((error as Error).name !== 'AbortError') {
        console.error('Stream error:', error);
        setState(prev => ({ ...prev, isWaitingForResponse: false }));
      }
    }
  }, [getAccessToken, appendTextToMessage, updateMessage]);

  const startSession = useCallback(async (sessionConfig: GameSessionConfig) => {
    setState(prev => ({ ...prev, phase: 'starting', error: null }));

    try {
      const response = await api.games.sessionsCreate(gameId, {
        shareId: sessionConfig.shareId,
        model: sessionConfig.model,
      });

      const firstMessage = response.data;
      const sceneMessage = mapApiMessageToScene(firstMessage);

      setState(prev => ({
        ...prev,
        phase: 'playing',
        sessionId: firstMessage.gameSessionId || null,
        messages: [{ 
          ...sceneMessage, 
          text: '', 
          isStreaming: true,
          isImageLoading: !!firstMessage.imagePrompt,
        }],
        statusFields: firstMessage.statusFields || [],
        isWaitingForResponse: true,
      }));

      if (firstMessage.id && firstMessage.stream) {
        connectToStream(firstMessage.id);
      } else {
        setState(prev => ({
          ...prev,
          messages: [sceneMessage],
          isWaitingForResponse: false,
        }));
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to start session';
      setState(prev => ({
        ...prev,
        phase: 'error',
        error: message,
      }));
    }
  }, [api, gameId, connectToStream]);

  const sendAction = useCallback(async (message: string) => {
    if (!state.sessionId || state.isWaitingForResponse) return;

    const playerMessage: SceneMessage = {
      id: crypto.randomUUID(),
      type: 'player',
      text: message,
      timestamp: new Date(),
    };

    setState(prev => ({
      ...prev,
      messages: [...prev.messages, playerMessage],
      isWaitingForResponse: true,
    }));

    try {
      const response = await api.sessions.sessionsCreate(state.sessionId, {
        message,
      });

      const gameResponse = response.data;
      const sceneMessage = mapApiMessageToScene(gameResponse);

      setState(prev => ({
        ...prev,
        messages: [...prev.messages, { 
          ...sceneMessage, 
          text: '', 
          isStreaming: true,
          isImageLoading: !!gameResponse.imagePrompt,
        }],
        statusFields: gameResponse.statusFields || prev.statusFields,
      }));

      if (gameResponse.id && gameResponse.stream) {
        connectToStream(gameResponse.id);
      } else {
        setState(prev => ({
          ...prev,
          messages: prev.messages.map(msg =>
            msg.id === sceneMessage.id ? sceneMessage : msg
          ),
          isWaitingForResponse: false,
        }));
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to send action';
      setState(prev => ({
        ...prev,
        isWaitingForResponse: false,
        error: errorMessage,
      }));
    }
  }, [api, state.sessionId, state.isWaitingForResponse, connectToStream]);

  const loadExistingSession = useCallback(async (sessionId: string) => {
    setState(prev => ({ ...prev, phase: 'starting', error: null }));

    try {
      const response = await api.sessions.sessionsDetail(sessionId, { messages: 'all' });
      const session: RoutesSessionResponse = response.data;

      const messages = (session.messages || []).map(mapApiMessageToScene);

      setState(prev => ({
        ...prev,
        phase: 'playing',
        sessionId,
        gameInfo: {
          id: session.gameId,
          name: session.gameName,
          description: session.gameDescription,
        },
        messages,
        statusFields: messages.length > 0 
          ? (messages[messages.length - 1].statusFields || [])
          : [],
        isWaitingForResponse: false,
      }));
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to load session';
      setState(prev => ({
        ...prev,
        phase: 'error',
        error: message,
      }));
    }
  }, [api]);

  const resetGame = useCallback(() => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
    setState(INITIAL_STATE);
  }, []);

  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
    };
  }, []);

  return {
    state,
    startSession,
    sendAction,
    loadExistingSession,
    resetGame,
  };
}

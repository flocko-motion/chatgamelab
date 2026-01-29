/* eslint-disable react-refresh/only-export-components -- Hook and context must be co-located */
import { createContext, useContext, type ReactNode } from 'react';
import type { GamePlayerState, GameSessionConfig, GameTheme } from '../types';

// ============================================================================
// Context Types
// ============================================================================

export type FontSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl';

export interface GamePlayerContextValue {
  state: GamePlayerState;
  theme: GameTheme;
  
  // Actions
  startSession: (config: GameSessionConfig) => Promise<void>;
  sendAction: (message: string) => Promise<void>;
  loadExistingSession: (sessionId: string) => Promise<void>;
  resetGame: () => void;
  
  // Image lightbox
  openLightbox: (imageUrl: string, alt?: string) => void;
  closeLightbox: () => void;
  lightboxImage: { url: string; alt?: string } | null;
  
  // Display settings
  fontSize: FontSize;
  increaseFontSize: () => void;
  decreaseFontSize: () => void;
  debugMode: boolean;
  toggleDebugMode: () => void;
  
  // Image generation
  isImageGenerationDisabled: boolean;
  disableImageGeneration: (errorCode: string) => void;
}

// ============================================================================
// Context
// ============================================================================

const GamePlayerContext = createContext<GamePlayerContextValue | null>(null);

export function useGamePlayerContext(): GamePlayerContextValue {
  const context = useContext(GamePlayerContext);
  if (!context) {
    throw new Error('useGamePlayerContext must be used within a GamePlayerProvider');
  }
  return context;
}

// ============================================================================
// Provider Props
// ============================================================================

export interface GamePlayerProviderProps {
  children: ReactNode;
  value: GamePlayerContextValue;
}

export function GamePlayerProvider({ children, value }: GamePlayerProviderProps) {
  return (
    <GamePlayerContext.Provider value={value}>
      {children}
    </GamePlayerContext.Provider>
  );
}

export { GamePlayerContext };

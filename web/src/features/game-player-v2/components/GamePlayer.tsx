import { useCallback, useEffect, useRef, useState } from 'react';
import {
  Group,
  Text,
  Loader,
  ActionIcon,
  Tooltip,
  Box,
  Stack,
  Button,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { useNavigate } from '@tanstack/react-router';
import { useTranslation } from 'react-i18next';
import { 
  IconArrowLeft, 
  IconAlertCircle,
  IconTextIncrease,
  IconTextDecrease,
  IconSparkles,
} from '@tabler/icons-react';
import { TextButton } from '@components/buttons';
import { ErrorModal } from '@/common/components/ErrorModal';
import { useGame } from '@/api/hooks';
import { useGameSession } from '../hooks/useGameSession';
import { GamePlayerProvider } from '../context';
import type { GamePlayerContextValue, FontSize } from '../context';
import { DEFAULT_THEME, mapApiThemeToPartial } from '../types';
import type { PartialGameTheme } from '../theme/types';
import { GameThemeProvider, useGameTheme } from '../theme';
import { ApiKeySelectModal } from './ApiKeySelectModal';
import { ThemeTestPanel } from './ThemeTestPanel';
import { SceneCard } from './SceneCard';
import { PlayerAction } from './PlayerAction';
import { SystemMessage } from './SystemMessage';
import { SceneDivider } from './SceneDivider';
import { TypingIndicator } from './TypingIndicator';
import { StatusBar } from './StatusBar';
import { PlayerInput } from './PlayerInput';
import { ImageLightbox } from './ImageLightbox';
import { BackgroundAnimation } from './BackgroundAnimation';
import classes from './GamePlayer.module.css';

const FONT_SIZES: FontSize[] = ['xs', 'sm', 'md', 'lg', 'xl', '2xl', '3xl'];

/** Scene area with theme-aware background animation */
interface SceneAreaWithThemeProps {
  renderMessages: () => React.ReactNode[];
  sceneEndRef: React.RefObject<HTMLDivElement | null>;
  animationEnabled: boolean;
}

function SceneAreaWithTheme({ renderMessages, sceneEndRef, animationEnabled }: SceneAreaWithThemeProps) {
  const { cssVars, theme } = useGameTheme();
  const animation = theme.background.animation || 'none';
  const sceneAreaRef = useRef<HTMLDivElement | null>(null);
  
  return (
    <Box 
      className={classes.sceneArea} 
      ref={sceneAreaRef}
      px={{ base: 'sm', sm: 'md' }} 
      py="md"
      style={{ ...cssVars, position: 'relative' }}
    >
      <BackgroundAnimation animation={animation} disabled={!animationEnabled} containerRef={sceneAreaRef} />
      <div className={classes.scenesContainer} style={{ position: 'relative', zIndex: 1 }}>
        {renderMessages()}
        <div ref={sceneEndRef} />
      </div>
    </Box>
  );
}

/** Header with theme-aware styling */
interface HeaderWithThemeProps {
  children: React.ReactNode;
}

function HeaderWithTheme({ children }: HeaderWithThemeProps) {
  const { cssVars } = useGameTheme();
  
  return (
    <Box className={classes.header} px="md" py="sm" style={cssVars}>
      {children}
    </Box>
  );
}

interface GamePlayerProps {
  gameId?: string;
  sessionId?: string;
}

export function GamePlayer({ gameId, sessionId }: GamePlayerProps) {
  const { t } = useTranslation('common');
  const navigate = useNavigate();
  const sceneEndRef = useRef<HTMLDivElement>(null);
  const isContinuation = !!sessionId;

  const [apiKeyModalOpened, { close: closeApiKeyModal }] = useDisclosure(!isContinuation);
  const [lightboxImage, setLightboxImage] = useState<{ url: string; alt?: string } | null>(null);
  const [fontSize, setFontSize] = useState<FontSize>('md');
  const [debugMode, setDebugMode] = useState(false);
  const [animationEnabled, setAnimationEnabled] = useState(true);

  const { data: game, isLoading: gameLoading, error: gameError } = useGame(
    isContinuation ? undefined : gameId
  );
  const { state, startSession, sendAction, loadExistingSession, resetGame } = useGameSession(gameId || '');

  const [themeOverridesBySessionId, setThemeOverridesBySessionId] = useState<Record<string, PartialGameTheme>>({});
  const themeOverride = state.sessionId ? (themeOverridesBySessionId[state.sessionId] ?? null) : null;

  const handleThemeChange = useCallback((theme: PartialGameTheme) => {
    if (!state.sessionId) return;
    setThemeOverridesBySessionId((prev) => ({
      ...prev,
      [state.sessionId as string]: theme,
    }));
  }, [state.sessionId]);

  useEffect(() => {
    if (sessionId && state.phase === 'selecting-key') {
      loadExistingSession(sessionId);
    }
  }, [sessionId, state.phase, loadExistingSession]);

  // Debug: Log received theme
  useEffect(() => {
    if (state.theme) {
      console.log('[GamePlayer] Received theme from session:', JSON.stringify(state.theme, null, 2));
    }
  }, [state.theme]);

  const displayGame = isContinuation ? state.gameInfo : game;
  const isSessionStarting = state.phase === 'starting';

  const scrollToBottom = useCallback(() => {
    setTimeout(() => {
      sceneEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, 100);
  }, []);

  useEffect(() => {
    scrollToBottom();
  }, [state.messages, scrollToBottom]);

  const handleStartGame = async (shareId: string, model?: string) => {
    closeApiKeyModal();
    await startSession({ shareId, model });
  };

  const handleSendAction = async (message: string) => {
    await sendAction(message);
  };

  const handleBack = () => {
    navigate({ to: (isContinuation ? '/sessions' : '/play') as '/' });
  };

  const openLightbox = useCallback((url: string, alt?: string) => {
    setLightboxImage({ url, alt });
  }, []);

  const closeLightbox = useCallback(() => {
    setLightboxImage(null);
  }, []);

  const increaseFontSize = useCallback(() => {
    setFontSize(current => {
      const idx = FONT_SIZES.indexOf(current);
      return idx < FONT_SIZES.length - 1 ? FONT_SIZES[idx + 1] : current;
    });
  }, []);

  const decreaseFontSize = useCallback(() => {
    setFontSize(current => {
      const idx = FONT_SIZES.indexOf(current);
      return idx > 0 ? FONT_SIZES[idx - 1] : current;
    });
  }, []);

  const toggleDebugMode = useCallback(() => {
    setDebugMode(current => !current);
  }, []);

  // Use flex: 1 to fill available space between app header and footer
  const containerHeight = undefined;

  const contextValue: GamePlayerContextValue = {
    state,
    theme: DEFAULT_THEME,
    startSession,
    sendAction,
    loadExistingSession,
    resetGame,
    openLightbox,
    closeLightbox,
    lightboxImage,
    fontSize,
    increaseFontSize,
    decreaseFontSize,
    debugMode,
    toggleDebugMode,
  };

  if (gameLoading || (isContinuation && state.phase === 'selecting-key')) {
    return (
      <Box className={classes.container} h={containerHeight}>
        <Stack className={classes.stateContainer} align="center" justify="center" gap="md">
          <Loader size="lg" color="accent" />
          <Text c="dimmed">{t('gamePlayer.loading.game')}</Text>
        </Stack>
      </Box>
    );
  }

  if (!isContinuation && (gameError || !game)) {
    return (
      <Box className={classes.container} h={containerHeight}>
        <Stack className={classes.stateContainer} align="center" justify="center" gap="md">
          <IconAlertCircle size={48} color="var(--mantine-color-red-5)" />
          <Text size="lg" fw={600}>{t('gamePlayer.error.gameNotFound')}</Text>
          <TextButton onClick={handleBack} leftSection={<IconArrowLeft size={16} />}>
            {t('gamePlayer.error.backToGames')}
          </TextButton>
        </Stack>
      </Box>
    );
  }

  // Validate required game fields before allowing play
  const getMissingFields = () => {
    if (!game || isContinuation) return [];
    const missing: string[] = [];
    if (!game.systemMessageScenario?.trim()) missing.push(t('games.editFields.scenario'));
    if (!game.systemMessageGameStart?.trim()) missing.push(t('games.editFields.gameStart'));
    if (!game.imageStyle?.trim()) missing.push(t('games.editFields.imageStyle'));
    return missing;
  };

  const missingFields = getMissingFields();
  if (!isContinuation && missingFields.length > 0) {
    return (
      <Box className={classes.container} h={containerHeight}>
        <Stack className={classes.stateContainer} align="center" justify="center" gap="md">
          <IconAlertCircle size={48} color="var(--mantine-color-red-5)" />
          <Text size="lg" fw={600}>{t('gamePlayer.error.missingFields')}</Text>
          <Text c="dimmed" ta="center">
            {missingFields.join(', ')}
          </Text>
          <TextButton onClick={handleBack} leftSection={<IconArrowLeft size={16} />}>
            {t('gamePlayer.error.backToGames')}
          </TextButton>
        </Stack>
      </Box>
    );
  }

  if (state.phase === 'error') {
    return (
      <>
        <Box className={classes.container} h={containerHeight}>
          <Stack className={classes.stateContainer} align="center" justify="center" gap="md">
            <Loader size="lg" color="accent" />
          </Stack>
        </Box>
        <ErrorModal
          opened={true}
          onClose={handleBack}
          error={state.errorObject}
          message={!state.errorObject ? state.error || undefined : undefined}
          title={t('gamePlayer.error.sessionFailed')}
        />
      </>
    );
  }

  if (state.phase === 'starting') {
    return (
      <Box className={classes.container} h={containerHeight}>
        <Stack className={classes.stateContainer} align="center" justify="center" gap="md">
          <Loader size="lg" color="accent" />
          <Text fw={600}>{t('gamePlayer.loading.starting')}</Text>
          <Text c="dimmed" size="sm">{t('gamePlayer.loading.startingHint')}</Text>
        </Stack>
      </Box>
    );
  }

  const showImages = true;

  const renderMessages = () => {
    const elements: React.ReactNode[] = [];
    
    state.messages.forEach((message, index) => {
      if (message.type === 'player') {
        elements.push(
          <PlayerAction key={message.id} text={message.text} />
        );
      } else if (message.type === 'system') {
        elements.push(
          <SystemMessage key={message.id} message={message} />
        );
      } else {
        if (index > 0 && state.messages[index - 1]?.type !== 'system') {
          elements.push(<SceneDivider key={`divider-${message.id}`} />);
        }
        elements.push(
          <SceneCard key={message.id} message={message} showImages={showImages} />
        );
      }
    });

    if (state.isWaitingForResponse && state.messages.length > 0) {
      const lastMessage = state.messages[state.messages.length - 1];
      if (lastMessage.type === 'player' || !lastMessage.isStreaming) {
        elements.push(<TypingIndicator key="typing" />);
      }
    }

    // Show input inline when user can type
    if (!state.isWaitingForResponse && state.messages.length > 0) {
      const inputClasses = classes.inlineInput;
      elements.push(
        <div key="inline-input" className={inputClasses}>
          <PlayerInput
            onSend={handleSendAction}
            disabled={state.isWaitingForResponse}
            placeholder={t('gamePlayer.input.placeholder')}
          />
        </div>
      );
    }

    return elements;
  };

  // Deep merge API theme with local override for testing
  const effectiveTheme = themeOverride 
    ? {
        corners: { ...mapApiThemeToPartial(state.theme)?.corners, ...themeOverride.corners },
        background: { ...mapApiThemeToPartial(state.theme)?.background, ...themeOverride.background },
        player: { ...mapApiThemeToPartial(state.theme)?.player, ...themeOverride.player },
        gameMessage: { ...mapApiThemeToPartial(state.theme)?.gameMessage, ...themeOverride.gameMessage },
        cards: { ...mapApiThemeToPartial(state.theme)?.cards, ...themeOverride.cards },
        thinking: { ...mapApiThemeToPartial(state.theme)?.thinking, ...themeOverride.thinking },
        typography: { ...mapApiThemeToPartial(state.theme)?.typography, ...themeOverride.typography },
        statusFields: { ...mapApiThemeToPartial(state.theme)?.statusFields, ...themeOverride.statusFields },
        header: { ...mapApiThemeToPartial(state.theme)?.header, ...themeOverride.header },
        divider: { ...mapApiThemeToPartial(state.theme)?.divider, ...themeOverride.divider },
        statusEmojis: { ...mapApiThemeToPartial(state.theme)?.statusEmojis, ...themeOverride.statusEmojis },
      }
    : mapApiThemeToPartial(state.theme);
  
  return (
    <GameThemeProvider theme={effectiveTheme}>
    <GamePlayerProvider value={contextValue}>
      <Box className={classes.container} h={containerHeight}>
        <HeaderWithTheme>
          <Group justify="space-between" wrap="nowrap">
            <Group gap="sm" wrap="nowrap" style={{ minWidth: 0, flex: 1 }}>
              <Tooltip label={t('gamePlayer.header.back')} position="bottom">
                <ActionIcon
                  variant="subtle"
                  color="gray"
                  onClick={handleBack}
                  aria-label={t('gamePlayer.header.back')}
                  size="lg"
                >
                  <IconArrowLeft size={20} />
                </ActionIcon>
              </Tooltip>
              <Box style={{ minWidth: 0, flex: 1 }}>
                <Text fw={600} truncate size="sm">{displayGame?.name || t('gamePlayer.unnamed')}</Text>
                {displayGame?.description && (
                  <Text size="xs" truncate className={classes.headerDescription}>{displayGame.description}</Text>
                )}
              </Box>
            </Group>
            <Group gap="xs" wrap="nowrap">
              <Tooltip label={t('gamePlayer.header.decreaseFont')} position="bottom">
                <ActionIcon
                  variant="subtle"
                  color="gray"
                  onClick={decreaseFontSize}
                  disabled={fontSize === 'xs'}
                  aria-label={t('gamePlayer.header.decreaseFont')}
                  size="lg"
                >
                  <IconTextDecrease size={18} />
                </ActionIcon>
              </Tooltip>
              <Tooltip label={t('gamePlayer.header.increaseFont')} position="bottom">
                <ActionIcon
                  variant="subtle"
                  color="gray"
                  onClick={increaseFontSize}
                  disabled={fontSize === '3xl'}
                  aria-label={t('gamePlayer.header.increaseFont')}
                  size="lg"
                >
                  <IconTextIncrease size={18} />
                </ActionIcon>
              </Tooltip>
              <Tooltip label={animationEnabled ? t('gamePlayer.header.disableAnimation') : t('gamePlayer.header.enableAnimation')} position="bottom">
                <ActionIcon
                  variant={animationEnabled ? 'light' : 'subtle'}
                  color={animationEnabled ? 'violet' : 'gray'}
                  onClick={() => setAnimationEnabled(!animationEnabled)}
                  aria-label={animationEnabled ? t('gamePlayer.header.disableAnimation') : t('gamePlayer.header.enableAnimation')}
                  size="lg"
                >
                  <IconSparkles size={18} />
                </ActionIcon>
              </Tooltip>
              <ThemeTestPanel
                currentTheme={effectiveTheme}
                onThemeChange={handleThemeChange}
              />
              <Button
                onClick={toggleDebugMode}
                size="xs"
                variant={debugMode ? 'light' : 'subtle'}
                color={debugMode ? 'accent' : 'gray'}
                radius="md"
              >
                {t('gamePlayer.header.debug')}
              </Button>
            </Group>
          </Group>
        </HeaderWithTheme>

        <StatusBar statusFields={state.statusFields} />

        <SceneAreaWithTheme
          renderMessages={renderMessages}
          sceneEndRef={sceneEndRef}
          animationEnabled={animationEnabled && !state.isWaitingForResponse}
        />

        {!isContinuation && (
          <ApiKeySelectModal
            opened={apiKeyModalOpened}
            onClose={handleBack}
            onStart={handleStartGame}
            gameName={displayGame?.name}
            isLoading={isSessionStarting}
          />
        )}

        <ImageLightbox />
      </Box>
    </GamePlayerProvider>
    </GameThemeProvider>
  );
}

import { useEffect, useRef, useState, useCallback } from 'react';
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
} from '@tabler/icons-react';
import { TextButton } from '@components/buttons';
import { useGame } from '@/api/hooks';
import { useGameSession } from '../hooks/useGameSession';
import { GamePlayerProvider } from '../context';
import type { GamePlayerContextValue, FontSize } from '../context';
import { DEFAULT_THEME } from '../types';
import { ApiKeySelectModal } from './ApiKeySelectModal';
import { SceneCard } from './SceneCard';
import { PlayerAction } from './PlayerAction';
import { SystemMessage } from './SystemMessage';
import { SceneDivider } from './SceneDivider';
import { TypingIndicator } from './TypingIndicator';
import { StatusBar } from './StatusBar';
import { PlayerInput } from './PlayerInput';
import { ImageLightbox } from './ImageLightbox';
import classes from './GamePlayer.module.css';

const FONT_SIZES: FontSize[] = ['xs', 'sm', 'md', 'lg', 'xl', '2xl', '3xl'];

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

  const { data: game, isLoading: gameLoading, error: gameError } = useGame(
    isContinuation ? undefined : gameId
  );
  const { state, startSession, sendAction, loadExistingSession, resetGame } = useGameSession(gameId || '');

  useEffect(() => {
    if (sessionId && state.phase === 'selecting-key') {
      loadExistingSession(sessionId);
    }
  }, [sessionId, state.phase, loadExistingSession]);

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

  if (state.phase === 'error') {
    return (
      <Box className={classes.container} h={containerHeight}>
        <Stack className={classes.stateContainer} align="center" justify="center" gap="md">
          <IconAlertCircle size={48} color="var(--mantine-color-red-5)" />
          <Text size="lg" fw={600}>{t('gamePlayer.error.sessionFailed')}</Text>
          <Text c="dimmed" maw={400}>{state.error}</Text>
          <TextButton onClick={handleBack} leftSection={<IconArrowLeft size={16} />}>
            {t('gamePlayer.error.backToGames')}
          </TextButton>
        </Stack>
      </Box>
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
      elements.push(
        <div key="inline-input" className={classes.inlineInput}>
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

  return (
    <GamePlayerProvider value={contextValue}>
      <Box className={classes.container} h={containerHeight}>
        <Box className={classes.header} px="md" py="sm">
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
                  <Text size="xs" c="dimmed" truncate>{displayGame.description}</Text>
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
        </Box>

        <StatusBar statusFields={state.statusFields} />

        <Box className={classes.sceneArea} px={{ base: 'sm', sm: 'md' }} py="md">
          <div className={classes.scenesContainer}>
            {renderMessages()}
            <div ref={sceneEndRef} />
          </div>
        </Box>

        {!isContinuation && (
          <ApiKeySelectModal
            opened={apiKeyModalOpened}
            onClose={closeApiKeyModal}
            onStart={handleStartGame}
            gameName={displayGame?.name}
            isLoading={isSessionStarting}
          />
        )}

        <ImageLightbox />
      </Box>
    </GamePlayerProvider>
  );
}

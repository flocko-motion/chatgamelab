import { useEffect, useRef } from 'react';
import {
  Group,
  Text,
  Loader,
  ActionIcon,
  Tooltip,
  Box,
  Stack,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { useNavigate } from '@tanstack/react-router';
import { useTranslation } from 'react-i18next';
import {
  IconArrowLeft,
  IconAlertCircle,
  IconEye,
  IconEyeOff,
} from '@tabler/icons-react';
import { TextButton } from '@components/buttons';
import { useGame } from '@/api/hooks';
import { useGameSession } from '../hooks/useGameSession';
import { ApiKeySelectModal } from './ApiKeySelectModal';
import { ChatMessage } from './ChatMessage';
import { StatusBar } from './StatusBar';
import { PlayerInput } from './PlayerInput';
import classes from './GamePlayer.module.css';

interface GamePlayerProps {
  gameId?: string;
  sessionId?: string;
}

export function GamePlayer({ gameId, sessionId }: GamePlayerProps) {
  const { t } = useTranslation('common');
  const navigate = useNavigate();
  const chatEndRef = useRef<HTMLDivElement>(null);
  const isContinuation = !!sessionId;

  const [debugMode, { toggle: toggleDebug }] = useDisclosure(false);
  const [apiKeyModalOpened, { close: closeApiKeyModal }] = useDisclosure(!isContinuation);

  // For new games, fetch game data; for continuation, game info comes from session
  const { data: game, isLoading: gameLoading, error: gameError } = useGame(
    isContinuation ? undefined : gameId
  );
  const { state, startSession, sendAction, loadExistingSession } = useGameSession(gameId || '');

  // Load existing session on mount if sessionId is provided
  useEffect(() => {
    if (sessionId && state.phase === 'selecting-key') {
      loadExistingSession(sessionId);
    }
  }, [sessionId, state.phase, loadExistingSession]);

  // Use game data from API for new games, or from loaded session for continuation
  const displayGame = isContinuation ? state.gameInfo : game;

  const isSessionStarting = state.phase === 'starting';

  const scrollToBottom = () => {
    setTimeout(() => {
      chatEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, 100);
  };

  useEffect(() => {
    scrollToBottom();
  }, [state.messages]);

  const handleStartGame = async (shareId: string, model?: string) => {
    closeApiKeyModal();
    await startSession({ shareId, model });
  };

  const handleSendAction = async (message: string) => {
    await sendAction(message);
  };

  const handleBack = () => {
    // Go back to sessions list for continuation, play list for new games
    navigate({ to: (isContinuation ? '/sessions' : '/play') as '/' });
  };


  // Height calculation for viewport - accounts for header
  const containerHeight = 'calc(100vh - 220px)';

  // Show loading for new games or when loading existing session
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

  // For new games, check for game load error; for continuation, game info comes later
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

  // Determine if session has images (for now, assume true - can be session config later)
  const showImages = true;

  return (
    <Box className={classes.container} h={containerHeight}>
      {/* Header */}
      <Box className={classes.header} px={{ base: 'sm', sm: 'md' }} py="sm">
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
            <Tooltip label={debugMode ? t('gamePlayer.header.hideDebug') : t('gamePlayer.header.showDebug')} position="bottom">
              <ActionIcon
                variant={debugMode ? 'light' : 'subtle'}
                color={debugMode ? 'accent' : 'gray'}
                onClick={toggleDebug}
                aria-label={debugMode ? t('gamePlayer.header.hideDebug') : t('gamePlayer.header.showDebug')}
                size="lg"
              >
                {debugMode ? <IconEyeOff size={18} /> : <IconEye size={18} />}
              </ActionIcon>
            </Tooltip>
          </Group>
        </Group>
      </Box>

      {/* Status Bar */}
      <StatusBar statusFields={state.statusFields} />

      {/* Chat Area */}
      <Box className={classes.chatArea} px={{ base: 'sm', sm: 'md' }} py="md">
        <div className={classes.messagesContainer}>
          {state.messages.map((message) => (
            <ChatMessage key={message.id} message={message} showImages={showImages} />
          ))}
          <div ref={chatEndRef} />
        </div>
      </Box>

      {/* Input Area */}
      <Box className={classes.inputArea} px={{ base: 'sm', sm: 'md' }} py="sm">
        <PlayerInput
          onSend={handleSendAction}
          disabled={state.isWaitingForResponse}
          placeholder={
            state.isWaitingForResponse
              ? t('gamePlayer.input.waiting')
              : t('gamePlayer.input.placeholder')
          }
        />
      </Box>

      {/* API Key Selection Modal - only for new games */}
      {!isContinuation && (
        <ApiKeySelectModal
          opened={apiKeyModalOpened}
          onClose={closeApiKeyModal}
          onStart={handleStartGame}
          gameName={displayGame?.name}
          isLoading={isSessionStarting}
        />
      )}
    </Box>
  );
}

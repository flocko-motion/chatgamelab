import { useState, useCallback } from 'react';
import { Group, TextInput, ActionIcon, Text, Progress } from '@mantine/core';
import { useTranslation } from 'react-i18next';
import { IconPlayerPlay, IconMicrophone } from '@tabler/icons-react';
import { useGameTheme, THEME_COLORS, CARD_BG_COLORS, FONT_COLORS } from '../theme';
import { useAudioRecorder } from '../hooks/useAudioRecorder';
import type { PlayerActionInput } from '../types';

interface PlayerInputProps {
  onSend: (input: PlayerActionInput) => void;
  disabled?: boolean;
  placeholder?: string;
  audioEnabled?: boolean;
}

export function PlayerInput({ onSend, disabled = false, placeholder, audioEnabled = false }: PlayerInputProps) {
  const { t } = useTranslation('common');
  const { theme } = useGameTheme();
  const [value, setValue] = useState('');

  const accentColor = THEME_COLORS[theme.corners.color] || THEME_COLORS.amber;
  const playerBgColor = CARD_BG_COLORS[theme.player.bgColor] || CARD_BG_COLORS.creme;
  const playerFontColor = FONT_COLORS[theme.player.fontColor] || FONT_COLORS.dark;
  const playerBorderColor = THEME_COLORS[theme.player.borderColor] || THEME_COLORS.cyan;

  const handleAudioComplete = useCallback(
    (audio: { base64: string; mimeType: string }) => {
      onSend({
        audioBase64: audio.base64,
        audioMimeType: audio.mimeType,
      });
    },
    [onSend],
  );

  const recorder = useAudioRecorder(handleAudioComplete);
  const isRecording = recorder.state === 'recording';
  const isProcessing = recorder.state === 'processing' || recorder.state === 'requesting';

  const handleSend = useCallback(() => {
    const trimmed = value.trim();
    if (!trimmed || disabled) return;
    onSend({ message: trimmed });
    setValue('');
  }, [value, disabled, onSend]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        handleSend();
      }
    },
    [handleSend]
  );

  const isActive = isRecording || isProcessing;
  const progressPct = isRecording ? (recorder.elapsed / recorder.maxDuration) * 100 : 0;
  const elapsedDisplay = Math.floor(recorder.elapsed);

  return (
    <Group gap="sm" wrap="nowrap">
      {audioEnabled && recorder.isSupported && (
        <ActionIcon
          variant={isActive ? 'filled' : 'subtle'}
          size="lg"
          radius="xl"
          onPointerDown={(e) => {
            e.preventDefault();
            if (!disabled && !isActive) recorder.startRecording();
          }}
          onPointerUp={() => { if (isRecording) recorder.stopRecording(); }}
          onPointerLeave={() => { if (isRecording) recorder.stopRecording(); }}
          onContextMenu={(e) => e.preventDefault()}
          disabled={disabled || isProcessing}
          aria-label={isActive ? t('gamePlayer.input.audioRecording') : t('gamePlayer.input.audioStart')}
          style={{
            background: isActive ? 'var(--mantine-color-red-6)' : undefined,
            color: isActive ? 'white' : playerFontColor,
            opacity: disabled ? 0.55 : 1,
            touchAction: 'none',
            transition: 'background 0.15s, color 0.15s',
          }}
        >
          <IconMicrophone size={20} />
        </ActionIcon>
      )}

      {isActive ? (
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 4 }}>
          <Group gap="xs" wrap="nowrap" align="center">
            <Text size="sm" fw={500}>
              {isProcessing
                ? t('gamePlayer.input.audioProcessing')
                : t('gamePlayer.input.audioRecording')}
            </Text>
            {isRecording && (
              <Text size="xs" c="dimmed" ml="auto">
                {elapsedDisplay}s / {recorder.maxDuration}s
              </Text>
            )}
          </Group>
          {isRecording && (
            <Progress
              value={progressPct}
              size="xs"
              radius="xl"
              color={progressPct > 80 ? 'red' : accentColor.primary}
              animated
            />
          )}
        </div>
      ) : (
        <TextInput
          value={value}
          onChange={(e) => setValue(e.currentTarget.value)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder || t('gamePlayer.input.placeholder')}
          disabled={disabled}
          size="md"
          radius="xl"
          styles={{
            root: { flex: 1 },
            input: {
              '--input-bd-focus': accentColor.primary,
            },
          }}
          rightSection={
            <ActionIcon
              variant="filled"
              size="md"
              radius="xl"
              onClick={handleSend}
              disabled={disabled || !value.trim()}
              aria-label={t('gamePlayer.input.send')}
              style={{
                background: playerBgColor.solid,
                color: playerFontColor,
                border: `1px solid ${playerBorderColor.primary}`,
                opacity: disabled || !value.trim() ? 0.55 : 1,
              }}
            >
              <IconPlayerPlay size={16} />
            </ActionIcon>
          }
          rightSectionWidth={42}
        />
      )}
    </Group>
  );
}

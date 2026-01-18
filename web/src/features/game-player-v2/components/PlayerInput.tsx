import { useState, useCallback } from 'react';
import { Group, TextInput, ActionIcon } from '@mantine/core';
import { useTranslation } from 'react-i18next';
import { IconPlayerPlay } from '@tabler/icons-react';
import { useGameTheme, THEME_COLORS } from '../theme';

interface PlayerInputProps {
  onSend: (message: string) => void;
  disabled?: boolean;
  placeholder?: string;
}

export function PlayerInput({ onSend, disabled = false, placeholder }: PlayerInputProps) {
  const { t } = useTranslation('common');
  const { theme } = useGameTheme();
  const [value, setValue] = useState('');
  
  const accentColor = THEME_COLORS[theme.corners.color] || THEME_COLORS.amber;

  const handleSend = useCallback(() => {
    const trimmed = value.trim();
    if (!trimmed || disabled) return;
    onSend(trimmed);
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

  return (
    <Group gap="sm" wrap="nowrap">
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
              background: disabled || !value.trim() 
                ? 'var(--mantine-color-gray-4)' 
                : accentColor.dark,
            }}
          >
            <IconPlayerPlay size={16} />
          </ActionIcon>
        }
        rightSectionWidth={42}
      />
    </Group>
  );
}

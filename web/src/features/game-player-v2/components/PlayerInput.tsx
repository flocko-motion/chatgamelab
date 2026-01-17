import { useState, useCallback } from 'react';
import { Group, TextInput, ActionIcon } from '@mantine/core';
import { useTranslation } from 'react-i18next';
import { IconSend } from '@tabler/icons-react';

interface PlayerInputProps {
  onSend: (message: string) => void;
  disabled?: boolean;
  placeholder?: string;
}

export function PlayerInput({ onSend, disabled = false, placeholder }: PlayerInputProps) {
  const { t } = useTranslation('common');
  const [value, setValue] = useState('');

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
        style={{ flex: 1 }}
        size="md"
        radius="xl"
        rightSection={
          <ActionIcon
            variant="filled"
            color="accent"
            size="md"
            radius="xl"
            onClick={handleSend}
            disabled={disabled || !value.trim()}
            aria-label={t('gamePlayer.input.send')}
          >
            <IconSend size={16} />
          </ActionIcon>
        }
        rightSectionWidth={42}
      />
    </Group>
  );
}

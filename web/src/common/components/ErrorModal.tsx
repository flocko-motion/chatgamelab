import { Modal, Stack, Text, Button, Group, ThemeIcon } from '@mantine/core';
import { IconAlertTriangle, IconX } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { translateError, translateErrorCode } from '../lib/errorHelpers';
import { TextWithLinks } from './TextWithLinks';
import type { ErrorCode } from '../types/errorCodes';

export interface ErrorModalProps {
  opened: boolean;
  onClose: () => void;
  /** Pass the raw error object from the API - handles everything automatically */
  error?: unknown;
  /** Or pass a known error code string directly */
  errorCode?: ErrorCode | string;
  /** Override the title */
  title?: string;
  /** Override the message */
  message?: string;
  onRetry?: () => void;
}

/**
 * Reusable modal for displaying backend errors.
 * 
 * Usage:
 * ```tsx
 * // Simply pass the API error - handles everything automatically
 * <ErrorModal opened={!!error} onClose={clearError} error={apiError} />
 * 
 * // Or pass an error code string
 * <ErrorModal opened={opened} onClose={close} errorCode="validation_error" />
 * 
 * // Or use custom title/message
 * <ErrorModal opened={opened} onClose={close} title="Oops" message="Something broke" />
 * ```
 */
export function ErrorModal({
  opened,
  onClose,
  error,
  errorCode,
  title,
  message,
  onRetry,
}: ErrorModalProps) {
  const { t } = useTranslation();

  // Resolve error to translated text
  const translated = error 
    ? translateError(error) 
    : errorCode 
      ? translateErrorCode(errorCode) 
      : null;

  // Use provided overrides or translated values
  const displayTitle = title ?? translated?.title ?? t('errors:titles.error');
  const displayMessage = message ?? translated?.message ?? t('errors:generic');
  const color = translated?.color ?? 'red';

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={
        <Group gap="xs">
          <ThemeIcon color={color} variant="light" size="sm">
            <IconAlertTriangle size={14} />
          </ThemeIcon>
          <Text fw={600}>{displayTitle}</Text>
        </Group>
      }
      centered
      size="md"
    >
      <Stack gap="md">
        <TextWithLinks size="sm" c="dimmed">
          {displayMessage}
        </TextWithLinks>

        {translated?.code && (
          <Text size="xs" c="dimmed" ff="monospace">
            ERROR_CODE: {translated.code}
          </Text>
        )}

        <Group justify="flex-end" gap="sm">
          {onRetry && (
            <Button variant="light" onClick={onRetry}>
              {t('common:tryAgain', { defaultValue: 'Try Again' })}
            </Button>
          )}
          <Button
            variant="filled"
            color={color}
            leftSection={<IconX size={16} />}
            onClick={onClose}
          >
            {t('common:close')}
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}

import { Button, type ButtonProps } from '@mantine/core';
import { IconPlayerPlay } from '@tabler/icons-react';
import { forwardRef } from 'react';

export interface PlayGameButtonProps extends Omit<ButtonProps, 'variant' | 'gradient' | 'leftSection'> {
  onClick?: () => void;
}

/**
 * PlayGameButton - Prominent button for play/continue actions
 * 
 * USE WHEN:
 * - Starting a new game ("Play now")
 * - Continuing an existing session ("Continue Game")
 * 
 * Features a gradient background using accent (cyan) and highlight (magenta) colors
 * for maximum visual prominence as the primary play action.
 * 
 * @example
 * <PlayGameButton onClick={handlePlay}>{t('play.playNow')}</PlayGameButton>
 * <PlayGameButton onClick={handleContinue}>{t('sessions.continueGame')}</PlayGameButton>
 */
export const PlayGameButton = forwardRef<HTMLButtonElement, PlayGameButtonProps>(
  function PlayGameButton({ children, size = 'xs', ...props }, ref) {
    return (
      <Button
        ref={ref}
        color="highlight.5"
        size={size}
        radius="md"
        leftSection={<IconPlayerPlay size={14} />}
        style={{
          transition: 'transform 0.15s ease, box-shadow 0.15s ease',
        }}
        styles={{
          root: {
            '&:hover:not(:disabled)': {
              transform: 'scale(1.02)',
              boxShadow: '0 4px 12px rgba(199, 45, 133, 0.3)',
            },
          },
        }}
        {...props}
      >
        {children}
      </Button>
    );
  }
);

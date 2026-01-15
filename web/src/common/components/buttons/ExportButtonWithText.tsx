import { Button, type ButtonProps } from '@mantine/core';
import { IconDownload } from '@tabler/icons-react';

export interface ExportButtonWithTextProps extends Omit<ButtonProps, 'leftSection' | 'variant' | 'color'> {
  onClick?: () => void;
}

/**
 * ExportButtonWithText - Button with download/export icon and customizable text
 * 
 * USE WHEN:
 * - Export actions that need descriptive text (e.g., "Export Game", "Download")
 * 
 * @example
 * <ExportButtonWithText onClick={handleExport}>Export Game</ExportButtonWithText>
 */
export function ExportButtonWithText({
  children,
  size = 'xs',
  ...props
}: ExportButtonWithTextProps) {
  return (
    <Button
      variant="subtle"
      color="accent.9"
      size={size}
      radius="md"
      leftSection={<IconDownload size={14} />}
      {...props}
    >
      {children}
    </Button>
  );
}

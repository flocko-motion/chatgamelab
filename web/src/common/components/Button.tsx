import { Button as MantineButton, type ButtonProps as MantineButtonProps } from '@mantine/core';

export interface ButtonProps extends Omit<MantineButtonProps, 'color' | 'onClick'> {
  variant?: 'primary' | 'secondary' | 'danger';
  fullWidth?: boolean;
  onClick?: () => void;
}

export function Button({ variant = 'primary', fullWidth = false, ...props }: ButtonProps) {
  const getVariantStyles = () => {
    switch (variant) {
      case 'primary':
        return {
          color: 'violet' as const,
          radius: 'md' as const,
        };
      case 'secondary':
        return {
          color: 'gray' as const,
          variant: 'outline' as const,
          radius: 'md' as const,
        };
      case 'danger':
        return {
          color: 'red' as const,
          radius: 'md' as const,
        };
      default:
        return {
          color: 'violet' as const,
          radius: 'md' as const,
        };
    }
  };

  return (
    <MantineButton
      {...getVariantStyles()}
      {...props}
      style={{ width: fullWidth ? '100%' : undefined, ...props.style }}
    />
  );
}

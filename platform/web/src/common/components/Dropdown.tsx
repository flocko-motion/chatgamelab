import { Select as MantineSelect, type SelectProps as MantineSelectProps } from '@mantine/core';

export interface DropdownProps extends Omit<MantineSelectProps, 'size' | 'variant'> {
  variant?: 'default' | 'filled' | 'outline' | 'subtle' | 'light';
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  fullWidth?: boolean;
}

export function Dropdown({ variant = 'default', size = 'sm', fullWidth = false, ...props }: DropdownProps) {
  const getVariantStyles = () => {
    switch (variant) {
      case 'default':
        return {
          variant: 'default' as const,
        };
      case 'filled':
        return {
          variant: 'filled' as const,
        };
      case 'outline':
        return {
          variant: 'outline' as const,
        };
      case 'subtle':
        return {
          variant: 'subtle' as const,
        };
      case 'light':
        return {
          variant: 'light' as const,
        };
      default:
        return {
          variant: 'default' as const,
        };
    }
  };

  return (
    <MantineSelect
      size={size}
      {...getVariantStyles()}
      {...props}
      style={{ width: fullWidth ? '100%' : undefined, ...props.style }}
      styles={{
        input: {
          minWidth: 60,
        },
        ...props.styles,
      }}
    />
  );
}

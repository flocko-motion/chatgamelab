import { Button as MantineButton, type ButtonProps as MantineButtonProps } from '@mantine/core';

/**
 * Custom Button component with semantic variants.
 * 
 * Uses Mantine's autoContrast feature for readable text on colored backgrounds.
 * No need for manual textColor - the theme handles contrast automatically.
 * 
 * Variants:
 * - primary: Filled accent button (default)
 * - secondary: Outline gray button
 * - danger: Filled red button
 */
export interface ButtonProps extends Omit<MantineButtonProps, 'onClick'> {
  variant?: 'primary' | 'secondary' | 'danger' | MantineButtonProps['variant'];
  fullWidth?: boolean;
  onClick?: () => void;
}

export function Button({ variant = 'primary', color, fullWidth = false, ...props }: ButtonProps) {
  // Map semantic variants to Mantine props
  const getMantineProps = (): Partial<MantineButtonProps> => {
    switch (variant) {
      case 'primary':
        return {
          color: color || 'accent',
          variant: 'filled',
        };
      case 'secondary':
        return {
          color: color || 'gray',
          variant: 'outline',
        };
      case 'danger':
        return {
          color: color || 'red',
          variant: 'filled',
        };
      default:
        // Pass through Mantine variants directly (light, subtle, etc.)
        return {
          color: color || 'accent',
          variant: variant as MantineButtonProps['variant'],
        };
    }
  };

  return (
    <MantineButton
      {...getMantineProps()}
      {...props}
      style={{ 
        width: fullWidth ? '100%' : undefined, 
        ...props.style 
      }}
    />
  );
}

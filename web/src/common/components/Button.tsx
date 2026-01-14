import { Button as MantineButton, type ButtonProps as MantineButtonProps } from '@mantine/core';

export interface ButtonProps extends Omit<MantineButtonProps, 'color' | 'onClick'> {
  variant?: 'primary' | 'secondary' | 'danger';
  color?: MantineButtonProps['color'];
  textColor?: string;
  fullWidth?: boolean;
  onClick?: () => void;
}

export function Button({ variant = 'primary', color, textColor, fullWidth = false, ...props }: ButtonProps) {
  const getVariantStyles = () => {
    // Use provided color prop, otherwise default to accent for primary
    const buttonColor = color || (variant === 'primary' ? 'accent' : 
                                 variant === 'secondary' ? 'gray' : 
                                 variant === 'danger' ? 'red' : 'accent');
    
    switch (variant) {
      case 'primary':
        return {
          color: buttonColor,
          radius: 'md' as const,
        };
      case 'secondary':
        return {
          color: buttonColor,
          variant: 'outline' as const,
          radius: 'md' as const,
        };
      case 'danger':
        return {
          color: buttonColor,
          radius: 'md' as const,
        };
      default:
        return {
          color: buttonColor,
          radius: 'md' as const,
        };
    }
  };

  return (
    <MantineButton
      {...getVariantStyles()}
      {...props}
      style={{ 
        width: fullWidth ? '100%' : undefined, 
        color: textColor || undefined,
        ...props.style 
      }}
    />
  );
}

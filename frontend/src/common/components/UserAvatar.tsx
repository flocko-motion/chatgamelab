import { Avatar } from '@mantine/core';
import { getUserInitials, getUserAvatarColor, shouldUseWhiteText } from '@/common/lib/userUtils';

export interface UserAvatarProps {
  /** User's name to generate initials and color */
  name: string;
  /** Avatar size - defaults to 'md' */
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  /** Custom style overrides */
  style?: React.CSSProperties;
  /** Additional CSS classes */
  className?: string;
  /** Whether to show the initials (default) or fallback to icon */
  showInitials?: boolean;
}

/**
 * UserAvatar component that generates initials and consistent colors
 * 
 * Examples:
 * - "Janos" -> Shows "J" with consistent color
 * - "Janos Brodbeck" -> Shows "JB" with consistent color  
 * - "Florian Blabla Metzger Nuger" -> Shows "FB" with consistent color
 */
export function UserAvatar({
  name,
  size = 'md',
  style,
  className,
  showInitials = true,
}: UserAvatarProps) {
  const initials = getUserInitials(name);
  const color = getUserAvatarColor(name);
  const textColor = shouldUseWhiteText(color) ? 'white' : 'dark';

  // If style has transparent background, use white text for header
  const isTransparentBackground = style?.backgroundColor === 'transparent';
  const finalTextColor = isTransparentBackground ? 'white' : textColor;

  return (
    <Avatar
      size={size}
      radius="xl"
      color={color}
      style={style}
      className={className}
      c={finalTextColor}
    >
      {showInitials ? initials : null}
    </Avatar>
  );
}

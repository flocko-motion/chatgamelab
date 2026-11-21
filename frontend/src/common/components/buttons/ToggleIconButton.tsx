import { ActionIcon, Tooltip } from '@mantine/core';
import type { ReactNode } from 'react';

/**
 * ToggleIconButton - An icon button that switches something on or off, such
 * as a filter ("only favorites"). It shows `activeIcon` while on and tells
 * screen readers its state (aria-pressed).
 *
 * USE WHEN:
 * - A setting is on or off and the icon alone says which
 *
 * @example
 * <ToggleIconButton
 *   active={onlyFavorites}
 *   onToggle={toggleFavorites}
 *   icon={<IconStar size={18} />}
 *   activeIcon={<IconStarFilled size={18} />}
 *   label={t('allGames.onlyFavorites')}
 * />
 */
export interface ToggleIconButtonProps {
  active: boolean;
  onToggle: () => void;
  icon: ReactNode;
  activeIcon: ReactNode;
  /** What the toggle does; the tooltip and the accessible name. */
  label: string;
  size?: 'xs' | 'sm' | 'md' | 'lg';
}

export function ToggleIconButton({
  active,
  onToggle,
  icon,
  activeIcon,
  label,
  size = 'md',
}: ToggleIconButtonProps) {
  return (
    <Tooltip label={label} withArrow>
      <ActionIcon
        variant={active ? 'light' : 'subtle'}
        color={active ? 'accent' : 'gray'}
        size={size}
        radius="md"
        onClick={onToggle}
        aria-label={label}
        aria-pressed={active}
      >
        {active ? activeIcon : icon}
      </ActionIcon>
    </Tooltip>
  );
}

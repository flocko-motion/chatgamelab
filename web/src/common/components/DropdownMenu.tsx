import {
  Menu,
  MenuTarget,
  MenuDropdown,
  MenuItem,
  useMantineTheme,
  type MenuItemProps,
} from '@mantine/core';
import type { ReactNode } from 'react';

export interface DropdownMenuItem extends Omit<MenuItemProps, 'children' | 'key'> {
  key: string;
  label: ReactNode;
  icon?: ReactNode;
  onClick?: () => void;
  href?: string;
  danger?: boolean;
}

export interface DropdownMenuProps {
  trigger: ReactNode;
  items: DropdownMenuItem[];
  position?: 'bottom' | 'left' | 'right' | 'top';
  offset?: number;
  triggerAction?: 'click' | 'hover' | 'click-hover';
  withArrow?: boolean;
  shadow?: string;
}

export function DropdownMenu({
  trigger,
  items,
  position = 'bottom',
  offset = 8,
  triggerAction = 'click',
  withArrow = true,
  shadow = 'md',
}: DropdownMenuProps) {
  const theme = useMantineTheme();

  return (
    <Menu
      position={position}
      offset={offset}
      trigger={triggerAction}
      withArrow={withArrow}
      shadow={shadow}
      styles={{
        dropdown: {
          backgroundColor: theme.other.layout.panelBg,
          borderColor: theme.other.layout.lineLight,
        },
        arrow: {
          backgroundColor: theme.other.layout.panelBg,
          borderColor: theme.other.layout.lineLight,
        },
        item: {
          color: 'white',
          '&:hover': {
            backgroundColor: theme.other.layout.bgHover,
          },
          '&:active': {
            backgroundColor: theme.other.layout.bgActive,
          },
        },
      }}
    >
      <MenuTarget>{trigger}</MenuTarget>
      <MenuDropdown>
        {items.map((item) => (
          <MenuItem
            key={item.key}
            leftSection={item.icon}
            onClick={item.onClick}
            component={item.href ? 'a' : undefined}
            href={item.href}
            target={item.href ? '_blank' : undefined}
            rel={item.href ? 'noopener noreferrer' : undefined}
            styles={{
              itemLabel: {
                color: item.danger ? theme.colors.red[6] : 'white',
              },
            }}
          >
            {item.label}
          </MenuItem>
        ))}
      </MenuDropdown>
    </Menu>
  );
}

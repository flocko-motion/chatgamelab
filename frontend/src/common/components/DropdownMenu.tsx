import {
  Menu,
  MenuTarget,
  MenuDropdown,
  MenuItem,
  type MenuItemProps,
} from '@mantine/core';
import type { ReactNode } from 'react';
import classes from './DropdownMenu.module.css';

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
  return (
    <Menu
      position={position}
      offset={offset}
      trigger={triggerAction}
      withArrow={withArrow}
      shadow={shadow}
      classNames={{
        dropdown: classes.dropdown,
        arrow: classes.arrow,
        item: classes.item,
        itemLabel: classes.itemLabel,
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
            className={item.danger ? classes.itemDanger : undefined}
          >
            {item.label}
          </MenuItem>
        ))}
      </MenuDropdown>
    </Menu>
  );
}

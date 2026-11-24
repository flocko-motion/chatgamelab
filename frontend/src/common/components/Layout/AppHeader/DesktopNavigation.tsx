import { Group, UnstyledButton } from "@mantine/core";
import { IconChevronDown } from "@tabler/icons-react";
import { DropdownMenu } from "../../DropdownMenu";
import type { NavItem } from "./types";
import classes from "./DesktopNavigation.module.css";

function NavButton({ item }: { item: NavItem }) {
  return (
    <UnstyledButton
      onClick={item.onClick}
      className={classes.navButton}
      data-active={item.active || undefined}
      aria-current={item.active ? "page" : undefined}
    >
      {item.icon}
      <span className={classes.label}>{item.label}</span>
    </UnstyledButton>
  );
}

function NavDropdownButton({ item }: { item: NavItem }) {
  const hasActiveChild = item.children?.some((child) => child.active);
  const isActive = item.active || hasActiveChild;

  const trigger = (
    <UnstyledButton className={classes.navButton} data-active={isActive || undefined}>
      {item.icon}
      <span className={classes.label}>{item.label}</span>
      <IconChevronDown size={14} className={classes.chevron} />
    </UnstyledButton>
  );

  const menuItems =
    item.children?.map((child, idx) => ({
      key: `${child.label}-${idx}`,
      label: child.label,
      icon: child.icon,
      onClick: child.onClick,
    })) || [];

  return <DropdownMenu trigger={trigger} items={menuItems} position="bottom" />;
}

export function DesktopNavigation({ items }: { items: NavItem[] }) {
  return (
    <Group gap="xs" wrap="nowrap">
      {items.map((item, index) =>
        item.children ? (
          <NavDropdownButton key={index} item={item} />
        ) : (
          <NavButton key={index} item={item} />
        ),
      )}
    </Group>
  );
}

import { Group, UnstyledButton, Text, useMantineTheme } from "@mantine/core";
import { IconChevronDown } from "@tabler/icons-react";
import { DropdownMenu } from "../../DropdownMenu";
import type { NavItem } from "./types";

function NavButton({ item }: { item: NavItem }) {
  const theme = useMantineTheme();

  return (
    <UnstyledButton
      onClick={item.onClick}
      py="xs"
      px="md"
      style={{
        borderRadius: "var(--mantine-radius-md)",
        color: "white",
        display: "flex",
        alignItems: "center",
        gap: "8px",
        transition: "background-color 150ms ease, box-shadow 150ms ease",
      }}
      styles={{
        root: {
          backgroundColor: item.active
            ? theme.other.layout.bgActive
            : "transparent",
          boxShadow: item.active
            ? "0 0 0 1px rgba(255, 255, 255, 0.3)"
            : "none",
          "&:hover": {
            backgroundColor: item.active
              ? theme.other.layout.bgActive
              : "rgba(255, 255, 255, 0.2)",
            boxShadow: item.active
              ? "0 0 0 1px rgba(255, 255, 255, 0.3)"
              : "none",
          },
        },
      }}
    >
      {item.icon}
      <Text size="sm" fw={500}>
        {item.label}
      </Text>
    </UnstyledButton>
  );
}

function NavDropdownButton({ item }: { item: NavItem }) {
  const theme = useMantineTheme();
  const hasActiveChild = item.children?.some((child) => child.active);
  const isActive = item.active || hasActiveChild;

  const trigger = (
    <UnstyledButton
      py="xs"
      px="md"
      style={{
        borderRadius: "var(--mantine-radius-md)",
        color: "white",
        display: "flex",
        alignItems: "center",
        gap: "8px",
        transition: "background-color 150ms ease, box-shadow 150ms ease",
      }}
      styles={{
        root: {
          backgroundColor: isActive
            ? theme.other.layout.bgActive
            : "transparent",
          boxShadow: isActive ? "0 0 0 1px rgba(255, 255, 255, 0.3)" : "none",
          "&:hover": {
            backgroundColor: isActive
              ? theme.other.layout.bgActive
              : "rgba(255, 255, 255, 0.2)",
          },
        },
      }}
    >
      {item.icon}
      <Text size="sm" fw={500}>
        {item.label}
      </Text>
      <IconChevronDown size={14} style={{ opacity: 0.7 }} />
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

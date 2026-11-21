import { ActionIcon, Chip, Group, Menu, Tooltip } from '@mantine/core';
import { useMediaQuery } from '@mantine/hooks';
import { IconCheck, IconFilter } from '@tabler/icons-react';

export interface FilterToggleOption<T extends string = string> {
  value: T;
  label: string;
}

export interface FilterTogglesProps<T extends string = string> {
  /** The toggles that are on. */
  active: readonly T[];
  onToggle: (value: T) => void;
  options: FilterToggleOption<T>[];
  /** The phone's menu button: its accessible name and tooltip. It is filled while a toggle is on. */
  menuLabel: string;
}

/**
 * Filters that are each on or off and combine, e.g. "My games" and "Public":
 * chips on a wide screen, an icon button with a menu of checkmarks on a phone.
 */
export function FilterToggles<T extends string = string>({
  active,
  onToggle,
  options,
  menuLabel,
}: FilterTogglesProps<T>) {
  const isMobile = useMediaQuery('(max-width: 48em)');

  if (isMobile) {
    return (
      <Menu position="bottom-start" withinPortal closeOnItemClick={false}>
        <Menu.Target>
          <Tooltip label={menuLabel} withArrow>
            <ActionIcon
              // filled while a filter is on: the theme's autoContrast keeps the icon readable
              variant={active.length > 0 ? 'filled' : 'light'}
              color={active.length > 0 ? 'accent' : 'gray'}
              size="md"
              aria-label={menuLabel}
              style={{ flexShrink: 0 }}
            >
              <IconFilter size={18} />
            </ActionIcon>
          </Tooltip>
        </Menu.Target>
        <Menu.Dropdown>
          {options.map((option) => {
            const on = active.includes(option.value);
            return (
              <Menu.Item
                key={option.value}
                onClick={() => onToggle(option.value)}
                fw={on ? 600 : 400}
                role="menuitemcheckbox"
                aria-checked={on}
                rightSection={on ? <IconCheck size={14} color="var(--mantine-color-green-5)" /> : null}
              >
                {option.label}
              </Menu.Item>
            );
          })}
        </Menu.Dropdown>
      </Menu>
    );
  }

  return (
    <Group gap="xs" wrap="nowrap">
      {options.map((option) => (
        <Chip
          key={option.value}
          size="sm"
          // filled: the theme's autoContrast picks readable text on the accent
          variant="filled"
          color="accent"
          checked={active.includes(option.value)}
          onChange={() => onToggle(option.value)}
        >
          {option.label}
        </Chip>
      ))}
    </Group>
  );
}

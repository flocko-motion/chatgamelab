import { ActionIcon, Menu, MultiSelect, Tooltip } from '@mantine/core';
import { useMediaQuery } from '@mantine/hooks';
import { IconCategory, IconCheck } from '@tabler/icons-react';

export interface FilterMultiSelectOption<T extends string = string> {
  value: T;
  label: string;
}

export interface FilterMultiSelectProps<T extends string = string> {
  /** The options picked; none picked narrows nothing. */
  picked: readonly T[];
  onToggle: (value: T) => void;
  onClear: () => void;
  options: FilterMultiSelectOption<T>[];
  /** What is picked from, e.g. "Genre": the field's name and the phone's menu button. */
  label: string;
  /** The phone's menu entry that picks none again. */
  clearLabel: string;
}

/**
 * A filter of several options picked together, e.g. genres: a list keeps
 * what matches any of them. A multi-select on a wide screen, an icon button
 * with a menu of checkmarks on a phone.
 */
export function FilterMultiSelect<T extends string = string>({
  picked,
  onToggle,
  onClear,
  options,
  label,
  clearLabel,
}: FilterMultiSelectProps<T>) {
  const isMobile = useMediaQuery('(max-width: 48em)');

  if (isMobile) {
    return (
      <Menu position="bottom-start" withinPortal closeOnItemClick={false}>
        <Menu.Target>
          <Tooltip label={label} withArrow>
            <ActionIcon
              // filled while an option is picked: the theme's autoContrast keeps the icon readable
              variant={picked.length > 0 ? 'filled' : 'light'}
              color={picked.length > 0 ? 'accent' : 'gray'}
              size="md"
              aria-label={label}
              style={{ flexShrink: 0 }}
            >
              <IconCategory size={18} />
            </ActionIcon>
          </Tooltip>
        </Menu.Target>
        <Menu.Dropdown>
          <Menu.Label>{label}</Menu.Label>
          {options.map((option) => {
            const on = picked.includes(option.value);
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
          {picked.length > 0 && (
            <>
              <Menu.Divider />
              <Menu.Item onClick={onClear}>{clearLabel}</Menu.Item>
            </>
          )}
        </Menu.Dropdown>
      </Menu>
    );
  }

  return (
    <MultiSelect
      size="sm"
      w={220}
      data={options}
      value={[...picked]}
      onChange={(next) => {
        // One change at a time: the option added or removed.
        const added = next.find((v) => !picked.includes(v as T));
        const removed = picked.find((v) => !next.includes(v));
        if (next.length === 0 && picked.length > 1) onClear();
        else if (added) onToggle(added as T);
        else if (removed) onToggle(removed);
      }}
      aria-label={label}
      placeholder={picked.length === 0 ? label : undefined}
      clearable
      clearButtonProps={{ 'aria-label': clearLabel }}
      comboboxProps={{ position: 'bottom-start' }}
    />
  );
}

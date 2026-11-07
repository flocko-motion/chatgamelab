import { Select, Box, Menu, ActionIcon } from '@mantine/core';
import { useMediaQuery } from '@mantine/hooks';
import { IconCheck, IconArrowsSort } from '@tabler/icons-react';

/**
 * SortSelector - Configurable sort dropdown for lists and tables
 * 
 * USE WHEN:
 * - Providing sorting options for data lists
 * - Need consistent sorting UI across the app
 * 
 * @example
 * const options = [
 *   { value: 'name', label: 'Name' },
 *   { value: 'date', label: 'Date' },
 * ];
 * <SortSelector options={options} value="name" onChange={setValue} />
 */

export interface SortOption {
  value: string;
  label: string;
}

export interface SortSelectorProps {
  options: SortOption[];
  value: string;
  onChange: (value: string) => void;
  label?: string;
  placeholder?: string;
  width?: number;
}

export function SortSelector({ 
  options, 
  value, 
  onChange, 
  label,
  placeholder,
  width = 200,
}: SortSelectorProps) {
  const isMobile = useMediaQuery('(max-width: 48em)');

  if (isMobile) {
    return (
      <Menu position="bottom-end" withinPortal>
        <Menu.Target>
          <ActionIcon variant="light" color="gray" size="md" aria-label={label || 'Sort'} style={{ flexShrink: 0 }}>
            <IconArrowsSort size={18} />
          </ActionIcon>
        </Menu.Target>
        <Menu.Dropdown>
          {options.map((option) => (
            <Menu.Item
              key={option.value}
              onClick={() => onChange(option.value)}
              fw={option.value === value ? 600 : 400}
              rightSection={option.value === value ? <IconCheck size={14} color="var(--mantine-color-green-5)" /> : null}
            >
              {option.label}
            </Menu.Item>
          ))}
        </Menu.Dropdown>
      </Menu>
    );
  }

  return (
    <Select
      value={value}
      onChange={(v) => v && onChange(v)}
      data={options}
      size="sm"
      w={width}
      aria-label={label}
      placeholder={placeholder}
      maxDropdownHeight={400}
      comboboxProps={{ position: 'bottom-end' }}
      renderOption={({ option }) => (
        <Box style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', width: '100%' }}>
          <span style={{ fontWeight: option.value === value ? 600 : 400 }}>{option.label}</span>
          {option.value === value && (
            <IconCheck size={16} color="var(--mantine-color-green-5)" />
          )}
        </Box>
      )}
    />
  );
}

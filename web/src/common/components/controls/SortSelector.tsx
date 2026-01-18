import { Select, Box } from '@mantine/core';
import { IconCheck } from '@tabler/icons-react';

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
}

export function SortSelector({ 
  options, 
  value, 
  onChange, 
  label,
  placeholder,
}: SortSelectorProps) {
  return (
    <Select
      value={value}
      onChange={(v) => v && onChange(v)}
      data={options}
      size="sm"
      w={180}
      aria-label={label}
      placeholder={placeholder}
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

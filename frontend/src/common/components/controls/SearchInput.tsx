import { ActionIcon, TextInput } from '@mantine/core';
import { IconSearch, IconX } from '@tabler/icons-react';

export interface SearchInputProps {
  value: string;
  onChange: (value: string) => void;
  /** Shown in the empty field; also its accessible name. */
  placeholder: string;
  /** The label of the clear button. */
  clearLabel: string;
}

/**
 * A search field that is always open: the magnifier sits inside it, a clear
 * button appears once something is typed. It fills the space it is given.
 */
export function SearchInput({ value, onChange, placeholder, clearLabel }: SearchInputProps) {
  return (
    <TextInput
      value={value}
      onChange={(e) => onChange(e.currentTarget.value)}
      placeholder={placeholder}
      aria-label={placeholder}
      size="sm"
      // 16px keeps iOS from zooming into the field
      styles={{ input: { fontSize: 16 } }}
      style={{ flex: 1, minWidth: 0 }}
      leftSection={<IconSearch size={16} />}
      leftSectionPointerEvents="none"
      rightSection={
        value ? (
          <ActionIcon size="sm" variant="subtle" color="gray" onClick={() => onChange('')} aria-label={clearLabel}>
            <IconX size={14} />
          </ActionIcon>
        ) : null
      }
    />
  );
}

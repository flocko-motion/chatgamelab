import { SegmentedControl, Menu, Button } from '@mantine/core';
import { useMediaQuery } from '@mantine/hooks';
import { IconFilter, IconCheck, IconChevronDown } from '@tabler/icons-react';

export interface FilterOption {
  value: string;
  label: string;
}

export interface FilterSegmentedControlProps<T extends string = string> {
  value: T;
  onChange: (value: T) => void;
  options: FilterOption[];
}

export function FilterSegmentedControl<T extends string = string>({
  value,
  onChange,
  options,
}: FilterSegmentedControlProps<T>) {
  const isMobile = useMediaQuery('(max-width: 48em)');

  if (isMobile) {
    const selectedLabel = options.find((o) => o.value === value)?.label ?? value;
    return (
      <Menu position="bottom-start" withinPortal>
        <Menu.Target>
          <Button
            variant="light"
            color="gray"
            size="compact-sm"
            leftSection={<IconFilter size={14} />}
            rightSection={<IconChevronDown size={12} />}
            style={{ flexShrink: 0, whiteSpace: 'nowrap' }}
          >
            {selectedLabel}
          </Button>
        </Menu.Target>
        <Menu.Dropdown>
          {options.map((option) => (
            <Menu.Item
              key={option.value}
              onClick={() => onChange(option.value as T)}
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
    <SegmentedControl
      size="sm"
      value={value}
      onChange={(v) => onChange(v as T)}
      data={options}
      styles={{
        root: {
          backgroundColor: 'var(--mantine-color-gray-2)',
          padding: '0px',
          borderRadius: 'var(--mantine-radius-md)',
          border: '1px solid var(--mantine-color-gray-3)',
        },
        indicator: {
          boxShadow: '0 1px 3px rgba(0, 0, 0, 0.15)',
        },
        label: {
          padding: '6px 12px',
          fontWeight: 500,
          transition: 'all 0.15s ease',
          color: 'var(--mantine-color-gray-7)',
        },
      }}
      classNames={{
        label: 'filter-segmented-label',
      }}
    />
  );
}

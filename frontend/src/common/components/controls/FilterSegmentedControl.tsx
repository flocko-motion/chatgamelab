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
    />
  );
}

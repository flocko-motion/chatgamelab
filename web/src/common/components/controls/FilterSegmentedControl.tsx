import { SegmentedControl } from '@mantine/core';
import { useMediaQuery } from '@mantine/hooks';

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

  return (
    <SegmentedControl
      size={isMobile ? 'xs' : 'sm'}
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

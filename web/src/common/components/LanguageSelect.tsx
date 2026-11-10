import { Box, Select } from '@mantine/core';
import type { ComboboxItem, SelectProps } from '@mantine/core';
import { IconCheck } from '@tabler/icons-react';
import { useLanguageSwitcher } from '@hooks/useTranslation';

interface LanguageSelectProps {
  value: string;
  onChange: (language: string) => void;
  /** Fires on every pick, also of the language already selected. */
  onOptionSubmit?: (language: string) => void;
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  label?: string;
  description?: string;
  disabled?: boolean;
  /** Styling, e.g. the language switcher's soft look on the landing page. */
  classNames?: SelectProps['classNames'];
}

/**
 * The app's languages in the order of the language switcher — English, Deutsch,
 * a separator, then the others — for picking a language that is not the
 * viewer's own, such as a workshop's.
 */
export function LanguageSelect({
  value,
  onChange,
  onOptionSubmit,
  size = 'sm',
  label,
  description,
  disabled,
  classNames,
}: LanguageSelectProps) {
  const { availableLanguages } = useLanguageSwitcher();

  const selectData: ComboboxItem[] = availableLanguages
    .filter((lang) => lang.isStatic || lang.code === '__separator__')
    .map((lang) => ({
      value: lang.code,
      label: lang.name,
      disabled: lang.code === '__separator__',
    }));

  return (
    <Select
      data={selectData}
      value={value}
      onChange={(next) => {
        if (!next || next === '__separator__') return;
        onChange(next);
      }}
      onOptionSubmit={onOptionSubmit}
      size={size}
      label={label}
      description={description}
      disabled={disabled}
      maxDropdownHeight={400}
      classNames={classNames}
      renderOption={({ option }) => {
        if (option.value === '__separator__') {
          return (
            <Box style={{
              textAlign: 'center',
              color: 'var(--mantine-color-gray-4)',
              cursor: 'default',
              userSelect: 'none',
            }}>
              {option.label}
            </Box>
          );
        }

        return (
          <Box style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', width: '100%' }}>
            <span style={{ fontWeight: option.value === value ? 600 : 400 }}>{option.label}</span>
            {option.value === value && (
              <IconCheck size={16} color="var(--mantine-color-green-5)" />
            )}
          </Box>
        );
      }}
    />
  );
}

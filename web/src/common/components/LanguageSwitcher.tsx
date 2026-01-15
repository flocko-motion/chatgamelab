import { Text, Box, Select } from '@mantine/core';
import { IconCheck } from '@tabler/icons-react';
import { useLanguageSwitcher } from '@hooks/useTranslation';
import { useTranslation } from 'react-i18next';
import { Dropdown } from './Dropdown';
import classes from './LanguageSwitcher.module.css';
import subtleClasses from './LanguageSwitcher-subtle.module.css';

interface LanguageSwitcherProps {
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  variant?: 'default' | 'filled' | 'outline' | 'subtle' | 'light' | 'compact';
}

export function LanguageSwitcher({ size = 'sm', variant = 'default' }: LanguageSwitcherProps) {
  const { t } = useTranslation('common');
  const { currentLanguage, availableLanguages, changeLanguage } = useLanguageSwitcher();
  const hasWipLanguages = availableLanguages.some((lang) => !lang.isStatic);

  // Compact mode for dashboard header
  if (variant === 'compact') {
    const selectData = availableLanguages
      .filter(lang => lang.isStatic || lang.code === '__separator__')
      .map((lang) => ({
        value: lang.code,
        label: lang.name,
        disabled: lang.code === '__separator__',
      }));
    
    return (
      <Select
        data={selectData}
        value={currentLanguage.code}
        onChange={(value) => {
          if (!value || value === '__separator__') return;
          void changeLanguage(value);
        }}
        size="sm"
        maxDropdownHeight={400}
        classNames={{
          input: classes.input,
          dropdown: classes.dropdown,
          option: classes.option,
        }}
        renderOption={({ option }) => {
          if (option.value === '__separator__') {
            return (
              <Box style={{ 
                textAlign: 'center', 
                color: 'rgba(255, 255, 255, 0.3)',
                cursor: 'default',
                userSelect: 'none',
              }}>
                {option.label}
              </Box>
            );
          }
          
          return (
            <Box style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', width: '100%' }}>
              <span style={{ fontWeight: option.value === currentLanguage.code ? 600 : 400 }}>{option.label}</span>
              {option.value === currentLanguage.code && (
                <IconCheck size={16} color="var(--mantine-color-green-5)" />
              )}
            </Box>
          );
        }}
      />
    );
  }

  // Subtle variant for light backgrounds (soft gradient style)
  if (variant === 'subtle') {
    const selectData = availableLanguages
      .filter(lang => lang.isStatic || lang.code === '__separator__')
      .map((lang) => ({
        value: lang.code,
        label: lang.name,
        disabled: lang.code === '__separator__',
      }));
    
    return (
      <Select
        data={selectData}
        value={currentLanguage.code}
        onChange={(value) => {
          if (!value || value === '__separator__') return;
          void changeLanguage(value);
        }}
        size={size}
        maxDropdownHeight={400}
        classNames={{
          input: subtleClasses.input,
          dropdown: subtleClasses.dropdown,
          option: subtleClasses.option,
        }}
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
              <span style={{ fontWeight: option.value === currentLanguage.code ? 600 : 400 }}>{option.label}</span>
              {option.value === currentLanguage.code && (
                <IconCheck size={16} color="var(--mantine-color-green-5)" />
              )}
            </Box>
          );
        }}
      />
    );
  }

  return (
    <Box>
      <Dropdown
        size={size}
        variant={variant}
        value={currentLanguage.code}
        onChange={(value) => {
          if (!value) return;
          const selected = availableLanguages.find((l) => l.code === value);
          if (!selected) return;
          if (!selected.isStatic) return;
          void changeLanguage(value);
        }}
        data={availableLanguages.map((lang: { code: string; name: string; isStatic: boolean }) => ({
          value: lang.code,
          disabled: !lang.isStatic,
          label: lang.isStatic ? lang.name : `${lang.name} (${t('languageSwitcher.wipLabel')})`,
        }))}
        styles={{
          input: {
            minWidth: 80,
            maxWidth: 150,
            textAlign: 'center',
          },
        }}
      />

      {hasWipLanguages && (
        <Text size="xs" c="dimmed" mt={4} style={{ whiteSpace: 'nowrap' }}>
          {t('languageSwitcher.wipHint')}
        </Text>
      )}
    </Box>
  );
}

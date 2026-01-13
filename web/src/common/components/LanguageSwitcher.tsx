import { Text, Box, Select } from '@mantine/core';
import { useLanguageSwitcher } from '@hooks/useTranslation';
import { useTranslation } from 'react-i18next';
import { Dropdown } from './Dropdown';

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
    const staticLanguages = availableLanguages.filter(lang => lang.isStatic);
    
    return (
      <Select
        data={staticLanguages.map((lang) => ({
          value: lang.code,
          label: lang.code.toUpperCase(),
        }))}
        value={currentLanguage.code}
        onChange={(value) => {
          if (!value) return;
          void changeLanguage(value);
        }}
        size="sm"
        styles={{
          input: {
            backgroundColor: 'rgba(255, 255, 255, 0.1)',
            borderColor: 'rgba(255, 255, 255, 0.2)',
            color: 'white',
            width: '60px',
            textAlign: 'center',
            fontWeight: 600,
          },
          dropdown: {
            backgroundColor: '#1a1a2e',
            borderColor: 'rgba(255, 255, 255, 0.1)',
          },
          option: {
            color: 'white',
            '&:hover': {
              backgroundColor: 'rgba(255, 255, 255, 0.1)',
            },
          },
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
        leftSection={<Text size="xs">🌍</Text>}
        styles={{
          input: {
            minWidth: 80,
            maxWidth: 150,
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

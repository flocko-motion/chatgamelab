import { Text, Box, Select, useMantineTheme } from '@mantine/core';
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
  const theme = useMantineTheme();

  // Compact mode for dashboard header
  if (variant === 'compact') {
    const staticLanguages = availableLanguages.filter(lang => lang.isStatic);
    
    return (
      <Select
        data={staticLanguages.map((lang) => ({
          value: lang.code,
          label: `${lang.flag}   ${lang.name}`,
        }))}
        value={currentLanguage.code}
        onChange={(value) => {
          if (!value) return;
          void changeLanguage(value);
        }}
        size="sm"
        styles={{
          input: {
            backgroundColor: theme.other.layout.bgSubtle,
            borderColor: theme.other.layout.borderSubtle,
            color: 'white',
            width: '140px',
            paddingLeft: '12px',
            paddingRight: '14px',
            fontWeight: 600,
            '&:hover': {
              backgroundColor: theme.other.layout.bgHover,
            },
            '&:focus': {
              backgroundColor: theme.other.layout.bgActive,
              borderColor: theme.other.layout.borderStrong,
            },
            '&:focus-within': {
              backgroundColor: theme.other.layout.bgActive,
              borderColor: theme.other.layout.borderStrong,
            },
          },
          dropdown: {
            backgroundColor: theme.other.layout.panelBg,
            borderColor: theme.other.layout.lineLight,
          },
          option: {
            color: 'white',
            '&:hover': {
              backgroundColor: theme.other.layout.bgHover,
            },
          },
        }}
      />
    );
  }

  // Subtle variant for light backgrounds (soft gradient style)
  if (variant === 'subtle') {
    return (
      <Box>
        <Dropdown
          size={size}
          variant="filled"
          value={currentLanguage.code}
          onChange={(value) => {
            if (!value) return;
            const selected = availableLanguages.find((l) => l.code === value);
            if (!selected) return;
            if (!selected.isStatic) return;
            void changeLanguage(value);
          }}
          data={availableLanguages.map((lang: { code: string; name: string; flag: string; isStatic: boolean }) => ({
            value: lang.code,
            disabled: !lang.isStatic,
            label: lang.isStatic ? `${lang.flag}   ${lang.name}` : `${lang.flag}   ${lang.name} (${t('languageSwitcher.wipLabel')})`,
          }))}
          styles={{
            input: {
              minWidth: 80,
              maxWidth: 150,
              textAlign: 'center',
              background: 'transparent',
              borderColor: 'var(--mantine-color-accent-5)',
              borderWidth: '2px',
              borderStyle: 'solid',
              color: 'var(--mantine-color-accent-9)',
              fontWeight: 600,
              transition: 'all 0.2s ease',
              '&:hover': {
                background: 'var(--mantine-color-accent-0)',
                borderColor: 'var(--mantine-color-accent-6)',
                transform: 'translateY(-1px)',
              },
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
        data={availableLanguages.map((lang: { code: string; name: string; flag: string; isStatic: boolean }) => ({
          value: lang.code,
          disabled: !lang.isStatic,
          label: lang.isStatic ? `${lang.flag}   ${lang.name}` : `${lang.flag}   ${lang.name} (${t('languageSwitcher.wipLabel')})`,
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

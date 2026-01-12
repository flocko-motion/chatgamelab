import { Text, Box } from '@mantine/core';
import { useLanguageSwitcher } from '@hooks/useTranslation';
import { useTranslation } from 'react-i18next';
import { Dropdown } from './Dropdown';

interface LanguageSwitcherProps {
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  variant?: 'default' | 'filled' | 'outline' | 'subtle' | 'light';
}

export function LanguageSwitcher({ size = 'sm', variant = 'default' }: LanguageSwitcherProps) {
  const { t } = useTranslation('common');
  const { currentLanguage, availableLanguages, changeLanguage } = useLanguageSwitcher();
  const hasWipLanguages = availableLanguages.some((lang) => !lang.isStatic);

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

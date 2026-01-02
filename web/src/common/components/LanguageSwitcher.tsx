import { Select, Loader, Group, Text } from '@mantine/core';
import { useLanguageSwitcher } from '../hooks/useTranslation';

interface LanguageSwitcherProps {
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  variant?: 'default' | 'filled' | 'outline' | 'subtle' | 'light';
}

export function LanguageSwitcher({ size = 'sm', variant = 'default' }: LanguageSwitcherProps) {
  const { currentLanguage, availableLanguages, changeLanguage } = useLanguageSwitcher();

  return (
    <Group gap="sm">
      <Select
        size={size}
        variant={variant}
        value={currentLanguage.code}
        onChange={(value) => value && changeLanguage(value)}
        data={availableLanguages.map((lang: { code: string; name: string; isStatic: boolean }) => ({
          value: lang.code,
          label: `${lang.name}${lang.isStatic ? ' ⚡' : ' 🌐'}`,
        }))}
        leftSection={<Text size="xs">🌍</Text>}
        styles={{
          input: {
            minWidth: 120,
          },
        }}
      />
      {!currentLanguage.isStatic && (
        <Group gap="xs">
          <Loader size="xs" />
          <Text size="xs" c="dimmed">
            Translating...
          </Text>
        </Group>
      )}
    </Group>
  );
}

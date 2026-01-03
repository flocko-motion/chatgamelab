import { createFileRoute, useRouter } from '@tanstack/react-router';
import { Title, Text, Stack, Center, Image } from '@mantine/core';
import { useTranslation } from 'react-i18next';
import logo from '@/assets/logo.png';
import { Button } from '@components/Button';

export const Route = createFileRoute('/')({
  component: HomePage,
});

function HomePage() {
  const { t } = useTranslation('common');
  const router = useRouter();

  return (
    <Stack gap="xl" py="xl">
      <Center>
        <Stack gap="lg" align="center" ta="center">
          <Image 
            src={logo} 
            alt="ChatGameLab Logo" 
            w={400}
            h={400}
            fit="contain"
          />
          <Text size="lg" c="dimmed" maw={600}>
            {t('home.splashDescription')}
          </Text>

          <Button 
            size="lg"
            onClick={() => {
              router.navigate({ to: '/auth/login' });
            }}
          >
            {t('home.loginCta')}
          </Button>
        </Stack>
      </Center>
    </Stack>
  );
}

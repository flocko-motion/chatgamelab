import { AppShell, Container, Anchor, Box, Group } from '@mantine/core';
import { useTranslation } from 'react-i18next';
import { HelperText } from '../typography';
import { VersionDisplay } from '../VersionDisplay';
import { useResponsiveDesign } from '../../hooks/useResponsiveDesign';

export interface FooterLink {
  label: string;
  href: string;
}

export interface AppFooterProps {
  links?: FooterLink[];
  showVersion?: boolean;
  transparent?: boolean;
}

const defaultLinks: FooterLink[] = [
  { label: 'Auth0', href: 'https://auth0.com' },
  { label: 'omnitopos.net', href: 'https://omnitopos.net' },
  { label: 'tausend-medien.de', href: 'https://tausend-medien.de' },
];

export function AppFooter({ links = defaultLinks, showVersion = true, transparent = false }: AppFooterProps) {
  const { mobileBreakpoint } = useResponsiveDesign();
  const { t } = useTranslation('common');

  return (
    <AppShell.Footer
      style={{
        backgroundColor: transparent ? 'transparent' : undefined,
        borderTop: transparent ? '1px solid var(--mantine-color-accent-2)' : undefined,
      }}
    >
      <Container size="xl" h="100%" px={{ base: 'sm', sm: 'md', lg: 'xl' }}>
        <Box
          h="100%"
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          {/* Desktop: Full footer with all links */}
          <Box visibleFrom={mobileBreakpoint}>
            <Group gap="xs" justify="center" wrap="wrap">
              <HelperText>
                {t('footer.loginVia')}{' '}
                <Anchor href={links[0]?.href} target="_blank" size="sm" c="accent">
                  {links[0]?.label}
                </Anchor>
              </HelperText>
              <HelperText>|</HelperText>
              <HelperText>
                {t('footer.programmedBy')}{' '}
                <Anchor href={links[1]?.href} target="_blank" size="sm" c="accent">
                  {links[1]?.label}
                </Anchor>
              </HelperText>
              <HelperText>|</HelperText>
              <HelperText>
                {t('footer.producedBy')}{' '}
                <Anchor href={links[2]?.href} target="_blank" size="sm" c="accent">
                  {links[2]?.label}
                </Anchor>
              </HelperText>
              {showVersion && (
                <>
                  <HelperText>|</HelperText>
                  <VersionDisplay />
                </>
              )}
            </Group>
          </Box>

          {/* Mobile: Condensed footer */}
          <Box hiddenFrom={mobileBreakpoint}>
            <Group gap="xs" justify="center">
              <HelperText>
                {t('footer.copyright')}
              </HelperText>
              {showVersion && (
                <>
                  <HelperText>•</HelperText>
                  <VersionDisplay />
                </>
              )}
            </Group>
          </Box>
        </Box>
      </Container>
    </AppShell.Footer>
  );
}

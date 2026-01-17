import { AppShell, Container, Anchor, Box, Group } from '@mantine/core';
import { useTranslation } from 'react-i18next';
import { HelperText } from '../typography';
import { VersionDisplay } from '../VersionDisplay';
import { useResponsiveDesign } from '../../hooks/useResponsiveDesign';
import { EXTERNAL_LINKS } from '../../../config/externalLinks';

export interface FooterLink {
  label: string;
  href: string;
}

export interface AppFooterProps {
  links?: FooterLink[];
  showVersion?: boolean;
  transparent?: boolean;
  darkMode?: boolean;
}

const defaultLinks: FooterLink[] = [
  { label: 'omnitopos.net', href: 'https://omnitopos.net' },
  { label: 'JFF - Institut für Medienpädagogik', href: EXTERNAL_LINKS.JFF.href },
];

export function AppFooter({ links = defaultLinks, showVersion = true, darkMode = false }: AppFooterProps) {
  const { mobileBreakpoint } = useResponsiveDesign();
  const { t } = useTranslation('common');

  const dimmedStyles = {
    backgroundColor: '#dddde3',
    borderTop: '1px solid rgba(0, 0, 0, 0.08)',
  };

  const lightStyles = {
    backgroundColor: 'var(--mantine-color-gray-0)',
    borderTop: '1px solid var(--mantine-color-gray-2)',
  };

  const textColor = darkMode ? 'gray.6' : 'gray.5';
  const linkColor = darkMode ? 'gray.7' : 'accent.8';

  return (
    <AppShell.Footer
      style={darkMode ? dimmedStyles : lightStyles}
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
              <HelperText c={textColor}>
                {t('footer.programmedBy')}{' '}
                <Anchor href={links[0]?.href} target="_blank" size="sm" c={linkColor}>
                  {links[0]?.label}
                </Anchor>
              </HelperText>
              <HelperText c={textColor}>|</HelperText>
              <HelperText c={textColor}>
                <Anchor href={links[1]?.href} target="_blank" size="sm" c={linkColor} title={t('footer.jffTitle')}>
                  {links[1]?.label}
                </Anchor>
              </HelperText>
              {showVersion && (
                <>
                  <HelperText c={textColor}>|</HelperText>
                  <VersionDisplay darkMode={darkMode} />
                </>
              )}
            </Group>
          </Box>

          {/* Mobile: Condensed footer */}
          <Box hiddenFrom={mobileBreakpoint}>
            <Group gap="xs" justify="center">
              <HelperText c={textColor}>
                {t('footer.copyright')}
              </HelperText>
              {showVersion && (
                <>
                  <HelperText c={textColor}>•</HelperText>
                  <VersionDisplay darkMode={darkMode} />
                </>
              )}
            </Group>
          </Box>
        </Box>
      </Container>
    </AppShell.Footer>
  );
}

import { AppShell, Container, Anchor, Box, Group } from '@mantine/core';
import { useTranslation } from 'react-i18next';
import { HelperText } from '../typography';
import { VersionDisplay } from '../VersionDisplay';
import { useResponsiveDesign } from '../../hooks/useResponsiveDesign';
import { useBranding } from '@/providers/branding';

export interface AppFooterProps {
  showVersion?: boolean;
  transparent?: boolean;
  darkMode?: boolean;
}

export function AppFooter({ showVersion = true, darkMode = false }: AppFooterProps) {
  const { mobileBreakpoint } = useResponsiveDesign();
  const { t } = useTranslation('common');
  const branding = useBranding();

  const dimmedStyles = {
    backgroundColor: 'var(--mantine-color-gray-1)',
    borderTop: '1px solid var(--mantine-color-gray-2)',
  };

  const lightStyles = {
    backgroundColor: 'var(--mantine-color-gray-0)',
    borderTop: '1px solid var(--mantine-color-gray-2)',
  };

  const textColor = darkMode ? 'gray.6' : 'gray.5';
  const linkColor = darkMode ? 'gray.7' : 'ink';

  return (
    <AppShell.Footer
      style={darkMode ? dimmedStyles : lightStyles}
      visibleFrom={mobileBreakpoint}
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
              {[
                branding.creditName && (
                  <HelperText key="credit" c={textColor}>
                    {t('footer.programmedBy')}{' '}
                    {branding.creditUrl ? (
                      <Anchor href={branding.creditUrl} target="_blank" size="sm" c={linkColor}>
                        {branding.creditName}
                      </Anchor>
                    ) : (
                      branding.creditName
                    )}
                  </HelperText>
                ),
                branding.operatorName && (
                  <HelperText key="operator" c={textColor}>
                    {branding.operatorUrl ? (
                      <Anchor href={branding.operatorUrl} target="_blank" size="sm" c={linkColor}>
                        {branding.operatorName}
                      </Anchor>
                    ) : (
                      branding.operatorName
                    )}
                  </HelperText>
                ),
                branding.contactUrl && (
                  <HelperText key="contact" c={textColor}>
                    <Anchor href={branding.contactUrl} target="_blank" size="sm" c={linkColor}>
                      {t('header.contact')}
                    </Anchor>
                  </HelperText>
                ),
                showVersion && <VersionDisplay key="version" darkMode={darkMode} />,
              ]
                .filter(Boolean)
                .flatMap((item, i) => (i === 0 ? [item] : [<HelperText key={`sep${i}`} c={textColor}>|</HelperText>, item]))}
            </Group>
          </Box>

          {/* Mobile: Footer is hidden - version and language switcher are in burger menu */}
        </Box>
      </Container>
    </AppShell.Footer>
  );
}

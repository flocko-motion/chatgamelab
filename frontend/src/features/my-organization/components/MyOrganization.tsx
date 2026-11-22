import {
  Card,
  Container,
  Textarea,
  Stack,
  Tabs,
  Alert,
  Divider,
} from '@mantine/core';
import { IconAlertCircle, IconUsers, IconSchool, IconKey, IconShieldLock } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { BodyText, HelperText, Label, PageTitle } from '@components/typography';
import { useOrganisation, useSystemConstraints, useUpdateOrganisation } from '@/api/hooks';
import { useAuth } from '@/providers/AuthProvider';
import { getUserInstitutionId, isAtLeastStaff } from '@/common/lib/roles';
import { useOrganisationTabs, type OrganisationTab } from '../hooks/useOrganisationTabs';
import { MembersTab } from './MembersTab';
import { WorkshopsTab } from './WorkshopsTab';
import { ApiKeysTab } from './ApiKeysTab';
import { OrganisationDeclarationCard } from './OrganisationDeclarationCard';

interface MyOrganizationProps {
  /** Auto-open the create workshop modal on mount */
  autoCreateWorkshop?: boolean;
  /** The tab in the URL, if any. */
  tab?: OrganisationTab;
}

export function MyOrganization({ autoCreateWorkshop, tab: urlTab }: MyOrganizationProps) {
  const { t } = useTranslation('common');
  const { backendUser } = useAuth();

  const institutionId = getUserInstitutionId(backendUser);
  const canManage = isAtLeastStaff(backendUser);
  const { data: institution } = useOrganisation(institutionId);
  // Whether the caller may change the organisation's settings: the server's flag.
  const canEditSettings = !!institution?.mayEditSettings;
  const { tab, setTab } = useOrganisationTabs({
    tab: urlTab,
    autoCreateWorkshop,
    canManage,
    canEditSettings,
  });

  const updateOrganisation = useUpdateOrganisation();
  const orgConstraintEmpty = !institution?.promptConstraints?.trim();
  // The site's fallback, shown while the organisation sets no constraint.
  const { data: siteConstraints } = useSystemConstraints(canEditSettings && orgConstraintEmpty);

  // Early returns for error states
  if (!institutionId) {
    return (
      <Container size="xl" py="xl">
        <Alert icon={<IconAlertCircle size={16} />} title={t('myOrganization.noOrganization')} color="yellow">
          {t('myOrganization.noOrganizationDescription')}
        </Alert>
      </Container>
    );
  }

  return (
    <Container size="xl" py="xl">
      <Stack gap="lg">
        {/* Header */}
        <Stack gap={0}>
          <PageTitle>{t('myOrganization.title')}</PageTitle>
          {institution?.name && (
            <HelperText>{institution.name}</HelperText>
          )}
        </Stack>

        <OrganisationDeclarationCard />

        {/* One panel at a time: each tab's list keeps its query in the URL. */}
        <Tabs value={tab} onChange={setTab} keepMounted={false}>
          <Tabs.List>
            <Tabs.Tab value="members" leftSection={<IconUsers size={16} />}>
              {t('navigation:orgMembers')}
            </Tabs.Tab>
            {canManage && (
              <Tabs.Tab value="workshops" leftSection={<IconSchool size={16} />}>
                {t('navigation:orgWorkshops')}
              </Tabs.Tab>
            )}
            {canManage && (
              <Tabs.Tab value="apikeys" leftSection={<IconKey size={16} />}>
                {t('navigation:orgApiKeys')}
              </Tabs.Tab>
            )}
            {canEditSettings && (
              <Tabs.Tab value="constraints" leftSection={<IconShieldLock size={16} />}>
                {t('navigation:orgConstraints')}
              </Tabs.Tab>
            )}
          </Tabs.List>

          <Tabs.Panel value="members" pt="md">
            <MembersTab institution={institution} />
          </Tabs.Panel>

          {canManage && (
            <Tabs.Panel value="workshops" pt="md">
              <WorkshopsTab
                institutionId={institutionId}
                institutionName={institution?.name}
                autoCreate={autoCreateWorkshop}
              />
            </Tabs.Panel>
          )}

          {canManage && (
            <Tabs.Panel value="apikeys" pt="md">
              <ApiKeysTab
                institutionId={institutionId}
                institutionName={institution?.name}
                freeUseApiKeyShareId={institution?.freeUseApiKeyShareId}
              />
            </Tabs.Panel>
          )}

          {canEditSettings && (
            <Tabs.Panel value="constraints" pt="md">
              <Card shadow="sm" padding="lg" radius="md" withBorder>
                <Stack gap="md">
                  <Textarea
                    label={t('myOrganization.promptConstraintsLabel')}
                    placeholder={t('myOrganization.promptConstraintsPlaceholder')}
                    description={t('myOrganization.promptConstraintsHint')}
                    size="sm"
                    minRows={3}
                    maxRows={4}
                    autosize
                    maxLength={200}
                    defaultValue={institution?.promptConstraints || ''}
                    key={`prompt-constraints-${institutionId}-${institution?.promptConstraints || ''}`}
                    disabled={updateOrganisation.isPending}
                    onBlur={(event) => {
                      const nextValue = event.currentTarget.value;
                      if ((institution?.promptConstraints || '') === nextValue) return;
                      updateOrganisation.mutate({ id: institutionId, promptConstraints: nextValue });
                    }}
                  />
                  {orgConstraintEmpty && siteConstraints && (
                    <>
                      <Divider />
                      <Stack gap="xs">
                        <Label>{t('myOrganization.promptConstraintsFallbackTitle')}</Label>
                        <HelperText>
                          {t('myOrganization.promptConstraintsFallbackHint')}
                        </HelperText>
                        {(['U13', 'U13p', 'U18'] as const).map((bucket) => {
                          const value =
                            bucket === 'U13'
                              ? siteConstraints.promptConstraintU13
                              : bucket === 'U13p'
                                ? siteConstraints.promptConstraintU13p
                                : siteConstraints.promptConstraintU18;
                          return (
                            <Stack key={bucket} gap={2}>
                              <BodyText size="sm" fw={500}>
                                {t(`myOrganization.promptConstraintsFallback${bucket}`)}
                              </BodyText>
                              {value ? (
                                <BodyText size="sm">{value}</BodyText>
                              ) : (
                                <HelperText>{t('myOrganization.promptConstraintsFallbackEmpty')}</HelperText>
                              )}
                            </Stack>
                          );
                        })}
                      </Stack>
                    </>
                  )}
                </Stack>
              </Card>
            </Tabs.Panel>
          )}
        </Tabs>
      </Stack>
    </Container>
  );
}

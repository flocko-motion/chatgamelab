import {
  Card,
  Container,
  Title,
  Text,
  Textarea,
  Stack,
  Tabs,
  Alert,
  Divider,
} from '@mantine/core';
import { IconAlertCircle, IconUsers, IconSchool, IconKey, IconShieldLock } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '@/api/queryKeys';
import { useRequiredAuthenticatedApi } from '@/api/useAuthenticatedApi';
import { useAuth } from '@/providers/AuthProvider';
import { getUserInstitutionId, isAtLeastHead, isAtLeastStaff } from '@/common/lib/roles';
import { MembersTab } from './MembersTab';
import { WorkshopsTab } from './WorkshopsTab';
import { ApiKeysTab } from './ApiKeysTab';

interface MyOrganizationProps {
  /** Auto-open the create workshop modal on mount */
  autoCreateWorkshop?: boolean;
}

export function MyOrganization({ autoCreateWorkshop }: MyOrganizationProps) {
  const { t } = useTranslation('common');
  const api = useRequiredAuthenticatedApi();
  const { backendUser } = useAuth();
  const queryClient = useQueryClient();

  const institutionId = getUserInstitutionId(backendUser);
  const canManage = isAtLeastStaff(backendUser);
  const canEditSettings = isAtLeastHead(backendUser);

  // Fetch institution members
  const { data: members = [], isLoading, error } = useQuery({
    queryKey: queryKeys.institutionMembers(institutionId!),
    queryFn: async () => {
      if (!institutionId) return [];
      const response = await api.institutions.membersList(institutionId);
      return response.data;
    },
    enabled: !!institutionId,
  });

  // Fetch institution details
  const { data: institution } = useQuery({
    queryKey: queryKeys.institution(institutionId!),
    queryFn: async () => {
      if (!institutionId) return null;
      const response = await api.institutions.institutionsDetail(institutionId);
      return response.data;
    },
    enabled: !!institutionId,
  });

  const updatePromptConstraints = useMutation({
    mutationFn: async (promptConstraints: string) => {
      if (!institutionId) return;
      await api.institutions.promptConstraintsPartialUpdate(institutionId, {
        promptConstraints: promptConstraints || undefined,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.institution(institutionId!) });
    },
  });

  const orgConstraintEmpty = !institution?.promptConstraints?.trim();
  const { data: siteConstraints } = useQuery({
    queryKey: ['systemConstraints'] as const,
    queryFn: async () => {
      const response = await api.system.constraintsList();
      return response.data;
    },
    enabled: canEditSettings && orgConstraintEmpty,
  });

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

  if (error) {
    return (
      <Container size="xl" py="xl">
        <Alert icon={<IconAlertCircle size={16} />} title={t('error')} color="red">
          {error instanceof Error ? error.message : 'Failed to load organization'}
        </Alert>
      </Container>
    );
  }

  return (
    <Container size="xl" py="xl">
      <Stack gap="lg">
        {/* Header */}
        <Stack gap={0}>
          <Title order={2}>{t('myOrganization.title')}</Title>
          {institution?.name && (
            <Text c="dimmed" size="sm">{institution.name}</Text>
          )}
        </Stack>

        <Tabs defaultValue="members">
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
            <MembersTab
              members={members}
              isLoading={isLoading}
              institution={institution}
            />
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
                    disabled={updatePromptConstraints.isPending}
                    onBlur={(event) => {
                      const nextValue = event.currentTarget.value;
                      if ((institution?.promptConstraints || '') === nextValue) return;
                      updatePromptConstraints.mutate(nextValue);
                    }}
                  />
                  {orgConstraintEmpty && siteConstraints && (
                    <>
                      <Divider />
                      <Stack gap="xs">
                        <Text size="sm" fw={600}>
                          {t('myOrganization.promptConstraintsFallbackTitle')}
                        </Text>
                        <Text size="xs" c="dimmed">
                          {t('myOrganization.promptConstraintsFallbackHint')}
                        </Text>
                        {(['U13', 'U13p', 'U18'] as const).map((bucket) => {
                          const value =
                            bucket === 'U13'
                              ? siteConstraints.promptConstraintU13
                              : bucket === 'U13p'
                                ? siteConstraints.promptConstraintU13p
                                : siteConstraints.promptConstraintU18;
                          return (
                            <Stack key={bucket} gap={2}>
                              <Text size="xs" fw={500}>
                                {t(`myOrganization.promptConstraintsFallback${bucket}`)}
                              </Text>
                              <Text size="xs" c={value ? undefined : 'dimmed'} fs={value ? undefined : 'italic'}>
                                {value || t('myOrganization.promptConstraintsFallbackEmpty')}
                              </Text>
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

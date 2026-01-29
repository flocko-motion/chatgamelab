import {
  Container,
  Title,
  Text,
  Stack,
  Alert,
} from '@mantine/core';
import { IconAlertCircle } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useQuery } from '@tanstack/react-query';
import { queryKeys } from '@/api/queryKeys';
import { useRequiredAuthenticatedApi } from '@/api/useAuthenticatedApi';
import { useAuth } from '@/providers/AuthProvider';
import { getUserInstitutionId } from '@/common/lib/roles';
import { MembersTab } from './MembersTab';

export function MyOrganization() {
  const { t } = useTranslation('common');
  const api = useRequiredAuthenticatedApi();
  const { backendUser } = useAuth();

  const institutionId = getUserInstitutionId(backendUser);

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

        {/* Members content */}
        <MembersTab
          members={members}
          isLoading={isLoading}
          institution={institution}
        />
      </Stack>
    </Container>
  );
}

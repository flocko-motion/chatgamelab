import { useState, useMemo, useCallback } from 'react';
import { Table, ActionIcon, Badge, Text, Alert, Stack, Card, Group, TextInput } from '@mantine/core';
import { IconTrash, IconAlertCircle, IconSearch } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useQuery } from '@tanstack/react-query';
import { useInvites, useRevokeInvite } from '@/api/hooks';
import { useRequiredAuthenticatedApi } from '@/api/useAuthenticatedApi';
import { useResponsiveDesign } from '@/common/hooks/useResponsiveDesign';
import { SortSelector, type SortOption } from '@/common/components/controls';
import type { RoutesInviteResponse, ObjInstitution } from '@/api/generated';

const getStatusColor = (status: string) => {
  switch (status) {
    case 'pending':
      return 'blue';
    case 'accepted':
      return 'green';
    case 'declined':
      return 'gray';
    case 'revoked':
      return 'red';
    case 'expired':
      return 'orange';
    default:
      return 'gray';
  }
};

const getDisplayDate = (invite: RoutesInviteResponse) => {
  return invite.modifiedAt || invite.createdAt;
};

interface InvitesListProps {
  institutionId?: string;
  showInstitutionColumn?: boolean;
}

export function InvitesList({ institutionId, showInstitutionColumn = false }: InvitesListProps) {
  const { t } = useTranslation('common');
  const { t: tAuth } = useTranslation('auth');
  const api = useRequiredAuthenticatedApi();
  const { isMobile } = useResponsiveDesign();
  const { data: invites, isLoading, error } = useInvites();
  const [sortValue, setSortValue] = useState('modifiedAt-desc');
  const [searchQuery, setSearchQuery] = useState('');
  const { data: institutions } = useQuery({
    queryKey: ['institutions'],
    queryFn: async () => {
      const response = await api.institutions.institutionsList();
      return response.data;
    },
  });
  const revokeInvite = useRevokeInvite();

  const getInstitutionName = useCallback((instId?: string) => {
    if (!instId || !institutions) return instId || '-';
    const institution = institutions.find((org: ObjInstitution) => org.id === instId);
    return institution?.name || instId;
  }, [institutions]);

  const translateRole = useCallback((role?: string) => {
    if (!role) return '-';
    const roleKey = role.toLowerCase();
    return tAuth(`profile.roles.${roleKey}`, role);
  }, [tAuth]);

  const translateStatus = useCallback((status?: string) => {
    if (!status) return 'pending';
    return t(`admin.invites.statuses.${status}`, status);
  }, [t]);

  const handleRevoke = useCallback((inviteId: string) => {
    if (confirm(t('admin.invites.revokeConfirm'))) {
      revokeInvite.mutate(inviteId);
    }
  }, [t, revokeInvite]);

  // Parse combined sort value into field and direction
  const [sortField, sortDirection] = sortValue.split('-') as [string, 'asc' | 'desc'];

  const sortOptions: SortOption[] = useMemo(() => {
    const options: SortOption[] = [
      { value: 'modifiedAt-desc', label: t('admin.invites.sort.modifiedAt-desc') },
      { value: 'modifiedAt-asc', label: t('admin.invites.sort.modifiedAt-asc') },
      { value: 'status-asc', label: t('admin.invites.sort.status-asc') },
      { value: 'status-desc', label: t('admin.invites.sort.status-desc') },
      { value: 'role-asc', label: t('admin.invites.sort.role-asc') },
      { value: 'role-desc', label: t('admin.invites.sort.role-desc') },
    ];
    if (showInstitutionColumn) {
      options.push(
        { value: 'organization-asc', label: t('admin.invites.sort.organization-asc') },
        { value: 'organization-desc', label: t('admin.invites.sort.organization-desc') }
      );
    }
    return options;
  }, [t, showInstitutionColumn]);

  // Base invites (before search filtering)
  const baseInvites = useMemo(() => {
    if (!invites) return [];
    return institutionId
      ? invites.filter((inv) => inv.institutionId === institutionId)
      : invites;
  }, [invites, institutionId]);

  const filteredInvites = useMemo(() => {
    if (!searchQuery.trim()) return baseInvites;
    
    const query = searchQuery.toLowerCase();
    return baseInvites.filter((inv) => {
      const email = inv.invitedEmail?.toLowerCase() || '';
      const role = translateRole(inv.role).toLowerCase();
      const status = translateStatus(inv.status).toLowerCase();
      const orgName = getInstitutionName(inv.institutionId).toLowerCase();
      return email.includes(query) || role.includes(query) || status.includes(query) || orgName.includes(query);
    });
  }, [baseInvites, searchQuery, translateRole, translateStatus, getInstitutionName]);

  const sortedInvites = useMemo(() => {
    if (filteredInvites.length === 0) return [];
    return [...filteredInvites].sort((a, b) => {
      let aVal: string;
      let bVal: string;

      switch (sortField) {
        case 'role':
          aVal = a.role || '';
          bVal = b.role || '';
          break;
        case 'organization':
          aVal = getInstitutionName(a.institutionId);
          bVal = getInstitutionName(b.institutionId);
          break;
        case 'status':
          aVal = a.status || '';
          bVal = b.status || '';
          break;
        case 'modifiedAt':
          aVal = getDisplayDate(a) || '';
          bVal = getDisplayDate(b) || '';
          break;
        default:
          return 0;
      }

      const comparison = aVal.localeCompare(bVal);
      return sortDirection === 'asc' ? comparison : -comparison;
    });
  }, [filteredInvites, sortField, sortDirection, getInstitutionName]);

  if (error) {
    return (
      <Alert icon={<IconAlertCircle size={16} />} color="red">
        {t('common.error')}: {error.message}
      </Alert>
    );
  }

  if (isLoading) {
    return <Text>{t('common.loading')}</Text>;
  }

  // Show empty state only if there are no invites at all (not due to search)
  if (baseInvites.length === 0 && !searchQuery.trim()) {
    return (
      <Text c="dimmed" size="sm">
        {t('admin.invites.empty')}
      </Text>
    );
  }

  // Render no results message when search yields nothing
  const renderNoResults = () => (
    <Text c="dimmed" size="sm" ta="center" py="md">
      {t('admin.invites.noResults')}
    </Text>
  );

  if (isMobile) {
    return (
      <Stack gap="sm">
        <Group justify="space-between" gap="sm">
          <TextInput
            placeholder={t('admin.invites.searchPlaceholder')}
            leftSection={<IconSearch size={16} />}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.currentTarget.value)}
            style={{ flex: 1 }}
          />
          <SortSelector
            options={sortOptions}
            value={sortValue}
            onChange={setSortValue}
            label={t('admin.invites.sort.label')}
            width={180}
          />
        </Group>
        {sortedInvites.length === 0 ? renderNoResults() : sortedInvites.map((invite: RoutesInviteResponse) => (
          <Card key={invite.id} withBorder padding="sm" radius="md">
            <Stack gap="xs">
              <Group justify="space-between" wrap="nowrap">
                <Text size="sm" fw={600} lineClamp={1}>
                  {invite.invitedEmail || invite.invitedUserId || '-'}
                </Text>
                <Badge size="sm" color={getStatusColor(invite.status || 'pending')}>
                  {translateStatus(invite.status)}
                </Badge>
              </Group>
              
              <Group gap="xs" wrap="wrap">
                <Badge size="xs" variant="light">
                  {translateRole(invite.role)}
                </Badge>
                {showInstitutionColumn && (
                  <Badge size="xs" variant="outline" color="gray">
                    {getInstitutionName(invite.institutionId)}
                  </Badge>
                )}
              </Group>

              <Group justify="space-between" align="center">
                <Text size="xs" c="dimmed">
                  {getDisplayDate(invite) ? new Date(getDisplayDate(invite)!).toLocaleDateString() : '-'}
                </Text>
                {invite.status === 'pending' && (
                  <ActionIcon
                    color="red"
                    variant="light"
                    size="sm"
                    onClick={() => handleRevoke(invite.id!)}
                    loading={revokeInvite.isPending}
                  >
                    <IconTrash size={14} />
                  </ActionIcon>
                )}
              </Group>
            </Stack>
          </Card>
        ))}
      </Stack>
    );
  }

  return (
    <Stack gap="md">
      <Group justify="space-between" gap="md">
        <TextInput
          placeholder={t('admin.invites.searchPlaceholder')}
          leftSection={<IconSearch size={16} />}
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.currentTarget.value)}
          style={{ flex: 1, maxWidth: 400 }}
        />
        <SortSelector
          options={sortOptions}
          value={sortValue}
          onChange={setSortValue}
          label={t('admin.invites.sort.label')}
          width={220}
        />
      </Group>
      {sortedInvites.length === 0 ? renderNoResults() : (
        <Table>
          <Table.Thead>
            <Table.Tr>
              {showInstitutionColumn && (
                <Table.Th>{t('admin.invites.organization')}</Table.Th>
              )}
              <Table.Th>{t('admin.invites.role')}</Table.Th>
              <Table.Th>{t('admin.invites.invitedEmail')}</Table.Th>
              <Table.Th>{t('admin.invites.status')}</Table.Th>
              <Table.Th>{t('admin.invites.modifiedAt')}</Table.Th>
              <Table.Th>{t('admin.invites.actions')}</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {sortedInvites.map((invite: RoutesInviteResponse) => (
              <Table.Tr key={invite.id}>
                {showInstitutionColumn && (
                  <Table.Td>
                    <Text size="sm">{getInstitutionName(invite.institutionId)}</Text>
                  </Table.Td>
                )}
                <Table.Td>
                  <Text size="sm" fw={600}>{translateRole(invite.role)}</Text>
                </Table.Td>
                <Table.Td>
                  <Text size="sm">{invite.invitedEmail || invite.invitedUserId || '-'}</Text>
                </Table.Td>
                <Table.Td>
                  <Badge color={getStatusColor(invite.status || 'pending')}>
                    {translateStatus(invite.status)}
                  </Badge>
                </Table.Td>
                <Table.Td>
                  <Text size="sm">
                    {getDisplayDate(invite) ? new Date(getDisplayDate(invite)!).toLocaleDateString() : '-'}
                  </Text>
                </Table.Td>
                <Table.Td>
                  {invite.status === 'pending' && (
                    <ActionIcon
                      color="red"
                      variant="subtle"
                      onClick={() => handleRevoke(invite.id!)}
                      loading={revokeInvite.isPending}
                    >
                      <IconTrash size={16} />
                    </ActionIcon>
                  )}
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      )}
    </Stack>
  );
}

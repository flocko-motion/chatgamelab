import { api } from "@/api/client/api";
import { useState, useMemo } from 'react';
import {
  Container,
  Title,
  Text,
  Stack,
  Card,
  Group,
  TextInput,
  Button,
  Table,
  Modal,
  LoadingOverlay,
  Alert,
  Badge,
  ActionIcon,
  Tooltip,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { IconPlus, IconAlertCircle, IconSend, IconEdit, IconTrash, IconSearch, IconLock, IconLockOpen } from '@tabler/icons-react';
import { ExpandableSearch } from '@/common/components/controls';
import { PlusIconButton } from '@/common/components/buttons';
import { useTranslation } from 'react-i18next';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '@/api/queryKeys';
import { useResponsiveDesign } from '@/common/hooks/useResponsiveDesign';
import type { OrganisationView } from "@/api/generated/api.gen";
import { InvitesList } from './InvitesList';
import { OrganizationMembersRow } from './OrganizationMembersRow';
import { OrganizationStatusBadges } from './OrganizationStatusBadges';
import { OrganizationSuspendModal } from './OrganizationSuspendModal';
import { RecentRegistrationsList } from './RecentRegistrationsList';
import { headNames } from '../lib/organizationHeads';

/** An organisation's operator and address, as an admin enters them. */
const noDetails = { operator: '', street: '', postalCode: '', city: '' };

export function OrganizationsManagement() {
  const { t } = useTranslation('common');
  const queryClient = useQueryClient();
  const { isMobile } = useResponsiveDesign();
  
  const [editingOrg, setEditingOrg] = useState<OrganisationView | null>(null);
  const [inviteOrg, setInviteOrg] = useState<OrganisationView | null>(null);
  const [newOrgName, setNewOrgName] = useState('');
  const [newOrgDetails, setNewOrgDetails] = useState(noDetails);
  const [suspendOrg, setSuspendOrg] = useState<OrganisationView | null>(null);
  const [editName, setEditName] = useState('');
  const [inviteEmail, setInviteEmail] = useState('');
  
  const [createModalOpened, { open: openCreateModal, close: closeCreateModal }] = useDisclosure(false);
  const [editModalOpened, { open: openEditModal, close: closeEditModal }] = useDisclosure(false);
  const [inviteModalOpened, { open: openInviteModal, close: closeInviteModal }] = useDisclosure(false);
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] = useDisclosure(false);
  const [orgToDelete, setOrgToDelete] = useState<OrganisationView | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [inviteError, setInviteError] = useState<string | null>(null);

  // Helper to extract error message from API error
  const getErrorMessage = (error: unknown): string => {
    if (error && typeof error === 'object' && 'error' in error) {
      const apiError = error as { error?: { error?: string } };
      const message = apiError.error?.error || '';
      if (message.includes('pending invite already exists')) {
        return t('myOrganization.inviteAlreadyExists');
      }
      return message || t('myOrganization.inviteError');
    }
    return t('myOrganization.inviteError');
  };

  // Fetch all institutions
  const { data: organizations, isLoading, error } = useQuery({
    queryKey: queryKeys.institutions,
    queryFn: async () => {
      const response = await api.getApiInstitutions();
      return response.data;
    },
  });

  // Filter organizations by search query
  const filteredOrganizations = useMemo(() => {
    if (!organizations) return [];
    if (!searchQuery.trim()) return organizations;
    
    const query = searchQuery.toLowerCase();
    return organizations.filter((org) =>
      [org.name, org.operator, org.city, headNames(org)].some((v) => v?.toLowerCase().includes(query))
    );
  }, [organizations, searchQuery]);

  // Create institution mutation
  const createMutation = useMutation({
    mutationFn: async (name: string) => {
      const response = await api.postApiInstitutions({
        name,
        operator: newOrgDetails.operator.trim(),
        street: newOrgDetails.street.trim(),
        postalCode: newOrgDetails.postalCode.trim(),
        city: newOrgDetails.city.trim(),
      });
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.institutions });
      closeCreateModal();
      setNewOrgName('');
      setNewOrgDetails(noDetails);
    },
  });

  // Update institution mutation
  const updateMutation = useMutation({
    mutationFn: async ({ id, name }: { id: string; name: string }) => {
      const response = await api.patchApiInstitutionsId(id, { name });
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.institutions });
      closeEditModal();
      setEditingOrg(null);
      setEditName('');
    },
  });

  // Delete institution mutation
  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      await api.deleteApiInstitutionsId(id);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.institutions });
      closeDeleteModal();
      setOrgToDelete(null);
    },
  });

  // Send invite mutation
  const inviteMutation = useMutation({
    mutationFn: async ({ institutionId, email }: { institutionId: string; email: string }) => {
      const response = await api.postApiInvitesInstitution({ 
        institutionId, 
        role: 'head',
        invitedEmail: email 
      });
      return response.data;
    },
    onSuccess: () => {
      setInviteError(null);
      queryClient.invalidateQueries({ queryKey: queryKeys.invites });
      closeInviteModal();
      setInviteOrg(null);
      setInviteEmail('');
    },
    onError: (error) => {
      setInviteError(getErrorMessage(error));
    },
  });

  const handleEdit = (org: OrganisationView) => {
    setEditingOrg(org);
    setEditName(org.name || '');
    openEditModal();
  };

  const handleInvite = (org: OrganisationView) => {
    setInviteOrg(org);
    setInviteEmail('');
    setInviteError(null);
    openInviteModal();
  };

  const handleDelete = (org: OrganisationView) => {
    setOrgToDelete(org);
    openDeleteModal();
  };

  if (error) {
    return (
      <Container size="xl" py="xl">
        <Alert icon={<IconAlertCircle size={16} />} title={t('error')} color="red">
          {error instanceof Error ? error.message : 'Failed to load organizations'}
        </Alert>
      </Container>
    );
  }

  return (
    <Container size="xl" py="xl">
      <Stack gap="lg">
        <Group justify="space-between" align="center">
          <Title order={2}>{t('admin.organizations.title')}</Title>
          {isMobile ? (
            <Tooltip label={t('admin.organizations.create')} withArrow>
              <PlusIconButton onClick={openCreateModal} variant="filled" aria-label={t('admin.organizations.create')} />
            </Tooltip>
          ) : (
            <Button leftSection={<IconPlus size={16} />} onClick={openCreateModal}>
              {t('admin.organizations.create')}
            </Button>
          )}
        </Group>

        <Card withBorder pos="relative" p={isMobile ? 'sm' : undefined}>
          <LoadingOverlay visible={isLoading} />
          
          {isMobile ? (
            <Group gap="sm" wrap="nowrap" mb="md">
              <ExpandableSearch
                value={searchQuery}
                onChange={setSearchQuery}
                placeholder={t('admin.organizations.searchPlaceholder')}
              />
            </Group>
          ) : (
            <TextInput
              placeholder={t('admin.organizations.searchPlaceholder')}
              leftSection={<IconSearch size={16} />}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.currentTarget.value)}
              mb="md"
            />
          )}
          
          {filteredOrganizations && filteredOrganizations.length > 0 ? (
            isMobile ? (
              <Stack gap="sm">
                {filteredOrganizations.map((org) => (
                  <Card key={org.id} withBorder padding="sm" radius="md">
                    <Stack gap="xs">
                      <Group justify="space-between" wrap="nowrap">
                        <Text fw={600} lineClamp={1}>{org.name}</Text>
                        <Badge variant="light" color="blue" size="sm">
                          {org.members?.length || 0} {t('admin.organizations.membersCount')}
                        </Badge>
                      </Group>
                      {(org.operator || org.city) && (
                        <Text size="xs" c="dimmed">{[org.operator, org.city].filter(Boolean).join(' · ')}</Text>
                      )}
                      {headNames(org) && (
                        <Text size="xs">{t('admin.organizations.heads')}: {headNames(org)}</Text>
                      )}
                      <OrganizationStatusBadges org={org} />
                      <Group gap="xs" justify="flex-end">
                        <ActionIcon
                          variant="subtle"
                          color="blue"
                          size="sm"
                          onClick={() => handleInvite(org)}
                          title={t('admin.organizations.sendInvite')}
                        >
                          <IconSend size={14} />
                        </ActionIcon>
                        <ActionIcon
                          variant="subtle"
                          color="gray"
                          size="sm"
                          onClick={() => handleEdit(org)}
                          title={t('edit')}
                        >
                          <IconEdit size={14} />
                        </ActionIcon>
                        <ActionIcon
                          variant="subtle"
                          color={org.suspendedAt ? 'green' : 'orange'}
                          size="sm"
                          onClick={() => setSuspendOrg(org)}
                          title={org.suspendedAt ? t('admin.organizations.lift') : t('admin.organizations.suspend')}
                        >
                          {org.suspendedAt ? <IconLockOpen size={14} /> : <IconLock size={14} />}
                        </ActionIcon>
                        <ActionIcon
                          variant="subtle"
                          color="red"
                          size="sm"
                          onClick={() => handleDelete(org)}
                          title={t('delete')}
                        >
                          <IconTrash size={14} />
                        </ActionIcon>
                      </Group>
                    </Stack>
                  </Card>
                ))}
              </Stack>
            ) : (
              <Table striped highlightOnHover>
                <Table.Thead>
                  <Table.Tr>
                    <Table.Th>{t('admin.organizations.name')}</Table.Th>
                    <Table.Th>{t('admin.organizations.heads')}</Table.Th>
                    <Table.Th>{t('admin.organizations.status')}</Table.Th>
                    <Table.Th>{t('admin.organizations.members')}</Table.Th>
                    <Table.Th style={{ width: 190 }}>{t('admin.organizations.actions')}</Table.Th>
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {filteredOrganizations.map((org) => (
                    <OrganizationMembersRow
                      key={org.id}
                      org={org}
                      onEdit={handleEdit}
                      onDelete={handleDelete}
                      onInvite={handleInvite}
                      onToggleSuspension={setSuspendOrg}
                    />
                  ))}
                </Table.Tbody>
              </Table>
            )
          ) : (
            <Text c="dimmed" ta="center" py="xl">
              {t('admin.organizations.empty')}
            </Text>
          )}
        </Card>

        {/* Organisations that registered themselves lately */}
        <Card shadow="sm" padding="lg" radius="md" withBorder>
          <Title order={3} mb={4}>{t('admin.registrations.title')}</Title>
          <Text size="sm" c="dimmed" mb="md">{t('admin.registrations.description')}</Text>
          <RecentRegistrationsList />
        </Card>

        {/* Pending Invites Section */}
        <Card shadow="sm" padding="lg" radius="md" withBorder>
          <Title order={3} mb="md">{t('admin.invites.title')}</Title>
          <InvitesList showInstitutionColumn={true} />
        </Card>
      </Stack>

      {/* Create Modal */}
      <Modal opened={createModalOpened} onClose={closeCreateModal} title={t('admin.organizations.createTitle')}>
        <Stack gap="md">
          <TextInput
            label={t('admin.organizations.name')}
            placeholder={t('admin.organizations.namePlaceholder')}
            value={newOrgName}
            onChange={(e) => setNewOrgName(e.currentTarget.value)}
          />
          <Text size="sm" c="dimmed">{t('admin.organizations.detailsOptional')}</Text>
          <TextInput
            label={t('admin.organizations.operator')}
            value={newOrgDetails.operator}
            onChange={(e) => { const operator = e.currentTarget.value; setNewOrgDetails((d) => ({ ...d, operator })); }}
          />
          <TextInput
            label={t('admin.organizations.street')}
            value={newOrgDetails.street}
            onChange={(e) => { const street = e.currentTarget.value; setNewOrgDetails((d) => ({ ...d, street })); }}
          />
          <Group grow>
            <TextInput
              label={t('admin.organizations.postalCode')}
              value={newOrgDetails.postalCode}
              onChange={(e) => { const postalCode = e.currentTarget.value; setNewOrgDetails((d) => ({ ...d, postalCode })); }}
            />
            <TextInput
              label={t('admin.organizations.city')}
              value={newOrgDetails.city}
              onChange={(e) => { const city = e.currentTarget.value; setNewOrgDetails((d) => ({ ...d, city })); }}
            />
          </Group>
          <Group justify="flex-end">
            <Button variant="subtle" onClick={closeCreateModal}>
              {t('cancel')}
            </Button>
            <Button
              onClick={() => createMutation.mutate(newOrgName)}
              loading={createMutation.isPending}
              disabled={!newOrgName.trim()}
            >
              {t('create')}
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Edit Modal */}
      <Modal opened={editModalOpened} onClose={closeEditModal} title={t('admin.organizations.editTitle')}>
        <Stack gap="md">
          <TextInput
            label={t('admin.organizations.name')}
            value={editName}
            onChange={(e) => setEditName(e.currentTarget.value)}
          />
          <Group justify="flex-end">
            <Button variant="subtle" onClick={closeEditModal}>
              {t('cancel')}
            </Button>
            <Button
              onClick={() => editingOrg?.id && updateMutation.mutate({ id: editingOrg.id, name: editName })}
              loading={updateMutation.isPending}
              disabled={!editName.trim()}
            >
              {t('save')}
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Invite Modal */}
      <Modal opened={inviteModalOpened} onClose={() => { setInviteError(null); closeInviteModal(); }} title={t('admin.organizations.inviteTitle')}>
        <Stack gap="md">
          <Text size="sm" c="dimmed">
            {t('admin.organizations.inviteDescription', { name: inviteOrg?.name })}
          </Text>
          <TextInput
            label={t('admin.organizations.email')}
            placeholder={t('admin.organizations.emailPlaceholder')}
            type="email"
            value={inviteEmail}
            onChange={(e) => setInviteEmail(e.currentTarget.value)}
          />
          {inviteError && (
            <Alert color="red" icon={<IconAlertCircle size={16} />}>
              {inviteError}
            </Alert>
          )}
          <Group justify="flex-end">
            <Button variant="subtle" onClick={closeInviteModal}>
              {t('cancel')}
            </Button>
            <Button
              onClick={() => inviteOrg?.id && inviteMutation.mutate({ institutionId: inviteOrg.id, email: inviteEmail })}
              loading={inviteMutation.isPending}
              disabled={!inviteEmail.trim()}
              leftSection={<IconSend size={16} />}
            >
              {t('admin.organizations.sendInvite')}
            </Button>
          </Group>
        </Stack>
      </Modal>

      <OrganizationSuspendModal org={suspendOrg} onClose={() => setSuspendOrg(null)} />

      {/* Delete Confirmation Modal */}
      <Modal opened={deleteModalOpened} onClose={closeDeleteModal} title={t('admin.organizations.deleteTitle')}>
        <Stack gap="md">
          <Text>
            {t('admin.organizations.deleteConfirm', { name: orgToDelete?.name })}
          </Text>
          <Group justify="flex-end">
            <Button variant="subtle" onClick={closeDeleteModal}>
              {t('cancel')}
            </Button>
            <Button
              color="red"
              onClick={() => orgToDelete?.id && deleteMutation.mutate(orgToDelete.id)}
              loading={deleteMutation.isPending}
            >
              {t('delete')}
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  );
}

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
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { IconPlus, IconAlertCircle, IconSend, IconEdit, IconTrash, IconSearch } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useRequiredAuthenticatedApi } from '@/api/useAuthenticatedApi';
import { useResponsiveDesign } from '@/common/hooks/useResponsiveDesign';
import type { ObjInstitution } from '@/api/generated';
import { InvitesList } from './InvitesList';
import { OrganizationMembersRow } from './OrganizationMembersRow';

export function OrganizationsManagement() {
  const { t } = useTranslation('common');
  const api = useRequiredAuthenticatedApi();
  const queryClient = useQueryClient();
  const { isMobile } = useResponsiveDesign();
  
  const [editingOrg, setEditingOrg] = useState<ObjInstitution | null>(null);
  const [inviteOrg, setInviteOrg] = useState<ObjInstitution | null>(null);
  const [newOrgName, setNewOrgName] = useState('');
  const [editName, setEditName] = useState('');
  const [inviteEmail, setInviteEmail] = useState('');
  
  const [createModalOpened, { open: openCreateModal, close: closeCreateModal }] = useDisclosure(false);
  const [editModalOpened, { open: openEditModal, close: closeEditModal }] = useDisclosure(false);
  const [inviteModalOpened, { open: openInviteModal, close: closeInviteModal }] = useDisclosure(false);
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] = useDisclosure(false);
  const [orgToDelete, setOrgToDelete] = useState<ObjInstitution | null>(null);
  const [searchQuery, setSearchQuery] = useState('');

  // Fetch all institutions
  const { data: organizations, isLoading, error } = useQuery({
    queryKey: ['institutions'],
    queryFn: async () => {
      const response = await api.institutions.institutionsList();
      return response.data;
    },
  });

  // Filter organizations by search query
  const filteredOrganizations = useMemo(() => {
    if (!organizations) return [];
    if (!searchQuery.trim()) return organizations;
    
    const query = searchQuery.toLowerCase();
    return organizations.filter((org) => 
      org.name?.toLowerCase().includes(query)
    );
  }, [organizations, searchQuery]);

  // Create institution mutation
  const createMutation = useMutation({
    mutationFn: async (name: string) => {
      const response = await api.institutions.institutionsCreate({ name });
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['institutions'] });
      closeCreateModal();
      setNewOrgName('');
    },
  });

  // Update institution mutation
  const updateMutation = useMutation({
    mutationFn: async ({ id, name }: { id: string; name: string }) => {
      const response = await api.institutions.institutionsPartialUpdate(id, { name });
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['institutions'] });
      closeEditModal();
      setEditingOrg(null);
      setEditName('');
    },
  });

  // Delete institution mutation
  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      await api.institutions.institutionsDelete(id);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['institutions'] });
      closeDeleteModal();
      setOrgToDelete(null);
    },
  });

  // Send invite mutation
  const inviteMutation = useMutation({
    mutationFn: async ({ institutionId, email }: { institutionId: string; email: string }) => {
      const response = await api.invites.institutionCreate({ 
        institutionId, 
        role: 'head',
        invitedEmail: email 
      });
      return response.data;
    },
    onSuccess: () => {
      closeInviteModal();
      setInviteOrg(null);
      setInviteEmail('');
    },
  });

  const handleEdit = (org: ObjInstitution) => {
    setEditingOrg(org);
    setEditName(org.name || '');
    openEditModal();
  };

  const handleInvite = (org: ObjInstitution) => {
    setInviteOrg(org);
    setInviteEmail('');
    openInviteModal();
  };

  const handleDelete = (org: ObjInstitution) => {
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
          <Button leftSection={<IconPlus size={16} />} onClick={openCreateModal}>
            {t('admin.organizations.create')}
          </Button>
        </Group>

        <Card withBorder pos="relative" p={isMobile ? 'sm' : undefined}>
          <LoadingOverlay visible={isLoading} />
          
          <TextInput
            placeholder={t('admin.organizations.searchPlaceholder')}
            leftSection={<IconSearch size={16} />}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.currentTarget.value)}
            mb="md"
          />
          
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
                      <Group gap="xs">
                        <ActionIcon
                          variant="light"
                          color="blue"
                          size="sm"
                          onClick={() => handleInvite(org)}
                          title={t('admin.organizations.sendInvite')}
                        >
                          <IconSend size={14} />
                        </ActionIcon>
                        <ActionIcon
                          variant="light"
                          color="gray"
                          size="sm"
                          onClick={() => handleEdit(org)}
                          title={t('edit')}
                        >
                          <IconEdit size={14} />
                        </ActionIcon>
                        <ActionIcon
                          variant="light"
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
                    <Table.Th>{t('admin.organizations.members')}</Table.Th>
                    <Table.Th style={{ width: 150 }}>{t('admin.organizations.actions')}</Table.Th>
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
      <Modal opened={inviteModalOpened} onClose={closeInviteModal} title={t('admin.organizations.inviteTitle')}>
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

import { useState, useMemo, useCallback } from 'react';
import {
  Container,
  Title,
  Text,
  Stack,
  Card,
  Group,
  TextInput,
  Table,
  Modal,
  LoadingOverlay,
  Alert,
  Badge,
  ActionIcon,
  Switch,
  Tooltip,
} from '@mantine/core';
import { useDebouncedValue, useDisclosure } from '@mantine/hooks';
import { IconTrash, IconAlertCircle, IconSearch, IconShieldCheck, IconShieldOff } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useRequiredAuthenticatedApi } from '@/api/useAuthenticatedApi';
import { useResponsiveDesign } from '@/common/hooks/useResponsiveDesign';
import { SortSelector, type SortOption } from '@/common/components/controls';
import { useAuth } from '@/providers/AuthProvider';
import type { ObjUser, ObjRole } from '@/api/generated';

type SortField = 'name' | 'email' | 'role' | 'organization' | 'joined';

const getRoleColor = (role?: ObjRole): string => {
  switch (role) {
    case 'admin':
      return 'red';
    case 'head':
      return 'violet';
    case 'staff':
      return 'blue';
    case 'participant':
      return 'gray';
    default:
      return 'gray';
  }
};

export function UserManagement() {
  const { t } = useTranslation('common');
  const { t: tAuth } = useTranslation('auth');
  const api = useRequiredAuthenticatedApi();
  const queryClient = useQueryClient();
  const { isMobile } = useResponsiveDesign();
  const { backendUser } = useAuth();

  const [searchQuery, setSearchQuery] = useState('');
  const [debouncedSearch] = useDebouncedValue(searchQuery, 300);
  const [hideGuests, setHideGuests] = useState(true);
  const [sortValue, setSortValue] = useState('name-asc');
  const [userToDelete, setUserToDelete] = useState<ObjUser | null>(null);
  const [userToPromote, setUserToPromote] = useState<ObjUser | null>(null);
  const [userToRemoveAdmin, setUserToRemoveAdmin] = useState<ObjUser | null>(null);
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] = useDisclosure(false);
  const [promoteModalOpened, { open: openPromoteModal, close: closePromoteModal }] = useDisclosure(false);
  const [removeAdminModalOpened, { open: openRemoveAdminModal, close: closeRemoveAdminModal }] = useDisclosure(false);

  // Fetch all users
  const { data: users, isLoading, error } = useQuery({
    queryKey: ['admin-users'],
    queryFn: async () => {
      const response = await api.users.usersList();
      return response.data;
    },
  });

  // Delete user mutation
  const deleteMutation = useMutation({
    mutationFn: async (userId: string) => {
      await api.users.usersDelete(userId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-users'] });
      closeDeleteModal();
      setUserToDelete(null);
    },
  });

  // Make admin mutation
  const makeAdminMutation = useMutation({
    mutationFn: async (userId: string) => {
      await api.users.roleCreate(userId, { role: 'admin' });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-users'] });
      closePromoteModal();
      setUserToPromote(null);
    },
  });

  // Remove admin mutation
  const removeAdminMutation = useMutation({
    mutationFn: async (userId: string) => {
      await api.users.roleDelete(userId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-users'] });
      closeRemoveAdminModal();
      setUserToRemoveAdmin(null);
    },
  });

  const translateRole = useCallback((role?: string) => {
    if (!role) return t('admin.users.noRole');
    const roleKey = role.toLowerCase();
    return tAuth(`profile.roles.${roleKey}`, role);
  }, [t, tAuth]);

  const formatDate = useCallback((dateStr?: string) => {
    if (!dateStr) return '-';
    return new Date(dateStr).toLocaleDateString();
  }, []);

  const isCurrentUser = useCallback((userId?: string) => {
    return userId === backendUser?.id;
  }, [backendUser?.id]);

  const isAdmin = useCallback((user: ObjUser) => {
    return user.role?.role === 'admin';
  }, []);

  const hasNoRole = useCallback((user: ObjUser) => {
    return !user.role || !user.role.role;
  }, []);

  const canPromoteToAdmin = useCallback((user: ObjUser) => {
    return !isCurrentUser(user.id) && hasNoRole(user);
  }, [isCurrentUser, hasNoRole]);

  const canRemoveAdmin = useCallback((user: ObjUser) => {
    return !isCurrentUser(user.id) && isAdmin(user);
  }, [isCurrentUser, isAdmin]);

  const getOrganizationName = useCallback((user: ObjUser) => {
    return user.role?.institution?.name || '-';
  }, []);

  const handleDelete = useCallback((user: ObjUser) => {
    setUserToDelete(user);
    openDeleteModal();
  }, [openDeleteModal]);

  const handlePromote = useCallback((user: ObjUser) => {
    setUserToPromote(user);
    openPromoteModal();
  }, [openPromoteModal]);

  const handleRemoveAdmin = useCallback((user: ObjUser) => {
    setUserToRemoveAdmin(user);
    openRemoveAdminModal();
  }, [openRemoveAdminModal]);

  // Parse combined sort value into field and direction
  const [sortField, sortDirection] = sortValue.split('-') as [SortField, 'asc' | 'desc'];

  const sortOptions: SortOption[] = useMemo(() => [
    { value: 'name-asc', label: t('admin.users.sort.name-asc') },
    { value: 'name-desc', label: t('admin.users.sort.name-desc') },
    { value: 'email-asc', label: t('admin.users.sort.email-asc') },
    { value: 'email-desc', label: t('admin.users.sort.email-desc') },
    { value: 'role-asc', label: t('admin.users.sort.role-asc') },
    { value: 'role-desc', label: t('admin.users.sort.role-desc') },
    { value: 'organization-asc', label: t('admin.users.sort.organization-asc') },
    { value: 'organization-desc', label: t('admin.users.sort.organization-desc') },
    { value: 'joined-asc', label: t('admin.users.sort.joined-asc') },
    { value: 'joined-desc', label: t('admin.users.sort.joined-desc') },
  ], [t]);

  // Filter users based on search and guest toggle
  const filteredUsers = useMemo(() => {
    if (!users) return [];
    
    return users.filter((user) => {
      // Filter out guests if toggle is on
      if (hideGuests && user.role?.role === 'participant') {
        return false;
      }

      // Search filter
      if (debouncedSearch) {
        const search = debouncedSearch.toLowerCase();
        const name = (user.name || '').toLowerCase();
        const email = (user.email || '').toLowerCase();
        const org = getOrganizationName(user).toLowerCase();
        const role = (user.role?.role || '').toLowerCase();
        
        return name.includes(search) || 
               email.includes(search) || 
               org.includes(search) ||
               role.includes(search);
      }
      
      return true;
    });
  }, [users, hideGuests, debouncedSearch, getOrganizationName]);

  // Sort filtered users
  const sortedUsers = useMemo(() => {
    if (filteredUsers.length === 0) return [];
    
    return [...filteredUsers].sort((a, b) => {
      let aVal: string;
      let bVal: string;

      switch (sortField) {
        case 'name':
          aVal = a.name || '';
          bVal = b.name || '';
          break;
        case 'email':
          aVal = a.email || '';
          bVal = b.email || '';
          break;
        case 'role':
          aVal = a.role?.role || '';
          bVal = b.role?.role || '';
          break;
        case 'organization':
          aVal = getOrganizationName(a);
          bVal = getOrganizationName(b);
          break;
        case 'joined':
          aVal = a.meta?.createdAt || '';
          bVal = b.meta?.createdAt || '';
          break;
        default:
          return 0;
      }

      const comparison = aVal.localeCompare(bVal);
      return sortDirection === 'asc' ? comparison : -comparison;
    });
  }, [filteredUsers, sortField, sortDirection, getOrganizationName]);

  // Count guests for display
  const guestCount = useMemo(() => {
    if (!users) return 0;
    return users.filter((u) => u.role?.role === 'participant').length;
  }, [users]);

  if (error) {
    return (
      <Container size="xl" py="xl">
        <Alert icon={<IconAlertCircle size={16} />} title={t('error')} color="red">
          {error instanceof Error ? error.message : t('admin.users.loadError')}
        </Alert>
      </Container>
    );
  }

  return (
    <Container size="xl" py="xl">
      <Stack gap="lg">
        <Title order={2}>{t('admin.users.title')}</Title>

        {/* Filters */}
        <Card withBorder p="md">
          <Group justify="space-between" wrap="wrap" gap="md">
            <TextInput
              placeholder={t('admin.users.searchPlaceholder')}
              leftSection={<IconSearch size={16} />}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.currentTarget.value)}
              style={{ flex: 1, minWidth: 200 }}
            />
            <Group gap="md" wrap="wrap">
              <Switch
                label={t('admin.users.hideGuests', { count: guestCount })}
                checked={hideGuests}
                onChange={(e) => setHideGuests(e.currentTarget.checked)}
              />
              <SortSelector
                options={sortOptions}
                value={sortValue}
                onChange={setSortValue}
                label={t('admin.users.sort.label')}
                width={200}
              />
            </Group>
          </Group>
        </Card>

        {/* Users Table/Cards */}
        <Card withBorder pos="relative" p={isMobile ? 'sm' : undefined}>
          <LoadingOverlay visible={isLoading} />
          
          {sortedUsers.length > 0 ? (
            isMobile ? (
              <Stack gap="sm">
                {sortedUsers.map((user) => (
                  <Card key={user.id} withBorder padding="sm" radius="md" bg={isCurrentUser(user.id) ? 'violet.0' : undefined}>
                    <Stack gap="xs">
                      <Group justify="space-between" wrap="nowrap">
                        <Group gap="xs">
                          <Text fw={600} lineClamp={1}>{user.name}</Text>
                          {isCurrentUser(user.id) && (
                            <Badge size="xs" variant="filled" color="violet">{t('admin.users.me')}</Badge>
                          )}
                        </Group>
                        {user.role?.role && (
                          <Badge 
                            variant="light" 
                            color={getRoleColor(user.role.role as ObjRole)} 
                            size="sm"
                          >
                            {translateRole(user.role.role)}
                          </Badge>
                        )}
                      </Group>
                      {user.email && (
                        <Text size="sm" c="dimmed" lineClamp={1}>{user.email}</Text>
                      )}
                      {user.role?.institution?.name && (
                        <Text size="xs" c="dimmed">{user.role.institution.name}</Text>
                      )}
                      <Group justify="space-between">
                        <Text size="xs" c="dimmed">{t('admin.users.joined')}: {formatDate(user.meta?.createdAt)}</Text>
                        <Group gap="xs">
                          {canPromoteToAdmin(user) && (
                            <Tooltip label={t('admin.users.makeAdmin')}>
                              <ActionIcon
                                variant="light"
                                color="green"
                                size="sm"
                                onClick={() => handlePromote(user)}
                              >
                                <IconShieldCheck size={14} />
                              </ActionIcon>
                            </Tooltip>
                          )}
                          {canRemoveAdmin(user) && (
                            <Tooltip label={t('admin.users.removeAdmin')}>
                              <ActionIcon
                                variant="light"
                                color="red"
                                size="sm"
                                onClick={() => handleRemoveAdmin(user)}
                              >
                                <IconShieldOff size={14} />
                              </ActionIcon>
                            </Tooltip>
                          )}
                          {!isCurrentUser(user.id) && (
                            <ActionIcon
                              variant="light"
                              color="red"
                              size="sm"
                              onClick={() => handleDelete(user)}
                              title={t('delete')}
                            >
                              <IconTrash size={14} />
                            </ActionIcon>
                          )}
                        </Group>
                      </Group>
                    </Stack>
                  </Card>
                ))}
              </Stack>
            ) : (
              <Table striped highlightOnHover>
                <Table.Thead>
                  <Table.Tr>
                    <Table.Th>{t('admin.users.name')}</Table.Th>
                    <Table.Th>{t('admin.users.email')}</Table.Th>
                    <Table.Th>{t('admin.users.role')}</Table.Th>
                    <Table.Th>{t('admin.users.organization')}</Table.Th>
                    <Table.Th>{t('admin.users.joined')}</Table.Th>
                    <Table.Th style={{ width: 100 }}>{t('admin.users.actions')}</Table.Th>
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {sortedUsers.map((user) => (
                    <Table.Tr key={user.id} bg={isCurrentUser(user.id) ? 'violet.0' : undefined}>
                      <Table.Td>
                        <Group gap="xs">
                          <Text fw={500}>{user.name}</Text>
                          {isCurrentUser(user.id) && (
                            <Badge size="xs" variant="filled" color="violet">{t('admin.users.me')}</Badge>
                          )}
                        </Group>
                      </Table.Td>
                      <Table.Td>
                        <Text size="sm" c="dimmed">{user.email || '-'}</Text>
                      </Table.Td>
                      <Table.Td>
                        {user.role?.role ? (
                          <Badge 
                            variant="light" 
                            color={getRoleColor(user.role.role as ObjRole)}
                          >
                            {translateRole(user.role.role)}
                          </Badge>
                        ) : (
                          <Text size="sm" c="dimmed">{t('admin.users.noRole')}</Text>
                        )}
                      </Table.Td>
                      <Table.Td>
                        <Text size="sm">{getOrganizationName(user)}</Text>
                      </Table.Td>
                      <Table.Td>
                        <Text size="sm" c="dimmed">{formatDate(user.meta?.createdAt)}</Text>
                      </Table.Td>
                      <Table.Td>
                        <Group gap="xs">
                          {canPromoteToAdmin(user) && (
                            <Tooltip label={t('admin.users.makeAdmin')}>
                              <ActionIcon
                                variant="subtle"
                                color="green"
                                onClick={() => handlePromote(user)}
                              >
                                <IconShieldCheck size={16} />
                              </ActionIcon>
                            </Tooltip>
                          )}
                          {canRemoveAdmin(user) && (
                            <Tooltip label={t('admin.users.removeAdmin')}>
                              <ActionIcon
                                variant="subtle"
                                color="red"
                                onClick={() => handleRemoveAdmin(user)}
                              >
                                <IconShieldOff size={16} />
                              </ActionIcon>
                            </Tooltip>
                          )}
                          {!isCurrentUser(user.id) && (
                            <ActionIcon
                              variant="subtle"
                              color="red"
                              onClick={() => handleDelete(user)}
                              title={t('delete')}
                            >
                              <IconTrash size={16} />
                            </ActionIcon>
                          )}
                        </Group>
                      </Table.Td>
                    </Table.Tr>
                  ))}
                </Table.Tbody>
              </Table>
            )
          ) : (
            <Text c="dimmed" ta="center" py="xl">
              {debouncedSearch ? t('admin.users.noResults') : t('admin.users.empty')}
            </Text>
          )}
        </Card>

        {/* Stats */}
        <Text size="sm" c="dimmed" ta="right">
          {t('admin.users.showing', { 
            count: sortedUsers.length, 
            total: users?.length || 0 
          })}
        </Text>
      </Stack>

      {/* Delete Confirmation Modal */}
      <Modal 
        opened={deleteModalOpened} 
        onClose={closeDeleteModal} 
        title={t('admin.users.deleteTitle')}
      >
        <Stack gap="md">
          <Text>
            {t('admin.users.deleteConfirm', { name: userToDelete?.name })}
          </Text>
          <Group justify="flex-end">
            <Text 
              size="sm" 
              c="dimmed" 
              style={{ cursor: 'pointer' }}
              onClick={closeDeleteModal}
            >
              {t('cancel')}
            </Text>
            <ActionIcon
              color="red"
              variant="filled"
              size="lg"
              onClick={() => userToDelete?.id && deleteMutation.mutate(userToDelete.id)}
              loading={deleteMutation.isPending}
            >
              <IconTrash size={16} />
            </ActionIcon>
          </Group>
        </Stack>
      </Modal>

      {/* Promote to Admin Confirmation Modal */}
      <Modal 
        opened={promoteModalOpened} 
        onClose={closePromoteModal} 
        title={t('admin.users.promoteTitle')}
      >
        <Stack gap="md">
          <Text>
            {t('admin.users.promoteConfirm', { 
              name: userToPromote?.name,
              email: userToPromote?.email || t('admin.users.noEmail')
            })}
          </Text>
          <Alert color="yellow" icon={<IconAlertCircle size={16} />}>
            {t('admin.users.promoteWarning')}
          </Alert>
          <Group justify="flex-end">
            <Text 
              size="sm" 
              c="dimmed" 
              style={{ cursor: 'pointer' }}
              onClick={closePromoteModal}
            >
              {t('cancel')}
            </Text>
            <ActionIcon
              color="green"
              variant="filled"
              size="lg"
              onClick={() => userToPromote?.id && makeAdminMutation.mutate(userToPromote.id)}
              loading={makeAdminMutation.isPending}
            >
              <IconShieldCheck size={16} />
            </ActionIcon>
          </Group>
        </Stack>
      </Modal>

      {/* Remove Admin Confirmation Modal */}
      <Modal 
        opened={removeAdminModalOpened} 
        onClose={closeRemoveAdminModal} 
        title={t('admin.users.removeAdminTitle')}
      >
        <Stack gap="md">
          <Text>
            {t('admin.users.removeAdminConfirm', { 
              name: userToRemoveAdmin?.name,
              email: userToRemoveAdmin?.email || t('admin.users.noEmail')
            })}
          </Text>
          <Alert color="orange" icon={<IconAlertCircle size={16} />}>
            {t('admin.users.removeAdminWarning')}
          </Alert>
          <Group justify="flex-end">
            <Text 
              size="sm" 
              c="dimmed" 
              style={{ cursor: 'pointer' }}
              onClick={closeRemoveAdminModal}
            >
              {t('cancel')}
            </Text>
            <ActionIcon
              color="red"
              variant="filled"
              size="lg"
              onClick={() => userToRemoveAdmin?.id && removeAdminMutation.mutate(userToRemoveAdmin.id)}
              loading={removeAdminMutation.isPending}
            >
              <IconShieldOff size={16} />
            </ActionIcon>
          </Group>
        </Stack>
      </Modal>
    </Container>
  );
}

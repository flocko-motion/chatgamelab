import { useState } from 'react';
import {
  Table,
  Text,
  Badge,
  Group,
  ActionIcon,
  Collapse,
  Stack,
  Loader,
  Box,
} from '@mantine/core';
import { IconChevronDown, IconChevronRight, IconTrash, IconSend, IconEdit } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useRequiredAuthenticatedApi } from '@/api/useAuthenticatedApi';
import type { ObjInstitution, ObjUser } from '@/api/generated';

interface OrganizationMembersRowProps {
  org: ObjInstitution;
  onEdit: (org: ObjInstitution) => void;
  onDelete: (org: ObjInstitution) => void;
  onInvite: (org: ObjInstitution) => void;
}

export function OrganizationMembersRow({ org, onEdit, onDelete, onInvite }: OrganizationMembersRowProps) {
  const { t } = useTranslation('common');
  const { t: tAuth } = useTranslation('auth');
  const api = useRequiredAuthenticatedApi();
  const queryClient = useQueryClient();
  const [expanded, setExpanded] = useState(false);

  const { data: members, isLoading: membersLoading } = useQuery({
    queryKey: ['institution-members', org.id],
    queryFn: async () => {
      const response = await api.institutions.membersList(org.id!);
      return response.data;
    },
    enabled: expanded,
  });

  const removeMemberMutation = useMutation({
    mutationFn: async (userId: string) => {
      await api.institutions.membersDelete(org.id!, userId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['institution-members', org.id] });
      queryClient.invalidateQueries({ queryKey: ['institutions'] });
    },
  });

  const handleRemoveMember = (userId: string, userName: string) => {
    if (confirm(t('admin.organizations.removeMemberConfirm', { name: userName }))) {
      removeMemberMutation.mutate(userId);
    }
  };

  const translateRole = (role?: string) => {
    if (!role) return '-';
    const roleKey = role.toLowerCase();
    return tAuth(`profile.roles.${roleKey}`, role);
  };

  return (
    <>
      <Table.Tr 
        style={{ cursor: 'pointer' }}
        onClick={() => setExpanded(!expanded)}
      >
        <Table.Td>
          <Group gap="xs">
            <ActionIcon variant="subtle" size="sm">
              {expanded ? <IconChevronDown size={16} /> : <IconChevronRight size={16} />}
            </ActionIcon>
            <Text fw={500}>{org.name}</Text>
          </Group>
        </Table.Td>
        <Table.Td>
          <Badge variant="light" color="blue">
            {org.members?.length || 0} {t('admin.organizations.membersCount')}
          </Badge>
        </Table.Td>
        <Table.Td onClick={(e) => e.stopPropagation()}>
          <Group gap="xs">
            <ActionIcon
              variant="subtle"
              color="blue"
              onClick={() => onInvite(org)}
              title={t('admin.organizations.sendInvite')}
            >
              <IconSend size={16} />
            </ActionIcon>
            <ActionIcon
              variant="subtle"
              color="gray"
              onClick={() => onEdit(org)}
              title={t('edit')}
            >
              <IconEdit size={16} />
            </ActionIcon>
            <ActionIcon
              variant="subtle"
              color="red"
              onClick={() => onDelete(org)}
              title={t('delete')}
            >
              <IconTrash size={16} />
            </ActionIcon>
          </Group>
        </Table.Td>
      </Table.Tr>
      {expanded && (
        <Table.Tr>
          <Table.Td colSpan={3} style={{ padding: 0 }}>
            <Collapse in={expanded}>
              <Box bg="gray.0" p="md">
                {membersLoading ? (
                  <Group justify="center" p="md">
                    <Loader size="sm" />
                  </Group>
                ) : members && members.length > 0 ? (
                  <Stack gap="xs">
                    <Text size="sm" fw={600} c="dimmed">{t('admin.organizations.membersList')}</Text>
                    <Table>
                      <Table.Thead>
                        <Table.Tr>
                          <Table.Th>{t('admin.organizations.memberName')}</Table.Th>
                          <Table.Th>{t('admin.organizations.memberEmail')}</Table.Th>
                          <Table.Th>{t('admin.organizations.memberRole')}</Table.Th>
                          <Table.Th style={{ width: 80 }}>{t('admin.organizations.actions')}</Table.Th>
                        </Table.Tr>
                      </Table.Thead>
                      <Table.Tbody>
                        {members.map((member: ObjUser) => (
                          <Table.Tr key={member.id}>
                            <Table.Td>
                              <Text size="sm">{member.name || '-'}</Text>
                            </Table.Td>
                            <Table.Td>
                              <Text size="sm" c="dimmed">{member.email || '-'}</Text>
                            </Table.Td>
                            <Table.Td>
                              <Badge size="sm" variant="light">
                                {translateRole(member.role?.role)}
                              </Badge>
                            </Table.Td>
                            <Table.Td>
                              <ActionIcon
                                variant="subtle"
                                color="red"
                                size="sm"
                                onClick={() => handleRemoveMember(member.id!, member.name || member.email || 'this user')}
                                loading={removeMemberMutation.isPending}
                                title={t('admin.organizations.removeMember')}
                              >
                                <IconTrash size={14} />
                              </ActionIcon>
                            </Table.Td>
                          </Table.Tr>
                        ))}
                      </Table.Tbody>
                    </Table>
                  </Stack>
                ) : (
                  <Text size="sm" c="dimmed" ta="center" py="md">
                    {t('admin.organizations.noMembers')}
                  </Text>
                )}
              </Box>
            </Collapse>
          </Table.Td>
        </Table.Tr>
      )}
    </>
  );
}

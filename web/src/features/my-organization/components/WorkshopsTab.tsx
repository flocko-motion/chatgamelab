import { useState } from 'react';
import {
  Stack,
  Card,
  Group,
  Badge,
  Loader,
  Center,
  Modal,
  TextInput,
  Switch,
  Alert,
  CopyButton,
  ActionIcon,
  Tooltip,
  Text,
  Collapse,
  Checkbox,
} from '@mantine/core';
import { useDisclosure, useDebouncedValue } from '@mantine/hooks';
import {
  IconPlus,
  IconTrash,
  IconUserPlus,
  IconCopy,
  IconCheck,
  IconAlertCircle,
  IconPlayerPause,
  IconPlayerPlay,
  IconChevronDown,
  IconChevronRight,
  IconUser,
  IconCalendar,
} from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useResponsiveDesign } from '@/common/hooks/useResponsiveDesign';
import { useWorkshops, useWorkshop, useCreateWorkshop, useUpdateWorkshop, useDeleteWorkshop, useCreateWorkshopInvite, useRevokeInvite } from '@/api/hooks';
import { ExpandableSearch, SortSelector, type SortOption } from '@/common/components/controls';
import { ActionButton } from '@/common/components/buttons/ActionButton';
import { TextButton } from '@/common/components/buttons/TextButton';
import { DangerButton } from '@/common/components/buttons/DangerButton';
import { ConfirmationModal } from './ConfirmationModal';
import type { ObjWorkshop } from '@/api/generated';

interface WorkshopsTabProps {
  institutionId: string;
}

export function WorkshopsTab({ institutionId }: WorkshopsTabProps) {
  const { t } = useTranslation('common');
  useResponsiveDesign(); // Keep hook for future responsive needs

  const [createModalOpened, { open: openCreateModal, close: closeCreateModal }] = useDisclosure(false);
  const [inviteModalOpened, { open: openInviteModal, close: closeInviteModal }] = useDisclosure(false);
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] = useDisclosure(false);

  const [newWorkshopName, setNewWorkshopName] = useState('');
  const [newWorkshopActive, setNewWorkshopActive] = useState(true);
  const [selectedWorkshop, setSelectedWorkshop] = useState<ObjWorkshop | null>(null);
  const [generatedInviteLink, setGeneratedInviteLink] = useState<string | null>(null);
  const [expandedWorkshops, setExpandedWorkshops] = useState<Set<string>>(new Set());

  // Search, filter, and sort state
  const [searchQuery, setSearchQuery] = useState('');
  const [debouncedSearch] = useDebouncedValue(searchQuery, 300);
  const [hideInactive, setHideInactive] = useState(false);
  const [sortValue, setSortValue] = useState('createdAt-desc');

  // Parse sort value
  const [sortField, sortDir] = sortValue.split('-') as ['name' | 'createdAt' | 'participantCount', 'asc' | 'desc'];

  const { data: workshops, isLoading, isError } = useWorkshops({
    institutionId,
    search: debouncedSearch || undefined,
    sortBy: sortField,
    sortDir,
    activeOnly: hideInactive || undefined,
  });
  const { data: selectedWorkshopDetails } = useWorkshop(selectedWorkshop?.id);
  const createWorkshop = useCreateWorkshop();
  const updateWorkshop = useUpdateWorkshop();
  const deleteWorkshop = useDeleteWorkshop();
  const createInvite = useCreateWorkshopInvite();
  const revokeInvite = useRevokeInvite();

  const handleCreateWorkshop = async () => {
    if (!newWorkshopName.trim()) return;

    await createWorkshop.mutateAsync({
      name: newWorkshopName.trim(),
      institutionId,
      active: newWorkshopActive,
      public: false,
    });

    setNewWorkshopName('');
    setNewWorkshopActive(true);
    closeCreateModal();
  };

  const handleOpenInviteModal = (workshop: ObjWorkshop) => {
    setSelectedWorkshop(workshop);
    setGeneratedInviteLink(null);
    openInviteModal();
  };

  const handleToggleActive = async (workshop: ObjWorkshop) => {
    if (!workshop.id) return;
    await updateWorkshop.mutateAsync({
      id: workshop.id,
      name: workshop.name || '',
      active: !workshop.active,
      public: workshop.public,
    });
  };

  const handleGenerateInvite = async () => {
    if (!selectedWorkshop?.id) return;

    const invite = await createInvite.mutateAsync({
      workshopId: selectedWorkshop.id,
    });

    if (invite.inviteToken) {
      const baseUrl = window.location.origin;
      setGeneratedInviteLink(`${baseUrl}/invites/${invite.inviteToken}/accept`);
    }
  };

  const handleOpenDeleteModal = (workshop: ObjWorkshop) => {
    setSelectedWorkshop(workshop);
    openDeleteModal();
  };

  const handleDeleteWorkshop = async () => {
    if (!selectedWorkshop?.id) return;

    await deleteWorkshop.mutateAsync(selectedWorkshop.id);
    closeDeleteModal();
    setSelectedWorkshop(null);
  };


  if (isLoading) {
    return (
      <Center py="xl">
        <Loader />
      </Center>
    );
  }

  if (isError) {
    return (
      <Alert color="red" icon={<IconAlertCircle size={16} />} title={t('error')}>
        {t('myOrganization.workshops.loadError')}
      </Alert>
    );
  }

  const sortOptions: SortOption[] = [
    { value: 'createdAt-desc', label: t('myOrganization.workshops.sort.createdAtDesc') },
    { value: 'createdAt-asc', label: t('myOrganization.workshops.sort.createdAtAsc') },
    { value: 'name-asc', label: t('myOrganization.workshops.sort.nameAsc') },
    { value: 'name-desc', label: t('myOrganization.workshops.sort.nameDesc') },
    { value: 'participantCount-desc', label: t('myOrganization.workshops.sort.participantsDesc') },
    { value: 'participantCount-asc', label: t('myOrganization.workshops.sort.participantsAsc') },
  ];

  return (
    <>
      <Stack gap="lg">
        {/* Header with create button */}
        <Group justify="space-between" align="center" wrap="wrap" gap="sm">
          <Text size="sm" c="dimmed">
            {t('myOrganization.workshops.subtitle')}
          </Text>
          <ActionButton
            onClick={openCreateModal}
            leftSection={<IconPlus size={16} />}
            size="sm"
          >
            {t('myOrganization.workshops.create')}
          </ActionButton>
        </Group>

        {/* Search, filter, and sort controls */}
        <Group justify="space-between" wrap="wrap" gap="sm">
          <Group gap="sm">
            <ExpandableSearch
              value={searchQuery}
              onChange={setSearchQuery}
              placeholder={t('search')}
            />
            <Checkbox
              label={t('myOrganization.workshops.hideInactive')}
              checked={hideInactive}
              onChange={(e) => setHideInactive(e.currentTarget.checked)}
              size="sm"
            />
          </Group>
          <SortSelector
            options={sortOptions}
            value={sortValue}
            onChange={setSortValue}
            label={t('myOrganization.workshops.sort.label')}
          />
        </Group>

        {/* Workshops list */}
        {workshops && workshops.length > 0 ? (
          <Stack gap="md">
            {workshops.map((workshop) => {
              const isExpanded = expandedWorkshops.has(workshop.id || '');
              const toggleExpand = () => {
                const newSet = new Set(expandedWorkshops);
                if (isExpanded) {
                  newSet.delete(workshop.id || '');
                } else {
                  newSet.add(workshop.id || '');
                }
                setExpandedWorkshops(newSet);
              };
              const createdDate = workshop.meta?.createdAt ? new Date(workshop.meta.createdAt).toLocaleDateString() : '';

              return (
                <Card key={workshop.id} shadow="sm" padding="md" radius="md" withBorder>
                  {/* Workshop header - always visible */}
                  <Group justify="space-between" wrap="nowrap" gap="sm">
                    <Group gap="sm" style={{ flex: 1, minWidth: 0 }}>
                      <ActionIcon variant="subtle" onClick={toggleExpand} size="sm">
                        {isExpanded ? <IconChevronDown size={16} /> : <IconChevronRight size={16} />}
                      </ActionIcon>
                      <Stack gap={2} style={{ minWidth: 0 }}>
                        <Text size="sm" fw={500} truncate>{workshop.name}</Text>
                        <Group gap="xs">
                          <Badge color={workshop.active ? 'green' : 'gray'} variant="light" size="xs">
                            {workshop.active ? t('myOrganization.workshops.active') : t('myOrganization.workshops.inactive')}
                          </Badge>
                          <Group gap={4}>
                            <IconUser size={12} color="gray" />
                            <Text size="xs" c="dimmed">{workshop.participants?.length || 0}</Text>
                          </Group>
                          {createdDate && (
                            <Group gap={4}>
                              <IconCalendar size={12} color="gray" />
                              <Text size="xs" c="dimmed">{createdDate}</Text>
                            </Group>
                          )}
                        </Group>
                      </Stack>
                    </Group>
                    <Group gap="xs" wrap="nowrap">
                      <Tooltip label={workshop.active ? t('myOrganization.workshops.deactivate') : t('myOrganization.workshops.activate')}>
                        <ActionIcon
                          variant="subtle"
                          color={workshop.active ? 'orange' : 'green'}
                          onClick={() => handleToggleActive(workshop)}
                          loading={updateWorkshop.isPending}
                        >
                          {workshop.active ? <IconPlayerPause size={16} /> : <IconPlayerPlay size={16} />}
                        </ActionIcon>
                      </Tooltip>
                      <Tooltip label={t('myOrganization.workshops.inviteParticipants')}>
                        <ActionIcon
                          variant="subtle"
                          color="blue"
                          onClick={() => handleOpenInviteModal(workshop)}
                        >
                          <IconUserPlus size={16} />
                        </ActionIcon>
                      </Tooltip>
                      <Tooltip label={t('delete')}>
                        <ActionIcon
                          variant="subtle"
                          color="red"
                          onClick={() => handleOpenDeleteModal(workshop)}
                        >
                          <IconTrash size={16} />
                        </ActionIcon>
                      </Tooltip>
                    </Group>
                  </Group>

                  {/* Expandable participants section */}
                  <Collapse in={isExpanded}>
                    <Stack gap="sm" mt="md" pt="md" style={{ borderTop: '1px solid var(--mantine-color-gray-3)' }}>
                      <Text size="sm" fw={500}>{t('myOrganization.workshops.participants')}</Text>
                      {workshop.participants && workshop.participants.length > 0 ? (
                        <Stack gap="xs">
                          {workshop.participants.map((participant) => (
                            <Group key={participant.id} gap="xs">
                              <IconUser size={14} />
                              <Text size="sm">{participant.name || t('myOrganization.workshops.anonymousParticipant')}</Text>
                            </Group>
                          ))}
                        </Stack>
                      ) : (
                        <Text size="sm" c="dimmed">{t('myOrganization.workshops.noParticipants')}</Text>
                      )}
                    </Stack>
                  </Collapse>
                </Card>
              );
            })}
          </Stack>
        ) : (
          <Card shadow="sm" padding="xl" radius="md" withBorder>
            <Stack align="center" gap="md">
              <Text c="dimmed" ta="center">
                {t('myOrganization.workshops.empty')}
              </Text>
              <ActionButton
                onClick={openCreateModal}
                leftSection={<IconPlus size={16} />}
              >
                {t('myOrganization.workshops.createFirst')}
              </ActionButton>
            </Stack>
          </Card>
        )}
      </Stack>

      {/* Create Workshop Modal */}
      <Modal
        opened={createModalOpened}
        onClose={closeCreateModal}
        title={t('myOrganization.workshops.createTitle')}
        size="md"
      >
        <Stack gap="md">
          <TextInput
            label={t('myOrganization.workshops.nameLabel')}
            placeholder={t('myOrganization.workshops.namePlaceholder')}
            value={newWorkshopName}
            onChange={(e) => setNewWorkshopName(e.currentTarget.value)}
            required
          />
          <Switch
            label={t('myOrganization.workshops.activeLabel')}
            description={t('myOrganization.workshops.activeDescription')}
            checked={newWorkshopActive}
            onChange={(e) => setNewWorkshopActive(e.currentTarget.checked)}
          />
          <Group justify="flex-end" mt="md">
            <TextButton onClick={closeCreateModal}>
              {t('cancel')}
            </TextButton>
            <ActionButton
              onClick={handleCreateWorkshop}
              loading={createWorkshop.isPending}
              disabled={!newWorkshopName.trim()}
            >
              {t('myOrganization.workshops.create')}
            </ActionButton>
          </Group>
        </Stack>
      </Modal>

      {/* Invite Participants Modal */}
      <Modal
        opened={inviteModalOpened}
        onClose={closeInviteModal}
        title={t('myOrganization.workshops.inviteTitle', { name: selectedWorkshop?.name })}
        size="md"
      >
        <Stack gap="md">
          <Text size="sm" c="dimmed">
            {t('myOrganization.workshops.inviteDescription')}
          </Text>

          {/* Show invite link (existing or newly generated) */}
          {(() => {
            const existingInvite = selectedWorkshopDetails?.invites?.find(inv => inv.status === 'pending' && inv.inviteToken);
            const inviteLink = generatedInviteLink || (existingInvite ? `${window.location.origin}/invites/${existingInvite.inviteToken}/accept` : null);
            const usesCount = existingInvite?.usesCount;

            const handleRevokeInvite = async () => {
              if (!existingInvite?.id) return;
              await revokeInvite.mutateAsync(existingInvite.id);
              setGeneratedInviteLink(null);
            };

            if (inviteLink) {
              return (
                <Stack gap="sm">
                  <Text size="sm" fw={500}>{t('myOrganization.workshops.inviteLink')}</Text>
                  <Group gap="xs">
                    <TextInput
                      value={inviteLink}
                      readOnly
                      style={{ flex: 1 }}
                    />
                    <CopyButton value={inviteLink}>
                      {({ copied, copy }) => (
                        <Tooltip label={copied ? t('copied') : t('copy')}>
                          <ActionIcon color={copied ? 'teal' : 'gray'} onClick={copy}>
                            {copied ? <IconCheck size={16} /> : <IconCopy size={16} />}
                          </ActionIcon>
                        </Tooltip>
                      )}
                    </CopyButton>
                  </Group>
                  {usesCount !== undefined && usesCount > 0 && (
                    <Badge size="sm" variant="light">
                      {t('myOrganization.workshops.usedCount', { count: usesCount })}
                    </Badge>
                  )}
                  <Text size="xs" c="dimmed">
                    {t('myOrganization.workshops.inviteLinkHint')}
                  </Text>
                  {existingInvite && (
                    <DangerButton
                      onClick={handleRevokeInvite}
                      loading={revokeInvite.isPending}
                      leftSection={<IconTrash size={14} />}
                    >
                      {t('myOrganization.workshops.revokeInvite')}
                    </DangerButton>
                  )}
                </Stack>
              );
            }

            return (
              <ActionButton
                onClick={handleGenerateInvite}
                loading={createInvite.isPending}
                leftSection={<IconUserPlus size={16} />}
              >
                {t('myOrganization.workshops.generateInvite')}
              </ActionButton>
            );
          })()}

          <Group justify="flex-end" mt="md">
            <TextButton onClick={closeInviteModal}>
              {t('close')}
            </TextButton>
          </Group>
        </Stack>
      </Modal>

      {/* Delete Confirmation Modal */}
      <ConfirmationModal
        opened={deleteModalOpened}
        onClose={closeDeleteModal}
        title={t('myOrganization.workshops.deleteTitle')}
        message={t('myOrganization.workshops.deleteConfirm', { name: selectedWorkshop?.name })}
        warning={t('myOrganization.workshops.deleteWarning')}
        warningColor="red"
        confirmIcon={<IconTrash size={16} />}
        confirmColor="red"
        onConfirm={handleDeleteWorkshop}
        isLoading={deleteWorkshop.isPending}
      />
    </>
  );
}

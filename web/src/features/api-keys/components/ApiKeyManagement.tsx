import { useState } from 'react';
import {
  Container,
  Stack,
  Group,
  Card,
  Modal,
  TextInput,
  Select,
  Text,
  Alert,
  Box,
  SimpleGrid,
  ActionIcon,
  Skeleton,
  useMantineTheme,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { useTranslation } from 'react-i18next';
import { IconPlus, IconAlertCircle, IconTrash } from '@tabler/icons-react';
import { ActionButton, TextButton, DangerButton } from '@components/buttons';
import { PageTitle } from '@components/typography';
import { useApiKeys, useCreateApiKey, useDeleteApiKey, usePlatforms } from '@/api/hooks';
import type { ObjAiPlatform } from '@/api/generated';


export function ApiKeyManagement() {
  const { t } = useTranslation('common');
  const theme = useMantineTheme();
  const [createModalOpened, { open: openCreateModal, close: closeCreateModal }] = useDisclosure(false);
  
  // Create form state
  const [createName, setCreateName] = useState('');
  const [createPlatform, setCreatePlatform] = useState('openai');
  const [createKey, setCreateKey] = useState('');
  const [createErrors, setCreateErrors] = useState<{ name?: string; platform?: string; key?: string }>({});
  
  const { data: apiKeys, isLoading, error } = useApiKeys();
  const { data: platforms, isLoading: platformsLoading } = usePlatforms();
  const createApiKey = useCreateApiKey();
  const deleteApiKey = useDeleteApiKey();
  
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] = useDisclosure(false);
  const [selectedKey, setSelectedKey] = useState<{ id: string; name: string } | null>(null);

  const validateCreateForm = () => {
    const errors: { name?: string; platform?: string; key?: string } = {};
    if (createName.trim().length === 0) errors.name = t('apiKeys.errors.nameRequired');
    if (createPlatform.length === 0) errors.platform = t('apiKeys.errors.platformRequired');
    if (createKey.trim().length === 0) errors.key = t('apiKeys.errors.keyRequired');
    setCreateErrors(errors);
    return Object.keys(errors).length === 0;
  };


  const handleCreateKey = async () => {
    if (!validateCreateForm()) return;
    try {
      await createApiKey.mutateAsync({
        name: createName,
        platform: createPlatform,
        key: createKey,
      });
      setCreateName('');
      setCreatePlatform('openai');
      setCreateKey('');
      setCreateErrors({});
      closeCreateModal();
    } catch {
      // Error is handled by the mutation
    }
  };


  const openCreateForPlatform = (platform: ObjAiPlatform) => {
    setCreatePlatform(platform.id || 'openai');
    setCreateName('');
    setCreateKey('');
    setCreateErrors({});
    openCreateModal();
  };

  const handleDeleteKey = async () => {
    if (!selectedKey?.id) return;
    try {
      await deleteApiKey.mutateAsync({
        id: selectedKey.id,
        cascade: true,
      });
      closeDeleteModal();
      setSelectedKey(null);
    } catch {
      // Error is handled by the mutation
    }
  };

  const openDelete = (keyId: string, keyName: string) => {
    setSelectedKey({ id: keyId, name: keyName });
    openDeleteModal();
  };

  if (isLoading) {
    return (
      <Container size="lg" py="xl">
        <Stack gap="xl">
          <Skeleton height={40} width="50%" />
          <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
            {[1, 2].map((i) => (
              <Card key={i} shadow="sm" p="lg" radius="md" withBorder>
                <Stack gap="md">
                  <Box>
                    <Skeleton height={28} width="60%" mb="xs" />
                    <Skeleton height={16} width="40%" />
                  </Box>
                  <Skeleton height={20} width="30%" />
                </Stack>
              </Card>
            ))}
          </SimpleGrid>
        </Stack>
      </Container>
    );
  }

  if (error) {
    return (
      <Container size="lg" py="xl">
        <Alert icon={<IconAlertCircle size={16} />} title={t('errors.titles.error')} color="red">
          {t('apiKeys.errors.loadFailed')}
        </Alert>
      </Container>
    );
  }

  return (
    <Container size="lg" py="xl">
      <Stack gap="xl">
        <Box>
          <PageTitle>{t('apiKeys.title')}</PageTitle>
          <Text c="dimmed" mt="xs">
            {t('apiKeys.subtitle')}
          </Text>
        </Box>

        {/* Info Block */}
        <Alert icon={<IconAlertCircle size={18} />} color="cyan" variant="light">
          <Stack gap="xs">
            <Text fw={600} size="sm">{t('apiKeys.aboutSection.title')}</Text>
            <Text size="sm">{t('apiKeys.aboutSection.description')}</Text>
          </Stack>
        </Alert>

        {/* Platform Cards */}
        {platformsLoading ? (
          <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
            {[1, 2].map((i) => (
              <Card key={i} shadow="sm" p="lg" radius="md" withBorder>
                <Stack gap="md">
                  <Box>
                    <Skeleton height={28} width="60%" mb="xs" />
                    <Skeleton height={16} width="40%" />
                  </Box>
                  <Skeleton height={20} width="30%" />
                </Stack>
              </Card>
            ))}
          </SimpleGrid>
        ) : (
          <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
            {platforms?.filter(p => p.id !== 'mock').map((platform) => {
              const platformKeys = apiKeys?.filter(k => k.apiKey?.platform === platform.id) || [];
              return (
                <Card
                  key={platform.id}
                  shadow="sm"
                  p="lg"
                  radius="md"
                  withBorder
                  style={{ 
                    borderTop: platform.supportsApiKey 
                      ? '3px solid var(--mantine-color-accent-5)' 
                      : '3px solid var(--mantine-color-red-5)' 
                  }}
                >
                  <Stack gap="md">
                    <Group justify="space-between" align="flex-start">
                      <Box>
                        <Group gap="xs" align="center">
                          <Text size="lg" fw={700}>{platform.name}</Text>
                          {!platform.supportsApiKey && (
                            <Text size="xs" c="red" fw={600}>
                              ({t('apiKeys.unsupportedPlatform.badge')})
                            </Text>
                          )}
                        </Group>
                        <Text size="sm" c="dimmed">
                          {platformKeys.length} {platformKeys.length === 1 ? t('apiKeys.key') : t('apiKeys.keys')}
                        </Text>
                        {!platform.supportsApiKey && (
                          <Text size="xs" c="red" mt={4}>
                            {t('apiKeys.unsupportedPlatform.hint')}
                          </Text>
                        )}
                      </Box>
                    </Group>
                    
                    {platformKeys.length > 0 && (
                      <Stack 
                        gap={0} 
                        style={{ 
                          backgroundColor: theme.colors.gray[0],
                          borderRadius: theme.radius.sm,
                          border: `1px solid ${theme.colors.gray[2]}`,
                        }}
                      >
                        {platformKeys.map((keyShare, index) => (
                          <Group
                            key={keyShare.id}
                            justify="space-between"
                            align="center"
                            px="sm"
                            py="xs"
                            style={{
                              borderTop: index === 0 ? 'none' : `1px solid ${theme.colors.gray[2]}`,
                            }}
                          >
                            <Box style={{ flex: 1, minWidth: 0 }}>
                              <Text size="sm" fw={600} truncate>
                                {keyShare.apiKey?.name || t('apiKeys.unnamed')}
                              </Text>
                              <Group gap={4} mt={2}>
                                <Text size="xs" c="dimmed">
                                  {t('apiKeys.addedOn')}:
                                </Text>
                                <Text size="xs" c="dimmed">
                                  {keyShare.meta?.createdAt ? new Date(keyShare.meta.createdAt).toLocaleDateString() : '-'}
                                </Text>
                              </Group>
                            </Box>
                            <ActionIcon
                              variant="subtle"
                              color="red"
                              size="sm"
                              onClick={() => openDelete(keyShare.id || '', keyShare.apiKey?.name || t('apiKeys.unnamed'))}
                            >
                              <IconTrash size={14} />
                            </ActionIcon>
                          </Group>
                        ))}
                      </Stack>
                    )}
                    
                    {platform.supportsApiKey && (
                      <TextButton
                        size="xs"
                        leftSection={<IconPlus size={14} />}
                        onClick={() => openCreateForPlatform(platform)}
                      >
                        {t('apiKeys.addKey')}
                      </TextButton>
                    )}
                  </Stack>
                </Card>
              );
            })}
          </SimpleGrid>
        )
        }

      </Stack>

      {/* Create Modal */}
      <Modal
        opened={createModalOpened}
        onClose={closeCreateModal}
        title={t('apiKeys.createModal.title')}
        size="md"
      >
        <Stack gap="md">
          <TextInput
            label={t('apiKeys.createModal.nameLabel')}
            placeholder={t('apiKeys.createModal.namePlaceholder')}
            required
            value={createName}
            onChange={(e) => setCreateName(e.currentTarget.value)}
            error={createErrors.name}
          />
          <Select
            label={t('apiKeys.createModal.platformLabel')}
            placeholder={t('apiKeys.createModal.platformPlaceholder')}
            data={platforms?.map(p => ({ value: p.id || '', label: p.name || '' })) || []}
            required
            value={createPlatform}
            onChange={(value) => setCreatePlatform(value || 'openai')}
            error={createErrors.platform}
            disabled
          />
          <TextInput
            label={t('apiKeys.createModal.keyLabel')}
            placeholder={t('apiKeys.createModal.keyPlaceholder')}
            description={t('apiKeys.createModal.keyDescription')}
            required
            type="password"
            value={createKey}
            onChange={(e) => setCreateKey(e.currentTarget.value)}
            error={createErrors.key}
          />
          <Group justify="flex-end" mt="md">
            <TextButton onClick={closeCreateModal}>{t('cancel')}</TextButton>
            <ActionButton size="md" onClick={handleCreateKey} loading={createApiKey.isPending}>
              {t('apiKeys.createModal.submit')}
            </ActionButton>
          </Group>
        </Stack>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        opened={deleteModalOpened}
        onClose={closeDeleteModal}
        title={t('apiKeys.deleteModal.title')}
        size="sm"
      >
        <Stack gap="md">
          <Text>
            {t('apiKeys.deleteModal.message', { name: selectedKey?.name || t('apiKeys.unnamed') })}
          </Text>
          <Alert icon={<IconAlertCircle size={16} />} color="red" variant="light">
            {t('apiKeys.deleteModal.warning')}
          </Alert>
          <Group justify="flex-end" mt="md">
            <TextButton onClick={closeDeleteModal}>{t('cancel')}</TextButton>
            <DangerButton onClick={handleDeleteKey} loading={deleteApiKey.isPending}>
              {t('delete')}
            </DangerButton>
          </Group>
        </Stack>
      </Modal>
    </Container>
  );
}

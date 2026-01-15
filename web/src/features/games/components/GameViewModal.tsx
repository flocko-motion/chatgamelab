import { 
  Modal, 
  Stack, 
  Group, 
  TextInput, 
  Textarea, 
  Switch, 
  Skeleton, 
  Alert, 
  Text, 
  Badge,
  Tabs,
  ScrollArea,
  Box,
} from '@mantine/core';
import { useMediaQuery } from '@mantine/hooks';
import { 
  IconAlertCircle, 
  IconWorld, 
  IconLock, 
  IconX,
  IconSettings,
  IconMessage,
  IconPhoto,
  IconRefresh,
} from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useState, useEffect } from 'react';
import { ActionButton, TextButton, EditIconButton } from '@components/buttons';
import { InfoCard } from '@components/cards';
import { SectionTitle } from '@components/typography';
import { useGame, useUpdateGame } from '@/api/hooks';

interface GameViewModalProps {
  gameId: string | null;
  opened: boolean;
  onClose: () => void;
  allowEdit?: boolean;
  startInEditMode?: boolean;
}

export function GameViewModal({ gameId, opened, onClose, allowEdit = true, startInEditMode = false }: GameViewModalProps) {
  const { t } = useTranslation('common');
  const isMobile = useMediaQuery('(max-width: 48em)');
  const [isEditing, setIsEditing] = useState(false);
  const [activeTab, setActiveTab] = useState<string | null>('basic');
  
  const { data: game, isLoading, error } = useGame(gameId ?? '');
  const updateGame = useUpdateGame();

  // Basic info
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [isPublic, setIsPublic] = useState(false);
  
  // Story & AI
  const [systemMessageScenario, setSystemMessageScenario] = useState('');
  const [systemMessageGameStart, setSystemMessageGameStart] = useState('');
  const [imageStyle, setImageStyle] = useState('');
  
  // Status fields (JSON string)
  const [statusFields, setStatusFields] = useState('');
  
  // Quick start
  const [firstMessage, setFirstMessage] = useState('');
  const [firstStatus, setFirstStatus] = useState('');

  const handleStartEdit = () => {
    setName(game?.name ?? '');
    setDescription(game?.description ?? '');
    setIsPublic(game?.public ?? false);
    setSystemMessageScenario(game?.systemMessageScenario ?? '');
    setSystemMessageGameStart(game?.systemMessageGameStart ?? '');
    setImageStyle(game?.imageStyle ?? '');
    setStatusFields(game?.statusFields ?? '');
    setFirstMessage(game?.firstMessage ?? '');
    setFirstStatus(game?.firstStatus ?? '');
    setIsEditing(true);
  };

  // Start in edit mode if prop is set and game is loaded
  useEffect(() => {
    if (opened && startInEditMode && game && !isLoading && !isEditing) {
      handleStartEdit();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [opened, startInEditMode, game, isLoading]);

  const handleRegenerate = () => {
    // TODO: Call API to regenerate first message and first status
    // This will be implemented when the backend endpoint is ready
    console.log('Regenerate quick start for game:', gameId);
  };

  const handleSave = async () => {
    if (!game?.id) return;
    
    try {
      await updateGame.mutateAsync({
        id: game.id,
        game: {
          ...game,
          name,
          description,
          public: isPublic,
          systemMessageScenario,
          systemMessageGameStart,
          imageStyle,
          statusFields,
          firstMessage: firstMessage || undefined,
          firstStatus: firstStatus || undefined,
        },
      });
      setIsEditing(false);
    } catch {
      // Error handled by mutation
    }
  };

  const handleCancel = () => {
    handleModalClose();
  };

  const handleModalClose = () => {
    setIsEditing(false);
    setActiveTab('basic');
    onClose();
  };

  const formatDate = (dateString?: string) => {
    if (!dateString) return '-';
    return new Date(dateString).toLocaleDateString();
  };

  const modalContent = () => {
    if (isLoading) {
      return (
        <Stack gap="md">
          <Skeleton height={32} width="60%" />
          <Skeleton height={80} />
          <Skeleton height={32} width="40%" />
        </Stack>
      );
    }

    if (error || !game) {
      return (
        <Alert icon={<IconAlertCircle size={16} />} color="red">
          {t('games.errors.loadFailed')}
        </Alert>
      );
    }

    if (isEditing) {
      return (
        <Stack gap="md">
          <Tabs value={activeTab} onChange={setActiveTab}>
            <Tabs.List>
              <Tabs.Tab value="basic" leftSection={<IconSettings size={14} />}>
                {t('games.tabs.basic')}
              </Tabs.Tab>
              <Tabs.Tab value="story" leftSection={<IconMessage size={14} />}>
                {t('games.tabs.story')}
              </Tabs.Tab>
              <Tabs.Tab value="quickstart" leftSection={<IconPhoto size={14} />}>
                {t('games.tabs.quickStart')}
              </Tabs.Tab>
            </Tabs.List>

            <ScrollArea h={isMobile ? 'calc(100vh - 250px)' : 400} mt="md">
              <Tabs.Panel value="basic">
                <Stack gap="md">
                  <TextInput
                    label={t('games.editFields.name')}
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    required
                  />
                  
                  <Textarea
                    label={t('games.editFields.description')}
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                    minRows={3}
                  />
                  
                  <Switch
                    label={t('games.createModal.publicLabel')}
                    description={t('games.createModal.publicDescription')}
                    checked={isPublic}
                    onChange={(e) => setIsPublic(e.currentTarget.checked)}
                  />

                  <TextInput
                    label={t('games.editFields.imageStyle')}
                    description={t('games.editFields.imageStyleHint')}
                    value={imageStyle}
                    onChange={(e) => setImageStyle(e.target.value)}
                    placeholder="e.g., pixel art, watercolor, realistic..."
                  />
                </Stack>
              </Tabs.Panel>

              <Tabs.Panel value="story">
                <Stack gap="md">
                  <Textarea
                    label={t('games.editFields.scenario')}
                    description={t('games.editFields.scenarioHint')}
                    value={systemMessageScenario}
                    onChange={(e) => setSystemMessageScenario(e.target.value)}
                    minRows={6}
                    autosize
                    maxRows={12}
                  />
                  
                  <Textarea
                    label={t('games.editFields.gameStart')}
                    description={t('games.editFields.gameStartHint')}
                    value={systemMessageGameStart}
                    onChange={(e) => setSystemMessageGameStart(e.target.value)}
                    minRows={4}
                    autosize
                    maxRows={8}
                  />

                  <Textarea
                    label={t('games.editFields.statusFields')}
                    description={t('games.editFields.statusFieldsHint')}
                    value={statusFields}
                    onChange={(e) => setStatusFields(e.target.value)}
                    minRows={3}
                    autosize
                    maxRows={6}
                    styles={{ input: { fontFamily: 'monospace' } }}
                  />
                </Stack>
              </Tabs.Panel>

              <Tabs.Panel value="quickstart">
                <Stack gap="md">
                  <InfoCard>
                    {t('games.editFields.quickStartInfo')}
                  </InfoCard>
                  
                  <Stack gap={4}>
                    <Text size="sm" fw={500}>{t('games.editFields.firstMessage')}</Text>
                    <Box 
                      p="sm" 
                      style={{ 
                        backgroundColor: 'var(--mantine-color-gray-0)',
                        borderRadius: 'var(--mantine-radius-sm)',
                        border: '1px solid var(--mantine-color-gray-2)',
                      }}
                    >
                      <Text size="sm" c="gray.7" style={{ whiteSpace: 'pre-wrap' }}>
                        {game?.firstMessage || t('games.quickStart.noContent')}
                      </Text>
                    </Box>
                  </Stack>
                  
                  <Stack gap={4}>
                    <Text size="sm" fw={500}>{t('games.editFields.firstStatus')}</Text>
                    <Box 
                      p="sm" 
                      style={{ 
                        backgroundColor: 'var(--mantine-color-gray-0)',
                        borderRadius: 'var(--mantine-radius-sm)',
                        border: '1px solid var(--mantine-color-gray-2)',
                        fontFamily: 'monospace',
                      }}
                    >
                      <Text size="sm" c="gray.7" style={{ whiteSpace: 'pre-wrap', fontFamily: 'inherit' }}>
                        {game?.firstStatus || t('games.quickStart.noContent')}
                      </Text>
                    </Box>
                  </Stack>

                  <Group justify="center">
                    <TextButton
                      leftSection={<IconRefresh size={16} />}
                      onClick={handleRegenerate}
                    >
                      {t('games.quickStart.regenerate')}
                    </TextButton>
                  </Group>
                </Stack>
              </Tabs.Panel>
            </ScrollArea>
          </Tabs>

          <Group justify="flex-end" mt="md">
            <TextButton onClick={handleCancel} disabled={updateGame.isPending}>
              {t('cancel')}
            </TextButton>
            <ActionButton onClick={handleSave} loading={updateGame.isPending}>
              {t('save')}
            </ActionButton>
          </Group>
        </Stack>
      );
    }

    // View mode with tabs
    return (
      <Stack gap="md">
        <Group justify="space-between" align="flex-start">
          <Stack gap="xs" style={{ flex: 1 }}>
            <Group gap="sm">
              <SectionTitle>{game.name}</SectionTitle>
              {game.public ? (
                <Badge size="sm" color="green" variant="light" leftSection={<IconWorld size={12} />}>
                  {t('games.visibility.public')}
                </Badge>
              ) : (
                <Badge size="sm" color="gray" variant="light" leftSection={<IconLock size={12} />}>
                  {t('games.visibility.private')}
                </Badge>
              )}
            </Group>
            {game.description && (
              <Text c="gray.6">{game.description}</Text>
            )}
          </Stack>
          {allowEdit && (
            <EditIconButton
              onClick={handleStartEdit}
              aria-label={t('edit')}
            />
          )}
        </Group>

        <Tabs value={activeTab} onChange={setActiveTab}>
          <Tabs.List>
            <Tabs.Tab value="basic" leftSection={<IconSettings size={14} />}>
              {t('games.tabs.basic')}
            </Tabs.Tab>
            <Tabs.Tab value="story" leftSection={<IconMessage size={14} />}>
              {t('games.tabs.story')}
            </Tabs.Tab>
            <Tabs.Tab value="quickstart" leftSection={<IconPhoto size={14} />}>
              {t('games.tabs.quickStart')}
            </Tabs.Tab>
          </Tabs.List>

          <ScrollArea h={isMobile ? 'calc(100vh - 300px)' : 350} mt="md">
            <Tabs.Panel value="basic">
              <Stack gap="md">
                <Group gap="xl">
                  <Stack gap={2}>
                    <Text size="xs" c="gray.5" fw={500}>{t('games.fields.created')}</Text>
                    <Text size="sm">{formatDate(game.meta?.createdAt)}</Text>
                  </Stack>
                  <Stack gap={2}>
                    <Text size="xs" c="gray.5" fw={500}>{t('games.fields.modified')}</Text>
                    <Text size="sm">{formatDate(game.meta?.modifiedAt)}</Text>
                  </Stack>
                </Group>

                {game.imageStyle && (
                  <Stack gap={4}>
                    <Text size="xs" c="gray.5" fw={500}>{t('games.editFields.imageStyle')}</Text>
                    <Text size="sm" c="gray.7">{game.imageStyle}</Text>
                  </Stack>
                )}
              </Stack>
            </Tabs.Panel>

            <Tabs.Panel value="story">
              <Stack gap="md">
                {game.systemMessageScenario ? (
                  <Stack gap={4}>
                    <Text size="xs" c="gray.5" fw={500}>{t('games.editFields.scenario')}</Text>
                    <Box 
                      p="sm" 
                      style={{ 
                        backgroundColor: 'var(--mantine-color-gray-0)',
                        borderRadius: 'var(--mantine-radius-sm)',
                        border: '1px solid var(--mantine-color-gray-2)',
                      }}
                    >
                      <Text size="sm" c="gray.7" style={{ whiteSpace: 'pre-wrap' }}>
                        {game.systemMessageScenario}
                      </Text>
                    </Box>
                  </Stack>
                ) : (
                  <Text size="sm" c="dimmed" fs="italic">{t('games.view.noScenario')}</Text>
                )}

                {game.systemMessageGameStart ? (
                  <Stack gap={4}>
                    <Text size="xs" c="gray.5" fw={500}>{t('games.editFields.gameStart')}</Text>
                    <Box 
                      p="sm" 
                      style={{ 
                        backgroundColor: 'var(--mantine-color-gray-0)',
                        borderRadius: 'var(--mantine-radius-sm)',
                        border: '1px solid var(--mantine-color-gray-2)',
                      }}
                    >
                      <Text size="sm" c="gray.7" style={{ whiteSpace: 'pre-wrap' }}>
                        {game.systemMessageGameStart}
                      </Text>
                    </Box>
                  </Stack>
                ) : (
                  <Text size="sm" c="dimmed" fs="italic">{t('games.view.noGameStart')}</Text>
                )}

                {game.statusFields && (
                  <Stack gap={4}>
                    <Text size="xs" c="gray.5" fw={500}>{t('games.editFields.statusFields')}</Text>
                    <Box 
                      p="sm" 
                      style={{ 
                        backgroundColor: 'var(--mantine-color-gray-0)',
                        borderRadius: 'var(--mantine-radius-sm)',
                        border: '1px solid var(--mantine-color-gray-2)',
                        fontFamily: 'monospace',
                      }}
                    >
                      <Text size="sm" c="gray.7" style={{ whiteSpace: 'pre-wrap', fontFamily: 'inherit' }}>
                        {game.statusFields}
                      </Text>
                    </Box>
                  </Stack>
                )}
              </Stack>
            </Tabs.Panel>

            <Tabs.Panel value="quickstart">
              <Stack gap="md">
                <InfoCard>
                  {t('games.editFields.quickStartInfo')}
                </InfoCard>
                
                <Stack gap={4}>
                  <Text size="xs" c="gray.5" fw={500}>{t('games.editFields.firstMessage')}</Text>
                  <Box 
                    p="sm" 
                    style={{ 
                      backgroundColor: 'var(--mantine-color-gray-0)',
                      borderRadius: 'var(--mantine-radius-sm)',
                      border: '1px solid var(--mantine-color-gray-2)',
                    }}
                  >
                    <Text size="sm" c="gray.7" style={{ whiteSpace: 'pre-wrap' }}>
                      {game.firstMessage || t('games.quickStart.noContent')}
                    </Text>
                  </Box>
                </Stack>
                
                <Stack gap={4}>
                  <Text size="xs" c="gray.5" fw={500}>{t('games.editFields.firstStatus')}</Text>
                  <Box 
                    p="sm" 
                    style={{ 
                      backgroundColor: 'var(--mantine-color-gray-0)',
                      borderRadius: 'var(--mantine-radius-sm)',
                      border: '1px solid var(--mantine-color-gray-2)',
                      fontFamily: 'monospace',
                    }}
                  >
                    <Text size="sm" c="gray.7" style={{ whiteSpace: 'pre-wrap', fontFamily: 'inherit' }}>
                      {game.firstStatus || t('games.quickStart.noContent')}
                    </Text>
                  </Box>
                </Stack>

                <Group justify="center">
                  <TextButton
                    leftSection={<IconRefresh size={16} />}
                    onClick={handleRegenerate}
                  >
                    {t('games.quickStart.regenerate')}
                  </TextButton>
                </Group>
              </Stack>
            </Tabs.Panel>
          </ScrollArea>
        </Tabs>

        <Group justify="flex-end" mt="md">
          <TextButton onClick={handleModalClose} leftSection={<IconX size={16} />}>
            {t('close')}
          </TextButton>
        </Group>
      </Stack>
    );
  };

  return (
    <Modal
      opened={opened}
      onClose={handleModalClose}
      title={isEditing ? t('games.editModal.title') : t('games.viewModal.title')}
      size={isMobile ? '100%' : 'xl'}
      fullScreen={isMobile}
      centered={!isMobile}
    >
      {modalContent()}
    </Modal>
  );
}

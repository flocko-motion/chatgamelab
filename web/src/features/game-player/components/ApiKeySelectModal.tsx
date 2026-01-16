import { useState, useMemo } from 'react';
import {
  Modal,
  Stack,
  Select,
  Group,
  Text,
  Alert,
  Skeleton,
  Box,
} from '@mantine/core';
import { useMediaQuery } from '@mantine/hooks';
import { useTranslation } from 'react-i18next';
import { IconAlertCircle, IconKey } from '@tabler/icons-react';
import { ActionButton, TextButton } from '@components/buttons';
import { SectionTitle } from '@components/typography';
import { useApiKeys, usePlatforms, useCurrentUser, useSystemSettings } from '@/api/hooks';
import { getDefaultApiKey, getModelsForApiKey } from '../types';

interface ApiKeySelectModalProps {
  opened: boolean;
  onClose: () => void;
  onStart: (shareId: string, model?: string) => void;
  gameName?: string;
  isLoading?: boolean;
}

export function ApiKeySelectModal({
  opened,
  onClose,
  onStart,
  gameName,
  isLoading = false,
}: ApiKeySelectModalProps) {
  const { t } = useTranslation('common');
  const isMobile = useMediaQuery('(max-width: 48em)');

  const { data: apiKeys, isLoading: keysLoading, error: keysError } = useApiKeys();
  const { data: platforms, isLoading: platformsLoading } = usePlatforms();
  const { data: currentUser } = useCurrentUser();
  const { data: systemSettings } = useSystemSettings();

  const defaultKey = useMemo(() => getDefaultApiKey(apiKeys || []), [apiKeys]);
  const [selectedKeyId, setSelectedKeyId] = useState<string | null>(null);
  const [selectedModel, setSelectedModel] = useState<string | null>(null);

  const selectedKey = useMemo(() => {
    if (!apiKeys) return undefined;
    if (selectedKeyId) {
      return apiKeys.find(k => k.id === selectedKeyId);
    }
    return defaultKey;
  }, [apiKeys, selectedKeyId, defaultKey]);

  const availableModels = useMemo(() => {
    return getModelsForApiKey(selectedKey, platforms || []);
  }, [selectedKey, platforms]);

  const keyOptions = useMemo(() => {
    if (!apiKeys) return [];
    return apiKeys.map(key => ({
      value: key.id || '',
      label: `${key.apiKey?.name || t('gamePlayer.unnamed')} (${key.apiKey?.platform || 'unknown'})`,
    }));
  }, [apiKeys, t]);

  const modelOptions = useMemo(() => {
    return availableModels.map(model => ({
      value: model.id || '',
      label: model.name || model.id || '',
    }));
  }, [availableModels]);

  const handleStart = () => {
    if (!selectedKey?.id) return;
    // Use selected model if user has enabled selector, otherwise use system default
    const modelToUse = currentUser?.showAiModelSelector 
      ? (selectedModel || undefined)
      : (systemSettings?.defaultAiModel || undefined);
    onStart(selectedKey.id, modelToUse);
  };

  const hasNoKeys = !keysLoading && (!apiKeys || apiKeys.length === 0);

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={
        <Group gap="xs">
          <IconKey size={20} />
          <Text fw={600}>{t('gamePlayer.selectApiKey.title')}</Text>
        </Group>
      }
      size={isMobile ? '100%' : 'md'}
      fullScreen={isMobile}
      centered={!isMobile}
    >
      <Stack gap="lg">
        {gameName && (
          <Box>
            <Text size="sm" c="dimmed">{t('gamePlayer.selectApiKey.playing')}</Text>
            <SectionTitle>{gameName}</SectionTitle>
          </Box>
        )}

        {keysLoading || platformsLoading ? (
          <Stack gap="md">
            <Skeleton height={36} />
            <Skeleton height={36} />
          </Stack>
        ) : keysError ? (
          <Alert icon={<IconAlertCircle size={16} />} color="red">
            {t('gamePlayer.selectApiKey.loadError')}
          </Alert>
        ) : hasNoKeys ? (
          <Alert icon={<IconAlertCircle size={16} />} color="orange">
            {t('gamePlayer.selectApiKey.noKeys')}
          </Alert>
        ) : (
          <>
            <Select
              label={t('gamePlayer.selectApiKey.keyLabel')}
              description={t('gamePlayer.selectApiKey.keyDescription')}
              placeholder={t('gamePlayer.selectApiKey.keyPlaceholder')}
              data={keyOptions}
              value={selectedKeyId || defaultKey?.id || null}
              onChange={setSelectedKeyId}
              searchable
              clearable={false}
            />

            {currentUser?.showAiModelSelector && availableModels.length > 0 && (
              <Select
                label={t('gamePlayer.selectApiKey.modelLabel')}
                description={t('gamePlayer.selectApiKey.modelDescription')}
                placeholder={t('gamePlayer.selectApiKey.modelPlaceholder')}
                data={modelOptions}
                value={selectedModel}
                onChange={setSelectedModel}
                searchable
                clearable
              />
            )}
          </>
        )}

        <Group justify="flex-end" mt="md">
          <TextButton onClick={onClose} disabled={isLoading}>
            {t('cancel')}
          </TextButton>
          <ActionButton
            onClick={handleStart}
            loading={isLoading}
            disabled={hasNoKeys || !selectedKey}
          >
            {t('gamePlayer.selectApiKey.startGame')}
          </ActionButton>
        </Group>
      </Stack>
    </Modal>
  );
}

import { useState, useMemo, useEffect } from 'react';
import {
  Modal,
  Stack,
  Select,
  Group,
  Text,
  Alert,
  Skeleton,
  Box,
  Badge,
} from '@mantine/core';
import type { ComboboxItem, ComboboxLikeRenderOptionInput } from '@mantine/core';
import { useMediaQuery } from '@mantine/hooks';
import { useTranslation } from 'react-i18next';
import { IconAlertCircle, IconKey } from '@tabler/icons-react';
import { ActionButton, TextButton } from '@components/buttons';
import { SectionTitle } from '@components/typography';
import { useApiKeys, usePlatforms, useCurrentUser, useSystemSettings, useAvailableKeysForGame } from '@/api/hooks';
import { getDefaultApiKey, getModelsForApiKey } from '../types';
import type { ObjAvailableKey } from '@/api/generated';

const STORAGE_KEY = 'chatgamelab-api-key-selection';

function getStoredKeyForPlatform(platform: string): string | null {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (!stored) return null;
    const selections = JSON.parse(stored) as Record<string, string>;
    return selections[platform] || null;
  } catch {
    return null;
  }
}

function saveKeyForPlatform(platform: string, keyId: string): void {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    const selections = stored ? JSON.parse(stored) as Record<string, string> : {};
    selections[platform] = keyId;
    localStorage.setItem(STORAGE_KEY, JSON.stringify(selections));
  } catch {
    // Ignore storage errors
  }
}

interface ApiKeySelectModalProps {
  opened: boolean;
  onClose: () => void;
  onStart: (shareId: string, model?: string) => void;
  gameId?: string;
  gameName?: string;
  isLoading?: boolean;
}

export function ApiKeySelectModal({
  opened,
  onClose,
  onStart,
  gameId,
  gameName,
  isLoading = false,
}: ApiKeySelectModalProps) {
  const { t } = useTranslation('common');
  const isMobile = useMediaQuery('(max-width: 48em)');

  // Use available keys for the specific game (includes sponsor, institution, personal keys)
  const { data: availableKeys, isLoading: availableKeysLoading, error: availableKeysError } = useAvailableKeysForGame(gameId);
  // Fallback to user's own API keys if no gameId provided
  const { data: apiKeys, isLoading: apiKeysLoading, error: apiKeysError } = useApiKeys();
  const { data: platforms, isLoading: platformsLoading } = usePlatforms();
  const { data: currentUser } = useCurrentUser();
  const { data: systemSettings } = useSystemSettings();

  // Determine which keys to show - prefer available keys for the game
  const keysLoading = gameId ? availableKeysLoading : apiKeysLoading;
  const keysError = gameId ? availableKeysError : apiKeysError;

  const defaultKey = useMemo(() => getDefaultApiKey(apiKeys || []), [apiKeys]);
  const [selectedKeyId, setSelectedKeyId] = useState<string | null>(null);
  const [selectedModel, setSelectedModel] = useState<string | null>(null);
  const [initializedFromStorage, setInitializedFromStorage] = useState(false);

  // Build key options from available keys (game-specific) or user keys (fallback)
  // Deduplicate keys that appear in multiple sources and track source info
  const keyOptions = useMemo(() => {
    if (gameId && availableKeys && availableKeys.length > 0) {
      // Group keys by name+platform to detect duplicates
      const keyMap = new Map<string, ObjAvailableKey[]>();
      for (const key of availableKeys) {
        const keyId = `${key.name}-${key.platform}`;
        const existing = keyMap.get(keyId) || [];
        existing.push(key);
        keyMap.set(keyId, existing);
      }

      // Build options, preferring personal keys but showing all sources
      const options: { value: string; label: string; source: string; isDefault: boolean; hasSponsor: boolean; hasPersonal: boolean; hasInstitution: boolean; hasPublic: boolean }[] = [];
      const seenKeys = new Set<string>();

      for (const key of availableKeys) {
        const keyId = `${key.name}-${key.platform}`;
        if (seenKeys.has(keyId)) continue;
        seenKeys.add(keyId);

        const allSources = keyMap.get(keyId) || [key];
        
        // Determine badges based on sources
        const hasPersonal = allSources.some(k => k.source === 'personal');
        const hasInstitution = allSources.some(k => k.source === 'institution');
        const hasSponsor = allSources.some(k => k.source === 'sponsor');
        const hasPublic = allSources.some(k => k.source === 'public');
        
        // Prefer personal share ID, then institution, then sponsor
        const preferredKey = allSources.find(k => k.source === 'personal') 
          || allSources.find(k => k.source === 'institution')
          || key;

        options.push({
          value: preferredKey.shareId || '',
          label: `${key.name || t('gamePlayer.unnamed')} (${key.platform || 'unknown'})`,
          source: preferredKey.source || 'unknown',
          isDefault: allSources.some(k => k.isDefault),
          hasSponsor,
          hasPersonal,
          hasInstitution,
          hasPublic,
        });
      }

      return options;
    }
    if (!apiKeys) return [];
    return apiKeys.map(key => ({
      value: key.id || '',
      label: `${key.apiKey?.name || t('gamePlayer.unnamed')} (${key.apiKey?.platform || 'unknown'})`,
      source: 'personal',
      isDefault: key.isUserDefault,
      hasSponsor: false,
      hasPersonal: true,
      hasInstitution: false,
      hasPublic: false,
    }));
  }, [gameId, availableKeys, apiKeys, t]);

  // Render badges for a key option
  const renderKeyBadges = (keyOption: typeof keyOptions[0], size: 'xs' | 'sm' = 'xs') => (
    <Group gap={4} wrap="nowrap">
      {keyOption.hasSponsor && (
        <Badge size={size} variant="filled" color="yellow">⭐</Badge>
      )}
      {keyOption.hasPersonal && (
        <Badge size={size} variant="filled" color="violet">{t('gamePlayer.selectApiKey.mine')}</Badge>
      )}
      {keyOption.hasInstitution && (
        <Badge size={size} variant="light" color="cyan">🏛️</Badge>
      )}
      {keyOption.hasPublic && (
        <Badge size={size} variant="light" color="green">�</Badge>
      )}
    </Group>
  );

  // Custom render function for Select options with badges
  const renderSelectOption = ({ option }: ComboboxLikeRenderOptionInput<ComboboxItem>) => {
    const keyOption = keyOptions.find(k => k.value === option.value);
    if (!keyOption) return option.label;
    
    return (
      <Group gap="xs" wrap="nowrap" justify="space-between" style={{ width: '100%' }}>
        <Text size="sm" truncate>{option.label}</Text>
        {renderKeyBadges(keyOption)}
      </Group>
    );
  };

  // Initialize selection - prefer sponsor key, then default, then first available
  useEffect(() => {
    if (keyOptions.length === 0 || initializedFromStorage) return;
    
    // First priority: sponsor key
    const sponsorKey = keyOptions.find(k => k.source === 'sponsor');
    if (sponsorKey) {
      setSelectedKeyId(sponsorKey.value);
      setInitializedFromStorage(true);
      return;
    }

    // Second priority: default key (institution or user default)
    const defaultOpt = keyOptions.find(k => k.isDefault);
    if (defaultOpt) {
      setSelectedKeyId(defaultOpt.value);
      setInitializedFromStorage(true);
      return;
    }

    // Third: check localStorage for previously used key
    for (const key of keyOptions) {
      const platform = key.label.match(/\(([^)]+)\)/)?.[1];
      if (!platform) continue;
      
      const storedKeyId = getStoredKeyForPlatform(platform);
      if (storedKeyId && keyOptions.some(k => k.value === storedKeyId)) {
        setSelectedKeyId(storedKeyId);
        setInitializedFromStorage(true);
        return;
      }
    }

    // Fallback: first key
    if (keyOptions.length > 0) {
      setSelectedKeyId(keyOptions[0].value);
    }
    setInitializedFromStorage(true);
  }, [keyOptions, initializedFromStorage]);

  const selectedKey = useMemo(() => {
    if (!apiKeys) return undefined;
    if (selectedKeyId) {
      return apiKeys.find(k => k.id === selectedKeyId);
    }
    return defaultKey;
  }, [apiKeys, selectedKeyId, defaultKey]);

  // For available keys (institution/sponsor), we need to get the platform from the selected option
  const selectedKeyPlatform = useMemo(() => {
    if (gameId && availableKeys && selectedKeyId) {
      const availableKey = availableKeys.find(k => k.shareId === selectedKeyId);
      return availableKey?.platform;
    }
    return selectedKey?.apiKey?.platform;
  }, [gameId, availableKeys, selectedKeyId, selectedKey]);

  const availableModels = useMemo(() => {
    // If using available keys (game-specific), get models from platform
    if (gameId && selectedKeyPlatform && platforms) {
      const platform = platforms.find(p => p.id === selectedKeyPlatform);
      return platform?.models || [];
    }
    // Otherwise use personal key
    return getModelsForApiKey(selectedKey, platforms || []);
  }, [gameId, selectedKeyPlatform, selectedKey, platforms]);

  const modelOptions = useMemo(() => {
    return availableModels.map(model => ({
      value: model.id || '',
      label: model.name || model.id || '',
    }));
  }, [availableModels]);

  const handleStart = () => {
    if (!selectedKeyId) return;
    
    // Save selection to localStorage for this platform
    const selectedOption = keyOptions.find(k => k.value === selectedKeyId);
    const platform = selectedOption?.label.match(/\(([^)]+)\)/)?.[1];
    if (platform) {
      saveKeyForPlatform(platform, selectedKeyId);
    }
    
    const modelToUse = currentUser?.showAiModelSelector 
      ? (selectedModel || undefined)
      : (systemSettings?.defaultAiModel || undefined);
    onStart(selectedKeyId, modelToUse);
  };

  const hasNoKeys = !keysLoading && keyOptions.length === 0;

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
            <Box>
              <Select
                label={t('gamePlayer.selectApiKey.keyLabel')}
                description={t('gamePlayer.selectApiKey.keyDescription')}
                placeholder={t('gamePlayer.selectApiKey.keyPlaceholder')}
                data={keyOptions.map(k => ({ value: k.value, label: k.label }))}
                value={selectedKeyId}
                onChange={setSelectedKeyId}
                renderOption={renderSelectOption}
                searchable
                clearable={false}
              />
              {selectedKeyId && (() => {
                const selected = keyOptions.find(k => k.value === selectedKeyId);
                return selected ? (
                  <Group gap="xs" mt="xs">
                    {renderKeyBadges(selected, 'sm')}
                  </Group>
                ) : null;
              })()}
            </Box>

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
            disabled={hasNoKeys || !selectedKeyId}
          >
            {t('gamePlayer.selectApiKey.startGame')}
          </ActionButton>
        </Group>
      </Stack>
    </Modal>
  );
}

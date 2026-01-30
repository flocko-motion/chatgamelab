import { 
  Modal, 
  Stack, 
  Group, 
  TextInput, 
  Textarea, 
  Switch, 
  Skeleton, 
  Alert, 
  ScrollArea,
  rem,
  Text,
} from '@mantine/core';
import { IconCopy } from '@tabler/icons-react';
import { useMediaQuery } from '@mantine/hooks';
import { useModals } from '@mantine/modals';
import { IconAlertCircle } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useState, useEffect, useRef, useCallback } from 'react';
import { ActionButton, CancelButton } from '@components/buttons';
import { useGame, useUpdateGame } from '@/api/hooks';
import { StatusFieldsEditor } from './StatusFieldsEditor';
import type { CreateGameFormData } from '../types';

interface FormValues {
  name: string;
  description: string;
  isPublic: boolean;
  systemMessageScenario: string;
  systemMessageGameStart: string;
  imageStyle: string;
  statusFields: string;
}

interface GameEditModalProps {
  gameId?: string | null;
  opened: boolean;
  onClose: () => void;
  onCreate?: (data: CreateGameFormData) => void;
  createLoading?: boolean;
  /** If true, all fields are read-only (view mode) */
  readOnly?: boolean;
  /** Callback when user clicks Copy button in view mode */
  onCopy?: () => void;
  /** Loading state for copy operation */
  copyLoading?: boolean;
}

export function GameEditModal({ 
  gameId, 
  opened, 
  onClose, 
  onCreate,
  createLoading = false,
  readOnly = false,
  onCopy,
  copyLoading = false,
}: GameEditModalProps) {
  const { t } = useTranslation('common');
  const isMobile = useMediaQuery('(max-width: 48em)');
    const modals = useModals();
  
  const isCreateMode = !gameId;
  const { data: game, isLoading, error } = useGame(gameId ?? '');
  const updateGame = useUpdateGame();

  // Form fields
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [isPublic, setIsPublic] = useState(false);
  const [systemMessageScenario, setSystemMessageScenario] = useState('');
  const [systemMessageGameStart, setSystemMessageGameStart] = useState('');
  const [imageStyle, setImageStyle] = useState('');
  const [statusFields, setStatusFields] = useState('');
  
  // Validation
  const [nameError, setNameError] = useState('');

  // Track initial values for dirty checking
  const initialValues = useRef<FormValues | null>(null);

  // Check if form has unsaved changes
  const isDirty = useCallback(() => {
    if (!initialValues.current) return false;
    return (
      name !== initialValues.current.name ||
      description !== initialValues.current.description ||
      isPublic !== initialValues.current.isPublic ||
      systemMessageScenario !== initialValues.current.systemMessageScenario ||
      systemMessageGameStart !== initialValues.current.systemMessageGameStart ||
      imageStyle !== initialValues.current.imageStyle ||
      statusFields !== initialValues.current.statusFields
    );
  }, [name, description, isPublic, systemMessageScenario, systemMessageGameStart, imageStyle, statusFields]);

  // Track if we've initialized form values
  const hasInitialized = useRef(false);
  
  // Initialize form values when game loads (only once)
  /* eslint-disable react-hooks/set-state-in-effect -- Intentional: initialize form from game data */
  useEffect(() => {
    if (!isCreateMode && game && !isLoading && !hasInitialized.current) {
      hasInitialized.current = true;
      const values: FormValues = {
        name: game.name ?? '',
        description: game.description ?? '',
        isPublic: game.public ?? false,
        systemMessageScenario: game.systemMessageScenario ?? '',
        systemMessageGameStart: game.systemMessageGameStart ?? '',
        imageStyle: game.imageStyle ?? '',
        statusFields: game.statusFields ?? '',
      };
      initialValues.current = values;
      setName(values.name);
      setDescription(values.description);
      setIsPublic(values.isPublic);
      setSystemMessageScenario(values.systemMessageScenario);
      setSystemMessageGameStart(values.systemMessageGameStart);
      setImageStyle(values.imageStyle);
      setStatusFields(values.statusFields);
    }
    // Initialize for create mode
    if (isCreateMode && opened && !hasInitialized.current) {
      hasInitialized.current = true;
      initialValues.current = {
        name: '',
        description: '',
        isPublic: false,
        systemMessageScenario: '',
        systemMessageGameStart: '',
        imageStyle: '',
        statusFields: '',
      };
    }
    // Reset when modal closes
    if (!opened) {
      hasInitialized.current = false;
      initialValues.current = null;
      if (isCreateMode) {
        // Reset form for create mode
        setName('');
        setDescription('');
        setIsPublic(false);
        setSystemMessageScenario('');
        setSystemMessageGameStart('');
        setImageStyle('');
        setStatusFields('');
        setNameError('');
      }
    }
  }, [isCreateMode, game, isLoading, opened]);

  const handleSave = async () => {
    if (!name.trim()) {
      setNameError(t('games.errors.nameRequired'));
      return;
    }

    if (isCreateMode) {
      // Create mode - use onCreate callback
      onCreate?.({
        name: name.trim(),
        description: description.trim(),
        isPublic,
        systemMessageScenario: systemMessageScenario.trim() || undefined,
        systemMessageGameStart: systemMessageGameStart.trim() || undefined,
        imageStyle: imageStyle.trim() || undefined,
        statusFields: statusFields.trim() || undefined,
      });
    } else if (game?.id) {
      // Edit mode - update existing game
      try {
        await updateGame.mutateAsync({
          id: game.id,
          game: {
            ...game,
            name: name.trim(),
            description: description.trim(),
            public: isPublic,
            systemMessageScenario: systemMessageScenario.trim() || undefined,
            systemMessageGameStart: systemMessageGameStart.trim() || undefined,
            imageStyle: imageStyle.trim() || undefined,
            statusFields: statusFields.trim() || undefined,
          },
        });
        onClose();
      } catch {
        // Error handled by mutation
      }
    }
  };

  
  const handleModalClose = () => {
    if (isDirty()) {
      modals.openConfirmModal({
        title: t('games.editModal.unsavedChanges.title'),
        children: (
          <Text size="sm">
            {t('games.editModal.unsavedChanges.message')}
          </Text>
        ),
        labels: {
          confirm: t('games.editModal.unsavedChanges.discard'),
          cancel: t('games.editModal.unsavedChanges.keepEditing'),
        },
        confirmProps: { color: 'red' },
        onConfirm: onClose,
      });
    } else {
      onClose();
    }
  };

  const isLoaderActive = !isCreateMode && isLoading;
  const isSaving = isCreateMode ? createLoading : updateGame.isPending;
  
  const modalContent = () => {
    if (isLoaderActive) {
      return (
        <Stack gap="md">
          <Skeleton height={32} width="60%" />
          <Skeleton height={80} />
          <Skeleton height={32} width="40%" />
        </Stack>
      );
    }

    if (!isCreateMode && (error || !game)) {
      return (
        <Alert icon={<IconAlertCircle size={16} />} color="red">
          {t('games.errors.loadFailed')}
        </Alert>
      );
    }

    return (
      <ScrollArea h={isMobile ? 'calc(100vh - 180px)' : 'calc(85vh - 140px)'}>
        <Stack gap="lg" pr="md">
          {/* Read-only notice */}
          {readOnly && (
            <Alert icon={<IconAlertCircle size={16} />} color="blue" variant="light">
              {t('games.viewModal.readOnlyNotice')}
            </Alert>
          )}
          
          {/* Name */}
          <TextInput
            label={t('games.editFields.name')}
            placeholder={t('games.createModal.namePlaceholder')}
            value={name}
            onChange={(e) => {
              setName(e.target.value);
              if (nameError) setNameError('');
            }}
            error={nameError}
            required
            readOnly={readOnly}
            data-autofocus
          />
          
          {/* Scenario */}
          <Textarea
            label={t('games.editFields.scenario')}
            description={t('games.editFields.scenarioHint')}
            placeholder={t('games.editFields.scenarioPlaceholder')}
            value={systemMessageScenario}
            onChange={(e) => setSystemMessageScenario(e.target.value)}
            minRows={6}
            autosize
            maxRows={12}
            required
            readOnly={readOnly}
          />
          
          {/* Game Start */}
          <Textarea
            label={t('games.editFields.gameStart')}
            description={t('games.editFields.gameStartHint')}
            placeholder={t('games.editFields.gameStartPlaceholder')}
            value={systemMessageGameStart}
            onChange={(e) => setSystemMessageGameStart(e.target.value)}
            minRows={4}
            autosize
            maxRows={8}
            required
            readOnly={readOnly}
          />

          {/* Image Style */}
          <TextInput
            label={t('games.editFields.imageStyle')}
            description={t('games.editFields.imageStyleHint')}
            value={imageStyle}
            onChange={(e) => setImageStyle(e.target.value)}
            placeholder="e.g., pixel art, watercolor, realistic..."
            required
            readOnly={readOnly}
          />

          {/* Status Fields */}
          <StatusFieldsEditor
            value={statusFields}
            onChange={setStatusFields}
            readOnly={readOnly}
          />

          {/* Description - moved to end */}
          <Textarea
            label={t('games.editFields.description')}
            placeholder={t('games.createModal.descriptionPlaceholder')}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            minRows={3}
            readOnly={readOnly}
          />
          
          {/* Visibility - last */}
          <Switch
            label={t('games.createModal.publicLabel')}
            description={t('games.createModal.publicDescription')}
            checked={isPublic}
            onChange={(e) => setIsPublic(e.currentTarget.checked)}
            disabled={readOnly}
          />
        </Stack>
      </ScrollArea>
    );
  };

  return (
    <Modal
      opened={opened}
      onClose={handleModalClose}
      title={isCreateMode ? t('games.createModal.title') : readOnly ? t('games.viewModal.title') : t('games.editModal.title')}
      size={isMobile ? '100%' : rem(1000)}
      fullScreen={isMobile}
      centered={!isMobile}
      styles={{
        content: { maxHeight: isMobile ? undefined : '85vh' },
        body: { maxHeight: isMobile ? undefined : 'calc(85vh - 60px)' },
      }}
    >
      <Stack gap="md">
        {modalContent()}

        <Group justify="flex-end" mt="md" gap="sm">
          {readOnly ? (
            <>
              <CancelButton onClick={onClose}>
                {t('close')}
              </CancelButton>
              {onCopy && (
                <ActionButton 
                  onClick={onCopy} 
                  loading={copyLoading} 
                  size="sm"
                  leftSection={<IconCopy size={16} />}
                >
                  {t('games.copyGame')}
                </ActionButton>
              )}
            </>
          ) : (
            <>
              <CancelButton onClick={handleModalClose} disabled={isSaving}>
                {t('cancel')}
              </CancelButton>
              <ActionButton onClick={handleSave} loading={isSaving} size="sm">
                {isCreateMode ? t('games.createModal.submit') : t('save')}
              </ActionButton>
            </>
          )}
        </Group>
      </Stack>
    </Modal>
  );
}

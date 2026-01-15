import { useState } from 'react';
import { Modal, TextInput, Textarea, Switch, Stack, Group, Button } from '@mantine/core';
import { useTranslation } from 'react-i18next';
import { ActionButton } from '@components/buttons';
import type { CreateGameFormData } from '../types';

interface CreateGameModalProps {
  opened: boolean;
  onClose: () => void;
  onSubmit: (data: CreateGameFormData) => void;
  loading?: boolean;
}

export function CreateGameModal({ opened, onClose, onSubmit, loading }: CreateGameModalProps) {
  const { t } = useTranslation('common');
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [isPublic, setIsPublic] = useState(false);
  const [nameError, setNameError] = useState('');

  const handleSubmit = () => {
    if (!name.trim()) {
      setNameError(t('games.errors.nameRequired'));
      return;
    }
    
    onSubmit({
      name: name.trim(),
      description: description.trim(),
      isPublic,
    });
  };

  const handleClose = () => {
    setName('');
    setDescription('');
    setIsPublic(false);
    setNameError('');
    onClose();
  };

  return (
    <Modal
      opened={opened}
      onClose={handleClose}
      title={t('games.createModal.title')}
      size="md"
    >
      <Stack gap="md">
        <TextInput
          label={t('games.createModal.nameLabel')}
          placeholder={t('games.createModal.namePlaceholder')}
          value={name}
          onChange={(e) => {
            setName(e.target.value);
            if (nameError) setNameError('');
          }}
          error={nameError}
          required
          data-autofocus
        />
        
        <Textarea
          label={t('games.createModal.descriptionLabel')}
          placeholder={t('games.createModal.descriptionPlaceholder')}
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

        <Group justify="flex-end" mt="md">
          <Button 
            variant="subtle" 
            color="gray" 
            onClick={handleClose} 
            disabled={loading}
            size="md"
          >
            {t('cancel')}
          </Button>
          <ActionButton onClick={handleSubmit} loading={loading}>
            {t('games.createModal.submit')}
          </ActionButton>
        </Group>
      </Stack>
    </Modal>
  );
}

import { useState } from 'react';
import { Modal, Stack, Text, TextInput, Group, ActionIcon, Alert } from '@mantine/core';
import { IconSend, IconAlertCircle } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';

export interface InviteModalProps {
  opened: boolean;
  onClose: () => void;
  title: string;
  description: string;
  onSubmit: (email: string) => void;
  isLoading?: boolean;
  error?: string | null;
}

export function InviteModal({
  opened,
  onClose,
  title,
  description,
  onSubmit,
  isLoading = false,
  error = null,
}: InviteModalProps) {
  const { t } = useTranslation('common');
  const [email, setEmail] = useState('');

  const handleSubmit = () => {
    if (email.trim()) {
      onSubmit(email);
      setEmail('');
    }
  };

  const handleClose = () => {
    setEmail('');
    onClose();
  };

  return (
    <Modal opened={opened} onClose={handleClose} title={title}>
      <Stack gap="md">
        <Text size="sm" c="dimmed">
          {description}
        </Text>
        <TextInput
          label={t('myOrganization.email')}
          placeholder={t('myOrganization.emailPlaceholder')}
          type="email"
          value={email}
          onChange={(e) => setEmail(e.currentTarget.value)}
        />
        {error && (
          <Alert color="red" icon={<IconAlertCircle size={16} />}>
            {error}
          </Alert>
        )}
        <Group justify="flex-end">
          <Text
            size="sm"
            c="dimmed"
            style={{ cursor: 'pointer' }}
            onClick={handleClose}
          >
            {t('cancel')}
          </Text>
          <ActionIcon
            color="blue"
            variant="filled"
            size="lg"
            onClick={handleSubmit}
            loading={isLoading}
            disabled={!email.trim()}
          >
            <IconSend size={16} />
          </ActionIcon>
        </Group>
      </Stack>
    </Modal>
  );
}

import { Modal, Text, Stack, Group } from '@mantine/core';
import { useTranslation } from 'react-i18next';
import { TextButton, DangerButton } from '@components/buttons';

interface DeleteGameModalProps {
  opened: boolean;
  onClose: () => void;
  onConfirm: () => void;
  gameName: string;
  loading?: boolean;
}

export function DeleteGameModal({ opened, onClose, onConfirm, gameName, loading }: DeleteGameModalProps) {
  const { t } = useTranslation('common');

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={t('games.deleteModal.title')}
      size="sm"
    >
      <Stack gap="md">
        <Text>
          {t('games.deleteModal.message', { name: gameName })}
        </Text>
        <Text size="sm" c="red.6">
          {t('games.deleteModal.warning')}
        </Text>
        
        <Group justify="flex-end" mt="md">
          <TextButton onClick={onClose} disabled={loading}>
            {t('cancel')}
          </TextButton>
          <DangerButton onClick={onConfirm} loading={loading}>
            {t('delete')}
          </DangerButton>
        </Group>
      </Stack>
    </Modal>
  );
}

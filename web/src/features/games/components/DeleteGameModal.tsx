import { Modal, Text, Stack, Group, Alert } from '@mantine/core';
import { IconAlertTriangle } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { TextButton, DangerButton } from '@components/buttons';
import { useAdmin } from '@/common/hooks/useAdmin';

interface DeleteGameModalProps {
  opened: boolean;
  onClose: () => void;
  onConfirm: () => void;
  gameName: string;
  loading?: boolean;
  isOwner?: boolean;
}

export function DeleteGameModal({ opened, onClose, onConfirm, gameName, loading, isOwner = true }: DeleteGameModalProps) {
  const { t } = useTranslation('common');
  const { isAdmin } = useAdmin();
  const isAdminAction = isAdmin && !isOwner;

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={t('games.deleteModal.title')}
      size="sm"
    >
      <Stack gap="md">
        {isAdminAction && (
          <Alert
            color="orange"
            icon={<IconAlertTriangle size={16} />}
            title={t('games.deleteModal.adminActionTitle')}
          >
            {t('games.deleteModal.adminActionWarning')}
          </Alert>
        )}
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

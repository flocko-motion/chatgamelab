import type { ReactNode } from "react";
import { Modal, Stack, Text, Group, ActionIcon, Alert } from "@mantine/core";
import { IconAlertCircle } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";

export interface ConfirmationModalProps {
  opened: boolean;
  onClose: () => void;
  title: string;
  message: string;
  warning?: string;
  warningColor?: string;
  confirmIcon: ReactNode;
  confirmColor: string;
  onConfirm: () => void;
  isLoading?: boolean;
  /** Error message to display (red alert) */
  error?: string | null;
}

export function ConfirmationModal({
  opened,
  onClose,
  title,
  message,
  warning,
  warningColor = "yellow",
  confirmIcon,
  confirmColor,
  onConfirm,
  isLoading = false,
  error,
}: ConfirmationModalProps) {
  const { t } = useTranslation("common");

  return (
    <Modal opened={opened} onClose={onClose} title={title}>
      <Stack gap="md">
        <Text>{message}</Text>
        {warning && (
          <Alert color={warningColor} icon={<IconAlertCircle size={16} />}>
            {warning}
          </Alert>
        )}
        {error && (
          <Alert color="red" icon={<IconAlertCircle size={16} />}>
            {error}
          </Alert>
        )}
        <Group justify="flex-end">
          <Text
            size="sm"
            c="dimmed"
            style={{ cursor: "pointer" }}
            onClick={onClose}
          >
            {t("cancel")}
          </Text>
          <ActionIcon
            color={confirmColor}
            variant="filled"
            size="lg"
            onClick={onConfirm}
            loading={isLoading}
          >
            {confirmIcon}
          </ActionIcon>
        </Group>
      </Stack>
    </Modal>
  );
}

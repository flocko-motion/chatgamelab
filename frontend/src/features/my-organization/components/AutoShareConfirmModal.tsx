import { IconShare } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { ConfirmationModal } from "./ConfirmationModal";

interface AutoShareConfirmModalProps {
  opened: boolean;
  onClose: () => void;
  onConfirm: () => void;
  keyName: string;
  orgName: string;
  isLoading?: boolean;
  error?: string | null;
}

export function AutoShareConfirmModal({
  opened,
  onClose,
  onConfirm,
  keyName,
  orgName,
  isLoading = false,
  error,
}: AutoShareConfirmModalProps) {
  const { t } = useTranslation("common");

  return (
    <ConfirmationModal
      opened={opened}
      onClose={onClose}
      title={t("myOrganization.autoShare.confirmTitle")}
      message={t("myOrganization.autoShare.confirmMessage", { orgName })}
      warning={keyName}
      warningColor="blue"
      confirmIcon={<IconShare size={18} />}
      confirmColor="violet"
      onConfirm={onConfirm}
      isLoading={isLoading}
      error={error}
    />
  );
}

import { useState } from "react";
import { Modal, Stack, Text, Checkbox, Button, Alert } from "@mantine/core";
import { IconInfoCircle } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { dismissInfoModal } from "../lib/infoModal";

interface WorkshopModeInfoModalProps {
  opened: boolean;
  onClose: () => void;
  workshopName: string;
  role?: string;
}

export function WorkshopModeInfoModal({
  opened,
  onClose,
  workshopName,
  role,
}: WorkshopModeInfoModalProps) {
  const { t } = useTranslation("myWorkshop");
  const [dontShowAgain, setDontShowAgain] = useState(false);

  const handleClose = () => {
    if (dontShowAgain) dismissInfoModal();
    onClose();
  };

  const isHeadOrStaff = role === "head" || role === "staff";
  const isIndividual = role === "individual";

  return (
    <Modal
      opened={opened}
      onClose={handleClose}
      title={t("infoModal.title", { name: workshopName })}
      centered
      size="md"
    >
      <Stack gap="md">
        <Text size="sm">{t("infoModal.body")}</Text>

        {isHeadOrStaff && (
          <Alert
            variant="light"
            color="blue"
            icon={<IconInfoCircle size={16} />}
          >
            {t("infoModal.headStaffNote")}
          </Alert>
        )}

        {isIndividual && (
          <Alert
            variant="light"
            color="orange"
            icon={<IconInfoCircle size={16} />}
          >
            {t("infoModal.individualNote")}
          </Alert>
        )}

        <Checkbox
          label={t("infoModal.dontShowAgain")}
          checked={dontShowAgain}
          onChange={(e) => setDontShowAgain(e.currentTarget.checked)}
        />

        <Button onClick={handleClose} fullWidth>
          {t("infoModal.gotIt")}
        </Button>
      </Stack>
    </Modal>
  );
}

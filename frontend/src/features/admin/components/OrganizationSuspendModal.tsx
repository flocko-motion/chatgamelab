import { useState } from "react";
import { Alert, Group, Modal, Stack, TextInput } from "@mantine/core";
import { IconAlertCircle } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { ActionButton, CancelButton, DangerButton } from "@components/buttons";
import { BodyText } from "@components/typography";
import type { OrganisationView } from "@/api/generated/api.gen";
import { useSetOrganisationSuspension } from "@/api/hooks";

interface OrganizationSuspendModalProps {
  /** The organisation to suspend or lift the suspension of; null: closed. */
  org: OrganisationView | null;
  onClose: () => void;
}

/**
 * Suspends an organisation, with an optional reason only admins see, or
 * lifts an admin's suspension. Suspending keeps every piece of its data.
 */
export function OrganizationSuspendModal({ org, onClose }: OrganizationSuspendModalProps) {
  const { t } = useTranslation("common");
  const setSuspension = useSetOrganisationSuspension();
  const [reason, setReason] = useState("");
  const suspended = !!org?.suspendedAt;

  const close = () => {
    setReason("");
    setSuspension.reset();
    onClose();
  };
  const confirm = () => {
    if (!org) return;
    setSuspension.mutate({ institutionId: org.id, reason: suspended ? null : reason }, { onSuccess: close });
  };

  return (
    <Modal
      opened={!!org}
      onClose={close}
      title={suspended ? t("admin.organizations.liftTitle") : t("admin.organizations.suspendTitle")}
    >
      <Stack gap="md">
        <BodyText size="sm">
          {suspended
            ? t("admin.organizations.liftConfirm", { name: org?.name })
            : t("admin.organizations.suspendConfirm", { name: org?.name })}
        </BodyText>
        {!suspended && (
          <TextInput
            label={t("admin.organizations.suspendReason")}
            description={t("admin.organizations.suspendReasonDescription")}
            value={reason}
            onChange={(e) => setReason(e.currentTarget.value)}
            maxLength={300}
          />
        )}
        {setSuspension.isError && (
          <Alert color="red" icon={<IconAlertCircle size={16} />}>
            {t("admin.organizations.suspendError")}
          </Alert>
        )}
        <Group justify="flex-end">
          <CancelButton onClick={close}>{t("cancel")}</CancelButton>
          {suspended ? (
            <ActionButton onClick={confirm} loading={setSuspension.isPending}>
              {t("admin.organizations.lift")}
            </ActionButton>
          ) : (
            <DangerButton onClick={confirm} loading={setSuspension.isPending}>
              {t("admin.organizations.suspend")}
            </DangerButton>
          )}
        </Group>
      </Stack>
    </Modal>
  );
}

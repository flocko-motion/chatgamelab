import { Alert, Card, Group, Modal, Stack } from "@mantine/core";
import { IconAlertCircle } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { CancelButton, DangerButton } from "@components/buttons";
import { BodyText, CardTitle } from "@components/typography";
import { formatDate } from "@/common/lib/formatters";
import { useConsentWithdrawal } from "../hooks/useConsentWithdrawal";

/**
 * A head's card on their organisation's consent to the declaration: which
 * version, since when, and withdrawing it, which suspends the organisation.
 */
export function OrganisationDeclarationCard() {
  const { t, i18n } = useTranslation("common");
  const w = useConsentWithdrawal();
  if (!w.status?.consentedVersion) return null;

  return (
    <Card withBorder padding="lg" radius="md">
      <Stack gap="sm">
        <CardTitle>{t("myOrganization.declaration.title")}</CardTitle>
        <BodyText size="sm">
          {t("myOrganization.declaration.consented", {
            version: w.status.consentedVersion,
            date: formatDate(w.status.consentedAt, i18n.language),
          })}
        </BodyText>
        <Group>
          <DangerButton onClick={w.open}>{t("myOrganization.declaration.withdraw")}</DangerButton>
        </Group>
      </Stack>
      <Modal opened={w.confirming} onClose={w.close} title={t("myOrganization.declaration.withdrawTitle")}>
        <Stack gap="md">
          <BodyText size="sm">{t("myOrganization.declaration.withdrawConfirm")}</BodyText>
          {w.failed && (
            <Alert color="red" icon={<IconAlertCircle size={16} />}>
              {t("myOrganization.declaration.withdrawFailed")}
            </Alert>
          )}
          <Group justify="flex-end">
            <CancelButton onClick={w.close}>{t("cancel")}</CancelButton>
            <DangerButton onClick={w.confirm} loading={w.withdrawing}>
              {t("myOrganization.declaration.withdraw")}
            </DangerButton>
          </Group>
        </Stack>
      </Modal>
    </Card>
  );
}

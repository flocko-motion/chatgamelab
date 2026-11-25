import { Alert, Card, Group, Modal, Stack } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { IconAlertCircle, IconFileText } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { ActionButton, CancelButton, DangerButton } from "@components/buttons";
import { DeclarationConsentPage } from "@/features/organization-registration";
import { BodyText, CardTitle } from "@components/typography";
import { formatDate } from "@/common/lib/formatters";
import { useConsentWithdrawal } from "../hooks/useConsentWithdrawal";

/**
 * A head's card on their organisation's consent to the declaration: which
 * version, since when, and withdrawing it, which suspends the organisation.
 * While the consent is to an older version, it asks the head to agree to the
 * current one; the organisation stays active meanwhile.
 */
export function OrganisationDeclarationCard() {
  const { t, i18n } = useTranslation("common");
  const w = useConsentWithdrawal();
  const [renewing, renew] = useDisclosure(false);
  if (!w.status?.consentedVersion) return null;
  const behind = w.status.pending && !w.status.withdrawnAt;

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
        {behind && (
          <Alert color="orange" icon={<IconFileText size={16} />}>
            {t("myOrganization.declaration.newVersion", { version: w.status.currentVersion })}
          </Alert>
        )}
        <Group>
          {behind && <ActionButton onClick={renew.open}>{t("myOrganization.declaration.agree")}</ActionButton>}
          <DangerButton onClick={w.open}>{t("myOrganization.declaration.withdraw")}</DangerButton>
        </Group>
      </Stack>
      <Modal opened={renewing} onClose={renew.close} size="xl" padding={0} withCloseButton={false}>
        <DeclarationConsentPage onLater={renew.close} onConsented={renew.close} />
      </Modal>
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

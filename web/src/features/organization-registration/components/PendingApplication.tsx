import { useState } from "react";
import { Alert, Card, Container, Stack, Text } from "@mantine/core";
import { IconAlertCircle, IconCheck, IconMailForward } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { ActionButton, TextButton } from "@components/buttons";
import { SectionTitle } from "@components/typography";
import type { ObjInstitutionApplication } from "@/api/generated";
import { useAuthenticatedApi } from "@/api/useAuthenticatedApi";
import { organizationErrorMessage } from "../lib/errors";

interface PendingApplicationProps {
  application: ObjInstitutionApplication;
  /** Lets the person start over with different details (the open link stops working). */
  onEdit: () => void;
}

export function PendingApplication({ application, onEdit }: PendingApplicationProps) {
  const { t } = useTranslation("auth");
  const api = useAuthenticatedApi();
  const [sending, setSending] = useState(false);
  const [sent, setSent] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const resend = async () => {
    if (!api) return;
    setSending(true);
    setError(null);
    try {
      await api.institutionApplications.resendCreate();
      setSent(true);
    } catch (e) {
      setError(organizationErrorMessage(e, t));
    } finally {
      setSending(false);
    }
  };

  return (
    <Container size="xs" py="xl">
      <Card shadow="md" padding="xl" radius="md" withBorder>
        <Stack gap="md" align="stretch">
          <Stack gap="xs" align="center" ta="center">
            <IconMailForward size={48} stroke={1.5} color="var(--mantine-color-blue-6)" />
            <SectionTitle>{t("organizationRegistration.pending.title")}</SectionTitle>
          </Stack>
          <Text size="sm">
            {t("organizationRegistration.pending.text", {
              email: application.email,
              name: application.institutionName,
            })}
          </Text>
          <Text size="sm" c="dimmed">
            {t("organizationRegistration.pending.validity")}
          </Text>
          {sent && (
            <Alert color="green" icon={<IconCheck size={16} />}>
              {t("organizationRegistration.pending.resent")}
            </Alert>
          )}
          {error && (
            <Alert color="red" icon={<IconAlertCircle size={16} />}>
              {error}
            </Alert>
          )}
          <ActionButton onClick={resend} loading={sending} fullWidth>
            {t("organizationRegistration.pending.resend")}
          </ActionButton>
          <TextButton onClick={onEdit}>{t("organizationRegistration.pending.edit")}</TextButton>
        </Stack>
      </Card>
    </Container>
  );
}

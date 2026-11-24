import { Anchor, Card, Center, Stack } from "@mantine/core";
import { useBranding } from "@/providers/branding";
import { IconLock } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { ActionButton } from "@components/buttons";
import { BodyText, SectionTitle } from "@components/typography";

interface OrganisationSuspendedMessageProps {
  /** Suspended for want of a consent, which a head gives; otherwise by an admin. */
  consentWanted: boolean;
  onLogout: () => void;
}

/** Shown to the members of a suspended organisation who cannot lift the suspension. */
export function OrganisationSuspendedMessage({ consentWanted, onLogout }: OrganisationSuspendedMessageProps) {
  const { t } = useTranslation("common");
  const branding = useBranding();
  return (
    <Center py="xl">
      <Card shadow="md" p="xl" radius="md" withBorder maw={600} w="100%">
        <Stack align="center" gap="lg" ta="center">
          <IconLock size={64} color="var(--mantine-color-orange-5)" />
          <SectionTitle>{t("organisationSuspended.title")}</SectionTitle>
          <BodyText c="dimmed">
            {consentWanted ? t("organisationSuspended.consentWanted") : t("organisationSuspended.byAdmin")}
          </BodyText>
          <BodyText size="sm">
            {t("organisationSuspended.contact")}{" "}
            <Anchor href={`mailto:${branding.supportEmail}`}>{branding.supportEmail}</Anchor>
          </BodyText>
          <ActionButton onClick={onLogout}>{t("organisationSuspended.logout")}</ActionButton>
        </Stack>
      </Card>
    </Center>
  );
}

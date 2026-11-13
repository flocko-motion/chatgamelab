import { Button, Card, Center, Stack, Text } from "@mantine/core";
import { IconLock } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { CONTACT_EMAILS } from "@/config/helpLinks";

interface InstitutionLockedMessageProps {
  onLogout: () => void;
}

/** Shown to heads, staff and participants of an organisation an admin has locked. */
export function InstitutionLockedMessage({ onLogout }: InstitutionLockedMessageProps) {
  const { t } = useTranslation("common");
  return (
    <Center py="xl">
      <Card shadow="md" p="xl" radius="md" withBorder maw={600} w="100%">
        <Stack align="center" gap="lg">
          <IconLock size={64} color="var(--mantine-color-orange-5)" />
          <Text size="xl" fw={600} ta="center">
            {t("organizationLocked.title")}
          </Text>
          <Text c="dimmed" ta="center">
            {t("organizationLocked.description")}
          </Text>
          <Text size="sm" ta="center">
            {t("organizationLocked.contact")}{" "}
            <a href={`mailto:${CONTACT_EMAILS.SUPPORT}`}>{CONTACT_EMAILS.SUPPORT}</a>
          </Text>
          <Button onClick={onLogout}>
            {t("organizationLocked.logout")}
          </Button>
        </Stack>
      </Card>
    </Center>
  );
}

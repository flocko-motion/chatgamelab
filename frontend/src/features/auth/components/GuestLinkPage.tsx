import { Container, Stack, Card, Loader } from "@mantine/core";
import { useTranslation } from "react-i18next";
import { HelperText } from "@components/typography";
import { useGuestLinkLogin } from "../hooks/useGuestLinkLogin";

interface GuestLinkPageProps {
  /** The Guest's token from the link, whole or as its words. */
  token: string;
}

/** The sign-in link of a Guest in a workshop: signs in, then opens the workshop. */
export function GuestLinkPage({ token }: GuestLinkPageProps) {
  const { t } = useTranslation("common");
  useGuestLinkLogin(token);

  return (
    <Container size="sm" py="xl">
      <Card shadow="sm" padding="xl" radius="md" withBorder>
        <Stack align="center" gap="md">
          <Loader size="lg" />
          <HelperText>{t("invites.guest.loggingIn")}</HelperText>
        </Stack>
      </Card>
    </Container>
  );
}

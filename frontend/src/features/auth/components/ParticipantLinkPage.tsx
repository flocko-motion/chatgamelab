import { Container, Stack, Card, Loader } from "@mantine/core";
import { useTranslation } from "react-i18next";
import { HelperText } from "@components/typography";
import { useParticipantLinkLogin } from "../hooks/useParticipantLinkLogin";

interface ParticipantLinkPageProps {
  /** The participant token from the link, whole or as its words. */
  token: string;
}

/** A participant's sign-in link: signs in, then opens the workshop. */
export function ParticipantLinkPage({ token }: ParticipantLinkPageProps) {
  const { t } = useTranslation("common");
  useParticipantLinkLogin(token);

  return (
    <Container size="sm" py="xl">
      <Card shadow="sm" padding="xl" radius="md" withBorder>
        <Stack align="center" gap="md">
          <Loader size="lg" />
          <HelperText>{t("invites.participant.loggingIn")}</HelperText>
        </Stack>
      </Card>
    </Container>
  );
}

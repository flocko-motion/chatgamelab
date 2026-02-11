import { createFileRoute } from "@tanstack/react-router";
import { Container, Stack, Card, Loader, Text } from "@mantine/core";
import { useTranslation } from "react-i18next";
import { useEffect } from "react";
import { storeParticipantToken } from "@/providers/AuthProvider";
import { ROUTES } from "@/common/routes/routes";
import { buildShareUrl } from "@/common/lib/url";

export const Route = createFileRoute("/invites/participant/$token")({
  component: ParticipantLoginPage,
});

function ParticipantLoginPage() {
  const { t } = useTranslation("common");
  const { token } = Route.useParams();

  useEffect(() => {
    // Store the participant token and navigate to workshop
    // The AuthProvider will validate the token and handle authentication
    storeParticipantToken(token);

    // Navigate to workshop - AuthProvider will pick up the stored token
    window.location.href = buildShareUrl(ROUTES.MY_WORKSHOP);
  }, [token]);

  return (
    <Container size="sm" py="xl">
      <Card shadow="sm" padding="xl" radius="md" withBorder>
        <Stack align="center" gap="md">
          <Loader size="lg" />
          <Text c="dimmed">{t("invites.participant.loggingIn")}</Text>
        </Stack>
      </Card>
    </Container>
  );
}

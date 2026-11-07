import { useState } from "react";
import { Stack, Text, Card, Center, Button, Divider } from "@mantine/core";
import { IconSchoolOff, IconQrcode } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";

import { ParticipantCodeDisplay } from "./share";
import { getStoredParticipantToken } from "@/providers/AuthProvider";
import { speakableCode } from "@/common/lib/wordToken";

interface InactiveWorkshopMessageProps {
  onLogout: () => void;
}

export function InactiveWorkshopMessage({
  onLogout,
}: InactiveWorkshopMessageProps) {
  const { t } = useTranslation("common");
  // Read once: logging out clears the token, and a later render must not lose the
  // code while it is still on screen.
  const [token] = useState(() => getStoredParticipantToken());
  const [showCode, setShowCode] = useState(false);

  return (
    <Center py="xl">
      <Card shadow="md" p="xl" radius="md" withBorder maw={700} w="100%">
        <Stack align="center" gap="lg">
          <IconSchoolOff size={64} color="var(--mantine-color-orange-5)" />
          <Text size="xl" fw={600} ta="center">
            {t("workshop.inactive.title")}
          </Text>
          <Text c="dimmed" ta="center">
            {t("workshop.inactive.description")}
          </Text>

          {/* The code cannot be fetched here - the backend answers 403 before any
              handler runs - but the browser still holds it, and logging out throws
              it away. Offer it before that button. */}
          <Text size="sm" ta="center" fw={500}>
            {token
              ? t("workshop.inactive.photoFirst")
              : t("workshop.inactive.noCode")}
          </Text>

          {token && !showCode && (
            <Button
              leftSection={<IconQrcode size={18} />}
              onClick={() => setShowCode(true)}
            >
              {t("workshop.inactive.showCode")}
            </Button>
          )}

          {token && showCode && (
            <>
              <Divider w="100%" />
              <ParticipantCodeDisplay
                token={token}
                words={speakableCode(token)}
              />
              <Divider w="100%" />
            </>
          )}

          <Button variant="subtle" color="gray" onClick={onLogout}>
            {t("logout")}
          </Button>
        </Stack>
      </Card>
    </Center>
  );
}

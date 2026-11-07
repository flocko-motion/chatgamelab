import { Card, Center, Stack, Text } from "@mantine/core";
import { IconTools } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";

/** Shown on the play routes until the engine's player mounts there. */
export function PlayerPlaceholder() {
  const { t } = useTranslation("common");

  return (
    <Center py="xl">
      <Card shadow="md" p="xl" radius="md" withBorder maw={700} w="100%">
        <Stack align="center" gap="lg">
          <IconTools size={64} color="var(--mantine-color-accent-5)" />
          <Text size="xl" fw={600} ta="center">
            {t("gamePlayer.rebuilding.title")}
          </Text>
          <Text c="dimmed" ta="center">
            {t("gamePlayer.rebuilding.description")}
          </Text>
        </Stack>
      </Card>
    </Center>
  );
}

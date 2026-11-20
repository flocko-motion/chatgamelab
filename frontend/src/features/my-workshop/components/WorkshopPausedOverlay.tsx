import { Overlay, Center, Stack, Title, Text } from "@mantine/core";
import { IconPlayerPause } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";

/**
 * Full-area overlay shown when a workshop is paused by head/staff.
 * Blocks interaction with the content area underneath.
 * Uses Mantine primitives directly since this is a dark-background overlay
 * that doesn't fit the semantic component color scheme.
 */
export function WorkshopPausedOverlay() {
  const { t } = useTranslation("myWorkshop");

  return (
    <Overlay backgroundOpacity={0.75} color="#000" zIndex={100} blur={2}>
      <Center h="100%">
        <Stack align="center" gap="md" p="xl">
          <IconPlayerPause size={48} color="var(--mantine-color-gray-4)" />
          <Title order={2} c="white" ta="center">
            {t("paused.title")}
          </Title>
          <Text c="gray.4" ta="center" maw={400}>
            {t("paused.description")}
          </Text>
        </Stack>
      </Center>
    </Overlay>
  );
}

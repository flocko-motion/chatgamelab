import { Card, Stack, Text } from "@mantine/core";
import { IconMoodEmpty } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";

export function WorkshopEmptyState() {
  const { t } = useTranslation("myWorkshop");

  return (
    <Card shadow="sm" p="xl" radius="md" withBorder>
      <Stack align="center" gap="md" py="xl">
        <IconMoodEmpty size={48} color="var(--mantine-color-gray-5)" />
        <Text c="gray.6" ta="center">
          {t("empty.title")}
        </Text>
        <Text size="sm" c="gray.5" ta="center">
          {t("empty.description")}
        </Text>
      </Stack>
    </Card>
  );
}

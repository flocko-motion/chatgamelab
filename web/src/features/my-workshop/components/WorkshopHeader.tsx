import { Group, Badge, Text } from "@mantine/core";
import { IconSchool } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { PageTitle } from "@components/typography";

interface WorkshopHeaderProps {
  workshopName?: string;
  organizationName?: string;
}

export function WorkshopHeader({
  workshopName,
  organizationName,
}: WorkshopHeaderProps) {
  const { t } = useTranslation("myWorkshop");

  return (
    <>
      <PageTitle>{t("title")}</PageTitle>
      <Group gap="md">
        {workshopName && (
          <Badge
            size="lg"
            color="accent"
            variant="light"
            leftSection={<IconSchool size={14} />}
          >
            {workshopName}
          </Badge>
        )}
        {organizationName && (
          <Text size="sm" c="dimmed">
            {organizationName}
          </Text>
        )}
      </Group>
    </>
  );
}

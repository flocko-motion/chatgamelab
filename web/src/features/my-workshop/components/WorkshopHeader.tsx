import { Text } from "@mantine/core";
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
      <PageTitle>{workshopName || t("title")}</PageTitle>
      {organizationName && (
        <Text size="sm" c="dimmed">
          {t("organizator", { name: organizationName })}
        </Text>
      )}
    </>
  );
}

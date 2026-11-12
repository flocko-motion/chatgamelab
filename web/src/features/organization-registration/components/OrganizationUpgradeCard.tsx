import { Card, Group, Stack, Text } from "@mantine/core";
import { IconBuilding } from "@tabler/icons-react";
import { useNavigate } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { ActionButton } from "@components/buttons";
import { SectionTitle } from "@components/typography";
import { useAuth } from "@/providers/AuthProvider";
import { ROUTES } from "@/common/routes/routes";
import { AgeGroup } from "@/constants/ageGroup";

/** Settings card for accounts without an organisation: "Zur Organisation upgraden". */
export function OrganizationUpgradeCard() {
  const { t } = useTranslation("auth");
  const navigate = useNavigate();
  const { backendUser } = useAuth();

  const role = backendUser?.role?.role;
  if (!backendUser || (role && role !== "individual")) return null;

  const isAdult = backendUser.ageGroup === AgeGroup.U18;

  return (
    <Card shadow="sm" padding="xl" radius="md" withBorder>
      <Stack gap="md">
        <SectionTitle>{t("organizationRegistration.upgradeCard.title")}</SectionTitle>
        <Text size="sm">
          {isAdult
            ? t("organizationRegistration.upgradeCard.text")
            : t("organizationRegistration.blocked.age")}
        </Text>
        {isAdult && (
          <Group>
            <ActionButton
              leftSection={<IconBuilding size={18} />}
              onClick={() => navigate({ to: ROUTES.ORGANIZATION_APPLY as "/" })}
            >
              {t("organizationRegistration.upgradeCard.button")}
            </ActionButton>
          </Group>
        )}
      </Stack>
    </Card>
  );
}

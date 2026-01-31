import { createFileRoute } from "@tanstack/react-router";
import { Container, Title, Text, Stack, Alert } from "@mantine/core";
import { IconAlertCircle } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useAuth } from "@/providers/AuthProvider";
import { useWorkshopMode } from "@/providers/WorkshopModeProvider";
import { getUserInstitutionId, Role, hasRole } from "@/common/lib/roles";
import { SingleWorkshopSettings } from "@/features/my-organization/components/SingleWorkshopSettings";

export const Route = createFileRoute("/my-workshop/settings")({
  component: WorkshopSettingsPage,
});

function WorkshopSettingsPage() {
  const { t } = useTranslation("common");
  const { backendUser } = useAuth();
  const { isInWorkshopMode, activeWorkshopId, activeWorkshopName } =
    useWorkshopMode();

  const institutionId = getUserInstitutionId(backendUser);
  const isHead = hasRole(backendUser, Role.Head);
  const isStaff = hasRole(backendUser, Role.Staff);
  const canManageWorkshops = isHead || isStaff;

  // Not in workshop mode
  if (!isInWorkshopMode || !activeWorkshopId) {
    return (
      <Container size="xl" py="xl">
        <Alert
          icon={<IconAlertCircle size={16} />}
          title={t("error")}
          color="yellow"
        >
          {t("myWorkshop.settings.notInWorkshopMode")}
        </Alert>
      </Container>
    );
  }

  // No organization
  if (!institutionId) {
    return (
      <Container size="xl" py="xl">
        <Alert
          icon={<IconAlertCircle size={16} />}
          title={t("myOrganization.noOrganization")}
          color="yellow"
        >
          {t("myOrganization.noOrganizationDescription")}
        </Alert>
      </Container>
    );
  }

  // Not authorized
  if (!canManageWorkshops) {
    return (
      <Container size="xl" py="xl">
        <Alert
          icon={<IconAlertCircle size={16} />}
          title={t("error")}
          color="red"
        >
          {t("myOrganization.workshops.notAuthorized")}
        </Alert>
      </Container>
    );
  }

  return (
    <Container size="xl" py="xl">
      <Stack gap="lg">
        {/* Header */}
        <Stack gap={0}>
          <Title order={2}>{t("myWorkshop.settings.title")}</Title>
          {activeWorkshopName && (
            <Text c="dimmed" size="sm">
              {activeWorkshopName}
            </Text>
          )}
        </Stack>

        {/* Single workshop settings */}
        <SingleWorkshopSettings
          workshopId={activeWorkshopId}
          institutionId={institutionId}
        />
      </Stack>
    </Container>
  );
}

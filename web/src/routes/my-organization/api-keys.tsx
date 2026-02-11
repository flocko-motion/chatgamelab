import { createFileRoute } from "@tanstack/react-router";
import { Container, Title, Text, Stack, Alert } from "@mantine/core";
import { IconAlertCircle } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useAuth } from "@/providers/AuthProvider";
import { getUserInstitutionId, Role, hasRole } from "@/common/lib/roles";
import { ApiKeysTab } from "@/features/my-organization/components/ApiKeysTab";
import { useQuery } from "@tanstack/react-query";
import { queryKeys } from "@/api/queryKeys";
import { useRequiredAuthenticatedApi } from "@/api/useAuthenticatedApi";

export const Route = createFileRoute("/my-organization/api-keys")({
  component: OrganizationApiKeysPage,
});

function OrganizationApiKeysPage() {
  const { t } = useTranslation("common");
  const { backendUser } = useAuth();
  const api = useRequiredAuthenticatedApi();

  const institutionId = getUserInstitutionId(backendUser);
  const isHead = hasRole(backendUser, Role.Head);
  const isStaff = hasRole(backendUser, Role.Staff);
  const canManageApiKeys = isHead || isStaff;

  // Fetch institution details
  const { data: institution } = useQuery({
    queryKey: queryKeys.institution(institutionId!),
    queryFn: async () => {
      if (!institutionId) return null;
      const response = await api.institutions.institutionsDetail(institutionId);
      return response.data;
    },
    enabled: !!institutionId,
  });

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
  if (!canManageApiKeys) {
    return (
      <Container size="xl" py="xl">
        <Alert
          icon={<IconAlertCircle size={16} />}
          title={t("error")}
          color="red"
        >
          {t("myOrganization.apiKeys.notAuthorized")}
        </Alert>
      </Container>
    );
  }

  return (
    <Container size="xl" py="xl">
      <Stack gap="lg">
        {/* Header */}
        <Stack gap={0}>
          <Title order={2}>{t("myOrganization.apiKeys.title")}</Title>
          {institution?.name && (
            <Text c="dimmed" size="sm">
              {institution.name}
            </Text>
          )}
        </Stack>

        {/* API Keys content */}
        <ApiKeysTab
          institutionId={institutionId}
          institutionName={institution?.name}
          freeUseApiKeyShareId={institution?.freeUseApiKeyShareId}
        />
      </Stack>
    </Container>
  );
}

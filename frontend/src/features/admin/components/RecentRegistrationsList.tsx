import { Alert, Group, Loader, ScrollArea, Stack, Table } from "@mantine/core";
import { IconAlertCircle } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { BodyText, HelperText } from "@components/typography";
import { useRecentlyRegisteredInstitutions } from "@/api/hooks";
import { formatDateTime } from "@/common/lib/formatters";
import { OrganizationStatusBadges } from "./OrganizationStatusBadges";

/** The organisations that registered themselves in the last two weeks, with who signed for them. */
export function RecentRegistrationsList() {
  const { t, i18n } = useTranslation("common");
  const { data, isLoading, error } = useRecentlyRegisteredInstitutions();

  if (isLoading) {
    return (
      <Group justify="center" py="md">
        <Loader size="sm" />
      </Group>
    );
  }
  if (error) {
    return (
      <Alert color="red" icon={<IconAlertCircle size={16} />}>
        {t("admin.registrations.loadError")}
      </Alert>
    );
  }
  if (!data || data.length === 0) {
    return <HelperText ta="center">{t("admin.registrations.empty")}</HelperText>;
  }

  return (
    <ScrollArea>
      <Table striped highlightOnHover miw={760}>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>{t("admin.registrations.date")}</Table.Th>
            <Table.Th>{t("admin.registrations.institution")}</Table.Th>
            <Table.Th>{t("admin.registrations.person")}</Table.Th>
            <Table.Th>{t("admin.organizations.status")}</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {data.map((org) => (
            <Table.Tr key={org.id}>
              <Table.Td>
                <BodyText size="sm">{formatDateTime(org.registration?.at, i18n.language)}</BodyText>
              </Table.Td>
              <Table.Td>
                <Stack gap={0}>
                  <BodyText size="sm" fw={500}>
                    {org.name}
                  </BodyText>
                  <HelperText>{[org.operator, org.city].filter(Boolean).join(" · ")}</HelperText>
                </Stack>
              </Table.Td>
              <Table.Td>
                {org.registration && (
                  <Stack gap={0}>
                    <BodyText size="sm">{`${org.registration.personName} (${org.registration.personFunction})`}</BodyText>
                    <HelperText>{org.registration.email}</HelperText>
                  </Stack>
                )}
              </Table.Td>
              <Table.Td>
                <OrganizationStatusBadges org={org} />
              </Table.Td>
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>
    </ScrollArea>
  );
}

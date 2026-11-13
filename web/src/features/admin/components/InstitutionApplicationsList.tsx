import {
  Alert,
  Badge,
  Loader,
  Group,
  ScrollArea,
  Stack,
  Table,
  Text,
} from "@mantine/core";
import { IconAlertCircle } from "@tabler/icons-react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { useRequiredAuthenticatedApi } from "@/api/useAuthenticatedApi";
import type { ObjInstitutionApplication } from "@/api/generated";
import { INSTITUTION_APPLICATIONS_QUERY_KEY } from "../lib/institutionApplications";

const STATUS_COLOR: Record<string, string> = {
  pending: "blue",
  confirmed: "green",
  expired: "gray",
};

/**
 * Admin list of organisation self-registrations, including attempts that were never confirmed.
 * Personal data of old unconfirmed attempts is removed automatically; the row then says so.
 * Stage 1 (no e-mail confirmation): new registrations are confirmed right away.
 */
export function InstitutionApplicationsList() {
  const { t, i18n } = useTranslation("common");
  const api = useRequiredAuthenticatedApi();

  const { data, isLoading, error } = useQuery({
    queryKey: INSTITUTION_APPLICATIONS_QUERY_KEY,
    queryFn: async () =>
      (await api.institutionApplications.institutionApplicationsList()).data,
  });

  const date = (iso?: string) =>
    iso
      ? new Date(iso).toLocaleString(i18n.language, {
          dateStyle: "short",
          timeStyle: "short",
        })
      : "";

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
        {t("admin.applications.loadError")}
      </Alert>
    );
  }
  if (!data || data.length === 0) {
    return (
      <Text c="dimmed" ta="center" py="md">
        {t("admin.applications.empty")}
      </Text>
    );
  }

  return (
    <Stack gap="sm">
      <ScrollArea>
        <Table striped highlightOnHover miw={760}>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>{t("admin.applications.date")}</Table.Th>
              <Table.Th>{t("admin.applications.institution")}</Table.Th>
              <Table.Th>{t("admin.applications.person")}</Table.Th>
              <Table.Th>{t("admin.applications.kind")}</Table.Th>
              <Table.Th>{t("admin.applications.status")}</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {data.map((a: ObjInstitutionApplication) => (
              <Table.Tr key={a.id}>
                <Table.Td>
                  <Text size="sm">{date(a.createdAt)}</Text>
                </Table.Td>
                <Table.Td>
                  <Stack gap={0}>
                    <Text size="sm" fw={500}>
                      {a.institutionName}
                    </Text>
                    <Text size="xs" c="dimmed">
                      {[a.operator, a.city].filter(Boolean).join(" · ")}
                    </Text>
                  </Stack>
                </Table.Td>
                <Table.Td>
                  {a.anonymizedAt ? (
                    <Text size="sm" c="dimmed" fs="italic">
                      {t("admin.applications.anonymized")}
                    </Text>
                  ) : (
                    <Stack gap={0}>
                      <Text size="sm">
                        {a.personName}
                        {a.personFunction ? ` (${a.personFunction})` : ""}
                      </Text>
                      <Text size="xs" c="dimmed">
                        {a.email}
                      </Text>
                    </Stack>
                  )}
                </Table.Td>
                <Table.Td>
                  <Text size="sm">
                    {a.kind === "upgrade"
                      ? t("admin.applications.kindUpgrade")
                      : t("admin.applications.kindRegister")}
                  </Text>
                </Table.Td>
                <Table.Td>
                  <Stack gap={2}>
                    <Badge
                      size="sm"
                      variant="light"
                      color={STATUS_COLOR[a.status ?? ""] ?? "gray"}
                    >
                      {t(
                        `admin.applications.statusValues.${a.status ?? "pending"}`,
                      )}
                    </Badge>
                    {a.status === "confirmed" && a.confirmedAt && (
                      <Text size="xs" c="dimmed">
                        {date(a.confirmedAt)}
                      </Text>
                    )}
                  </Stack>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      </ScrollArea>
    </Stack>
  );
}

import { Card, Group, Stack } from "@mantine/core";
import { useTranslation } from "react-i18next";
import type { OrganisationListSort } from "@/api/hooks";
import { useOrganisations } from "@/api/hooks";
import type { OrganisationView } from "@/api/types";
import { ListTable, type ListColumn } from "@components/ListTable";
import { BodyText, HelperText } from "@components/typography";
import { formatDateTime } from "@/common/lib/formatters";
import {
  RECENT_REGISTRATION_PRESET,
  type RecentRegistrationColumnKey,
} from "../lib/organisationListPresets";
import { operatorAndCity } from "../lib/organizationHeads";
import { OrganizationStatusBadges } from "./OrganizationStatusBadges";

type RegistrationColumn = ListColumn<OrganisationView, OrganisationListSort>;

/**
 * The organisations that registered themselves in the last two weeks, with
 * who signed for them. Embedded in the organisation page; it refreshes itself.
 */
export function RecentRegistrationsList() {
  const { t, i18n } = useTranslation("common");
  const list = useOrganisations(RECENT_REGISTRATION_PRESET.list);

  const registeredAt = (org: OrganisationView) =>
    formatDateTime(org.registration?.at, i18n.language);
  const organisation = (org: OrganisationView) => (
    <Stack gap={0}>
      <BodyText size="sm" fw={500}>
        {org.name}
      </BodyText>
      <HelperText>{operatorAndCity(org)}</HelperText>
    </Stack>
  );
  const person = (org: OrganisationView) =>
    org.registration && (
      <Stack gap={0}>
        <BodyText size="sm">{`${org.registration.personName} (${org.registration.personFunction})`}</BodyText>
        <HelperText>{org.registration.email}</HelperText>
      </Stack>
    );

  const columns: Record<RecentRegistrationColumnKey, RegistrationColumn> = {
    registered: {
      key: "registered",
      header: t("admin.registrations.date"),
      sortBy: "registeredAt",
      width: 180,
      render: (org) => <BodyText size="sm">{registeredAt(org)}</BodyText>,
    },
    organisation: {
      key: "organisation",
      header: t("admin.registrations.institution"),
      sortBy: "name",
      render: organisation,
    },
    person: { key: "person", header: t("admin.registrations.person"), render: person },
    status: {
      key: "status",
      header: t("admin.organizations.status"),
      render: (org) => <OrganizationStatusBadges org={org} />,
    },
  };

  const renderCard = (org: OrganisationView) => (
    <Card withBorder padding="sm" radius="md">
      <Stack gap="xs">
        <Group justify="space-between" wrap="nowrap" align="flex-start">
          {organisation(org)}
          <HelperText>{registeredAt(org)}</HelperText>
        </Group>
        {person(org)}
        <OrganizationStatusBadges org={org} />
      </Stack>
    </Card>
  );

  return (
    <ListTable
      compact
      list={list}
      columns={RECENT_REGISTRATION_PRESET.columns.map((key) => columns[key])}
      getRowKey={(org) => org.id}
      renderCard={renderCard}
      empty={{ title: t("admin.registrations.empty") }}
      errorMessage={t("admin.registrations.loadError")}
    />
  );
}

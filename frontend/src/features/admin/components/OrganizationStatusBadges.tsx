import { Badge, Group, Tooltip } from "@mantine/core";
import { useTranslation } from "react-i18next";
import type { OrganisationView } from "@/api/types";
import { formatDate } from "@/common/lib/formatters";

/**
 * An organisation's suspension and consent, as admins see them in their
 * list: suspended by an admin (with the reason), or for want of a consent.
 */
export function OrganizationStatusBadges({ org }: { org: OrganisationView }) {
  const { t, i18n } = useTranslation("common");
  const declaration = org.declaration;

  let consent: string | undefined;
  if (declaration?.withdrawnAt) consent = t("admin.organizations.consentWithdrawn");
  else if (declaration?.pending && declaration.consentedVersion)
    consent = t("admin.organizations.consentOutdated", { version: declaration.consentedVersion });
  else if (declaration?.pending) consent = t("admin.organizations.consentMissing");
  else if (declaration)
    consent = t("admin.organizations.consentGiven", {
      version: declaration.consentedVersion,
      date: formatDate(declaration.consentedAt, i18n.language),
    });

  return (
    <Group gap={4} wrap="wrap">
      {org.suspendedAt ? (
        <Tooltip label={org.suspendedReason || t("admin.organizations.suspendedNoReason")} withArrow>
          <Badge color="red" variant="filled" size="sm">
            {t("admin.organizations.statusSuspended")}
          </Badge>
        </Tooltip>
      ) : org.suspended ? (
        <Badge color="orange" variant="filled" size="sm">
          {t("admin.organizations.statusSuspendedConsent")}
        </Badge>
      ) : (
        <Badge color="green" variant="light" size="sm">
          {t("admin.organizations.statusActive")}
        </Badge>
      )}
      {consent && (
        <Badge color={declaration?.pending ? "orange" : "gray"} variant="light" size="sm">
          {consent}
        </Badge>
      )}
    </Group>
  );
}

import { Alert, Anchor } from "@mantine/core";
import { IconFileText } from "@tabler/icons-react";
import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { ROUTES } from "@/common/routes/routes";
import { useConsentWithdrawal } from "../hooks/useConsentWithdrawal";

/**
 * Reminds a head whose organisation consented to an older declaration to
 * agree to the current one; nothing for anyone else.
 */
export function DeclarationReminder() {
  const { t } = useTranslation("common");
  const { status } = useConsentWithdrawal();
  if (!status?.pending || !status.consentedVersion || status.withdrawnAt) return null;

  return (
    <Alert color="orange" icon={<IconFileText size={16} />}>
      {t("myOrganization.declaration.newVersion", { version: status.currentVersion })}{" "}
      <Anchor component={Link} to={ROUTES.MY_ORGANIZATION} size="sm">
        {t("myOrganization.declaration.goAgree")}
      </Anchor>
    </Alert>
  );
}

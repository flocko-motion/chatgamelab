import { Alert, Card, Container, Group, Loader, Stack, TextInput } from "@mantine/core";
import { IconAlertCircle } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { ActionButton, TextButton } from "@components/buttons";
import { BodyText, CardTitle, HelperText, SectionTitle } from "@components/typography";
import type { ConsentStatusView } from "@/api/types";
import { useDeclarationConsentForm } from "../hooks/useDeclarationConsentForm";
import { DeclarationConsent } from "./DeclarationConsent";
import { YouthProtectionField } from "./YouthProtectionField";

/** The intro's key: a new version, a withdrawn consent, or none yet. */
function introKey(status: ConsentStatusView | undefined): string {
  if (status?.withdrawnAt) return "organizationRegistration.consentPage.introWithdrawn";
  if (status?.consentedVersion) return "organizationRegistration.consentPage.introNewVersion";
  return "organizationRegistration.consentPage.introExisting";
}

/**
 * A head's consent to the current declaration: the declaration, with the
 * organisation's details and youth-protection restriction to confirm. Shown
 * as a page while the organisation is suspended for want of a consent (with
 * onLogout), or in a dialog to bring an older consent up to date (with
 * onLater and onConsented).
 */
export function DeclarationConsentPage({
  onLogout,
  onLater,
  onConsented,
}: {
  onLogout?: () => void;
  onLater?: () => void;
  onConsented?: () => void;
}) {
  const { t } = useTranslation("auth");
  const p = useDeclarationConsentForm(onConsented);

  if (p.loadFailed) {
    return (
      <Container size="sm" py="xl">
        <Alert color="red" icon={<IconAlertCircle size={16} />}>
          {t("organizationRegistration.errors.loadFailed")}
        </Alert>
      </Container>
    );
  }
  if (!p.declaration || p.constraint === null) {
    return (
      <Group justify="center" py="xl">
        <Loader />
      </Group>
    );
  }

  return (
    <Container size="sm" py={{ base: "md", sm: "xl" }}>
      <Card shadow="md" padding="xl" radius="md" withBorder>
        <Stack gap="md">
          <SectionTitle>{t("organizationRegistration.consentPage.title")}</SectionTitle>
          <BodyText size="sm">
            {t(introKey(p.status), { name: p.organisationName, version: p.declaration.declaration.version })}
          </BodyText>
          <HelperText>{t("organizationRegistration.consentPage.oneHeadIsEnough")}</HelperText>

          <CardTitle>{t("organizationRegistration.personSection")}</CardTitle>
          <TextInput
            label={t("organizationRegistration.personNameLabel")}
            value={p.form.personName}
            onChange={(e) => p.set("personName", e.currentTarget.value)}
            error={p.fieldError("personName")}
            maxLength={100}
            required
          />
          <TextInput
            label={t("organizationRegistration.personFunctionLabel")}
            placeholder={t("organizationRegistration.personFunctionPlaceholder")}
            value={p.form.personFunction}
            onChange={(e) => p.set("personFunction", e.currentTarget.value)}
            error={p.fieldError("personFunction")}
            maxLength={100}
            required
          />
          <TextInput
            type="email"
            label={t("organizationRegistration.emailLabel")}
            description={t("organizationRegistration.consentPage.emailDescription")}
            value={p.form.email}
            onChange={(e) => p.set("email", e.currentTarget.value)}
            error={p.fieldError("email")}
            maxLength={200}
            required
          />

          <CardTitle>{t("organizationRegistration.institutionSection")}</CardTitle>
          <TextInput
            label={t("organizationRegistration.operatorLabel")}
            description={t("organizationRegistration.operatorDescription")}
            value={p.form.operator}
            onChange={(e) => p.set("operator", e.currentTarget.value)}
            error={p.fieldError("operator")}
            maxLength={150}
            required
          />
          <TextInput
            label={t("organizationRegistration.streetLabel")}
            value={p.form.street}
            onChange={(e) => p.set("street", e.currentTarget.value)}
            error={p.fieldError("street")}
            maxLength={150}
            required
          />
          <Group grow align="flex-start">
            <TextInput
              label={t("organizationRegistration.postalCodeLabel")}
              value={p.form.postalCode}
              onChange={(e) => p.set("postalCode", e.currentTarget.value)}
              error={p.fieldError("postalCode")}
              maxLength={20}
              required
              maw={140}
            />
            <TextInput
              label={t("organizationRegistration.cityLabel")}
              value={p.form.city}
              onChange={(e) => p.set("city", e.currentTarget.value)}
              error={p.fieldError("city")}
              maxLength={150}
              required
            />
          </Group>

          <YouthProtectionField
            declaration={p.declaration}
            value={p.constraint}
            onChange={p.setConstraint}
            showErrors={p.showErrors}
            intro={t("organizationRegistration.consentPage.constraintIntro")}
          />

          <DeclarationConsent
            declaration={p.declaration.declaration}
            accepted={p.accepted}
            onChange={p.setAccepted}
            error={p.showErrors && !p.accepted ? t("organizationRegistration.errors.required") : undefined}
          />

          {p.error && (
            <Alert color="red" icon={<IconAlertCircle size={16} />}>
              {p.error}
            </Alert>
          )}
          <Group justify="space-between">
            {onLater ? (
              <TextButton onClick={onLater} disabled={p.saving}>
                {t("organizationRegistration.consentPage.later")}
              </TextButton>
            ) : (
              <TextButton onClick={onLogout} disabled={p.saving}>
                {t("organizationRegistration.consentPage.logout")}
              </TextButton>
            )}
            <ActionButton onClick={p.submit} loading={p.saving}>
              {t("organizationRegistration.consentPage.submit")}
            </ActionButton>
          </Group>
        </Stack>
      </Card>
    </Container>
  );
}

import { Alert, Anchor, Box, Card, Checkbox, Container, Group, Loader, Stack, TextInput } from "@mantine/core";
import { IconAlertCircle, IconBuilding, IconMail, IconUser } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { ActionButton, TextButton } from "@components/buttons";
import { BodyText, CardTitle, ErrorText, HelperText, SectionTitle } from "@components/typography";
import { HELP_LINKS } from "@/config/helpLinks";
import { useOrganisationForm } from "../hooks/useOrganisationForm";
import { DeclarationConsent } from "./DeclarationConsent";
import { YouthProtectionField } from "./YouthProtectionField";

interface OrganizationRegistrationFormProps {
  /** Suggested display name of the new account. */
  initialName?: string;
  /** The sign-in's email; fixed when given. */
  initialEmail?: string;
  /** Called once the organisation exists and the caller is its head. */
  onSubmitted: () => void;
  onCancel?: () => void;
}

/**
 * The organisation's form, which completes a new account's registration: the
 * person with their display name, the organisation, its youth-protection
 * restriction, and the consents with the declaration and the terms.
 */
export function OrganizationRegistrationForm({
  initialName,
  initialEmail,
  onSubmitted,
  onCancel,
}: OrganizationRegistrationFormProps) {
  const { t } = useTranslation("auth");
  const f = useOrganisationForm({ initialName, initialEmail, onSubmitted });

  if (f.loadFailed) {
    return (
      <Container size="sm" py="xl">
        <Alert color="red" icon={<IconAlertCircle size={16} />}>
          {t("organizationRegistration.errors.loadFailed")}
        </Alert>
      </Container>
    );
  }
  if (!f.declaration) {
    return (
      <Container size="sm" py="xl">
        <Group justify="center">
          <Loader />
        </Group>
      </Container>
    );
  }

  const emailDescription = f.emailLocked
    ? t("organizationRegistration.emailFromLogin")
    : t("organizationRegistration.emailDescription");

  return (
    <Container size="sm" py={{ base: "md", sm: "xl" }}>
      <Card shadow="md" padding="xl" radius="md" withBorder>
        <form onSubmit={f.submit} noValidate>
          <Stack gap="lg">
            <Box ta="center">
              <SectionTitle>{t("organizationRegistration.titleRegister")}</SectionTitle>
              <HelperText>{t("organizationRegistration.subtitle")}</HelperText>
            </Box>

            <Stack gap="sm">
              <CardTitle>{t("organizationRegistration.personSection")}</CardTitle>
              <TextInput
                label={t("organizationRegistration.accountNameLabel")}
                description={t("organizationRegistration.accountNameDescription")}
                leftSection={<IconUser size={16} />}
                maxLength={24}
                value={f.form.accountName}
                onChange={(e) => f.set("accountName", e.currentTarget.value)}
                error={f.fieldError("accountName")}
                required
              />
              <TextInput
                label={t("organizationRegistration.personNameLabel")}
                value={f.form.personName}
                maxLength={100}
                onChange={(e) => f.set("personName", e.currentTarget.value)}
                error={f.fieldError("personName")}
                autoComplete="name"
                required
              />
              <TextInput
                label={t("organizationRegistration.personFunctionLabel")}
                placeholder={t("organizationRegistration.personFunctionPlaceholder")}
                value={f.form.personFunction}
                maxLength={100}
                onChange={(e) => f.set("personFunction", e.currentTarget.value)}
                error={f.fieldError("personFunction")}
                required
              />
              <TextInput
                type="email"
                label={t("organizationRegistration.emailLabel")}
                description={emailDescription}
                leftSection={<IconMail size={16} />}
                value={f.form.email}
                maxLength={200}
                readOnly={f.emailLocked}
                variant={f.emailLocked ? "filled" : "default"}
                onChange={(e) => f.set("email", e.currentTarget.value)}
                error={
                  f.fieldError("email") ??
                  (f.emailInvalid ? t("organizationRegistration.errors.emailInvalid") : undefined)
                }
                autoComplete="email"
                required
              />
            </Stack>

            <Stack gap="sm">
              <CardTitle>{t("organizationRegistration.institutionSection")}</CardTitle>
              <TextInput
                label={t("organizationRegistration.institutionNameLabel")}
                leftSection={<IconBuilding size={16} />}
                value={f.form.institutionName}
                maxLength={150}
                onChange={(e) => f.set("institutionName", e.currentTarget.value)}
                error={f.fieldError("institutionName")}
                autoComplete="organization"
                required
              />
              <TextInput
                label={t("organizationRegistration.operatorLabel")}
                description={t("organizationRegistration.operatorDescription")}
                value={f.form.operator}
                maxLength={150}
                onChange={(e) => f.set("operator", e.currentTarget.value)}
                error={f.fieldError("operator")}
                required
              />
              <TextInput
                label={t("organizationRegistration.streetLabel")}
                value={f.form.street}
                maxLength={150}
                onChange={(e) => f.set("street", e.currentTarget.value)}
                error={f.fieldError("street")}
                autoComplete="street-address"
                required
              />
              <Group grow align="flex-start">
                <TextInput
                  label={t("organizationRegistration.postalCodeLabel")}
                  value={f.form.postalCode}
                  maxLength={20}
                  onChange={(e) => f.set("postalCode", e.currentTarget.value)}
                  error={f.fieldError("postalCode")}
                  autoComplete="postal-code"
                  required
                  maw={140}
                />
                <TextInput
                  label={t("organizationRegistration.cityLabel")}
                  value={f.form.city}
                  maxLength={150}
                  onChange={(e) => f.set("city", e.currentTarget.value)}
                  error={f.fieldError("city")}
                  autoComplete="address-level2"
                  required
                />
              </Group>
            </Stack>

            <YouthProtectionField
              declaration={f.declaration}
              value={f.form.promptConstraint}
              onChange={(v) => f.set("promptConstraint", v)}
              showErrors={f.showErrors}
            />

            <Stack gap="sm">
              <CardTitle>{t("organizationRegistration.consentSection")}</CardTitle>
              <Checkbox
                label={t("organizationRegistration.confirmAdult")}
                checked={f.form.confirmAdult}
                onChange={(e) => f.set("confirmAdult", e.currentTarget.checked)}
                error={f.checkError("confirmAdult")}
              />
              <Checkbox
                label={
                  <BodyText size="sm" component="span">
                    {t("registerPage.agreePrefix")}{" "}
                    <Anchor href={HELP_LINKS.TERMS_OF_SERVICE} target="_blank" rel="noopener noreferrer" size="sm">
                      {t("registerPage.termsOfService")}
                    </Anchor>{" "}
                    {t("registerPage.agreeSuffix")}{" "}
                    <Anchor href={HELP_LINKS.PRIVACY_POLICY} target="_blank" rel="noopener noreferrer" size="sm">
                      {t("registerPage.privacyPolicy")}
                    </Anchor>
                    {t("registerPage.agreeSuffix2") && ` ${t("registerPage.agreeSuffix2")}`}
                  </BodyText>
                }
                checked={f.form.acceptTerms}
                onChange={(e) => f.set("acceptTerms", e.currentTarget.checked)}
                error={f.checkError("acceptTerms")}
              />
              <DeclarationConsent
                declaration={f.declaration.declaration}
                accepted={f.form.acceptDeclaration}
                onChange={(accepted) => f.set("acceptDeclaration", accepted)}
                error={f.checkError("acceptDeclaration")}
              />
            </Stack>

            <Alert color="gray" variant="light" icon={<IconMail size={16} />}>
              {t("organizationRegistration.emailConfirmHint")}
            </Alert>

            {f.submitError && (
              <Alert color="red" icon={<IconAlertCircle size={16} />}>
                {f.submitError}
              </Alert>
            )}
            {f.showErrors && !f.canSubmit && !f.submitError && (
              <ErrorText>{t("organizationRegistration.errors.formIncomplete")}</ErrorText>
            )}

            <Stack gap="xs">
              <ActionButton type="submit" fullWidth loading={f.isSubmitting}>
                {t("organizationRegistration.submit")}
              </ActionButton>
              {onCancel && (
                <TextButton onClick={onCancel} disabled={f.isSubmitting}>
                  {t("organizationRegistration.switchToPersonal")}
                </TextButton>
              )}
            </Stack>
          </Stack>
        </form>
      </Card>
    </Container>
  );
}

import { useEffect, useMemo, useState } from "react";
import {
  Alert,
  Anchor,
  Box,
  Card,
  Checkbox,
  Container,
  Group,
  Loader,
  Stack,
  Text,
  TextInput,
  Title,
} from "@mantine/core";
import {
  IconAlertCircle,
  IconBuilding,
  IconInfoCircle,
  IconMail,
  IconUser,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { ActionButton, TextButton } from "@components/buttons";
import { SectionTitle, HelperText } from "@components/typography";
import { Api } from "@/api/generated";
import type { RoutesInstitutionDeclarationResponse } from "@/api/generated";
import { createAuthenticatedApiConfig, getApiConfig } from "@/api/client/http";
import { useAuth } from "@/providers/AuthProvider";
import { HELP_LINKS } from "@/config/helpLinks";
import { authLogger } from "@/config/logger";
import { DeclarationText } from "./DeclarationText";
import { YouthProtectionField } from "./YouthProtectionField";
import { organizationErrorMessage } from "../lib/errors";

export type OrganizationFormMode = "register" | "upgrade";

interface OrganizationRegistrationFormProps {
  mode: OrganizationFormMode;
  /** Prefill for new persons (from the Auth0 profile). */
  initialName?: string;
  initialEmail?: string;
  /** Called after the application was stored and the confirmation mail was sent. */
  onSubmitted: (email: string) => void;
  /** Register mode only: switch to the normal (personal) registration instead. */
  onCancel?: () => void;
}

interface FormState {
  accountName: string;
  personName: string;
  personFunction: string;
  email: string;
  institutionName: string;
  operator: string;
  street: string;
  postalCode: string;
  city: string;
  promptConstraint: string;
  confirmAdult: boolean;
  acceptTerms: boolean;
  acceptPrivacy: boolean;
  acceptDeclaration: boolean;
}

export function OrganizationRegistrationForm({
  mode,
  initialName = "",
  initialEmail = "",
  onSubmitted,
  onCancel,
}: OrganizationRegistrationFormProps) {
  const { t, i18n } = useTranslation("auth");
  const { getAccessToken } = useAuth();
  const isRegister = mode === "register";
  // The address of the login (or of the account, for upgrades) is fixed, so it can be
  // confirmed via Auth0 later. Only without one (dev login) can it be typed in.
  const emailLocked = initialEmail.trim() !== "";

  const [legal, setLegal] = useState<RoutesInstitutionDeclarationResponse | null>(null);
  const [legalError, setLegalError] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [showErrors, setShowErrors] = useState(false);

  const [form, setForm] = useState<FormState>({
    accountName: initialName,
    personName: "",
    personFunction: "",
    email: initialEmail,
    institutionName: "",
    operator: "",
    street: "",
    postalCode: "",
    city: "",
    promptConstraint: "",
    confirmAdult: false,
    acceptTerms: !isRegister,
    acceptPrivacy: !isRegister,
    acceptDeclaration: false,
  });

  const set = <K extends keyof FormState>(key: K, value: FormState[K]) =>
    setForm((f) => ({ ...f, [key]: value }));

  // Declaration, suggestions and the 13+ default come from one public endpoint.
  useEffect(() => {
    let cancelled = false;
    new Api(getApiConfig()).legal
      .institutionDeclarationList()
      .then((res) => {
        if (cancelled) return;
        setLegal(res.data);
        setForm((f) =>
          f.promptConstraint ? f : { ...f, promptConstraint: res.data.defaultPromptConstraint ?? "" },
        );
      })
      .catch((error) => {
        authLogger.error("Failed to load institution declaration", { error });
        if (!cancelled) setLegalError(true);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const maxLength = legal?.maxPromptConstraintLength ?? 200;
  const isGermanUi = (i18n.resolvedLanguage ?? i18n.language ?? "").startsWith("de");

  const missing = useMemo(() => {
    const required: (keyof FormState)[] = [
      "personName",
      "personFunction",
      "email",
      "institutionName",
      "operator",
      "street",
      "postalCode",
      "city",
      "promptConstraint",
    ];
    if (isRegister) required.push("accountName");
    return new Set(required.filter((k) => !String(form[k]).trim()));
  }, [form, isRegister]);

  const emailInvalid = form.email.trim() !== "" && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email.trim());
  const constraintTooLong = form.promptConstraint.trim().length > maxLength;
  const consentsMissing =
    !form.confirmAdult || !form.acceptTerms || !form.acceptPrivacy || !form.acceptDeclaration;
  const canSubmit = !!legal && missing.size === 0 && !emailInvalid && !constraintTooLong && !consentsMissing;

  const fieldError = (key: keyof FormState) =>
    showErrors && missing.has(key) ? t("organizationRegistration.errors.required") : undefined;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setShowErrors(true);
    if (!canSubmit || !legal?.declaration?.version) return;

    setIsSubmitting(true);
    setSubmitError(null);
    const details = {
      personName: form.personName.trim(),
      personFunction: form.personFunction.trim(),
      email: form.email.trim(),
      institutionName: form.institutionName.trim(),
      operator: form.operator.trim(),
      street: form.street.trim(),
      postalCode: form.postalCode.trim(),
      city: form.city.trim(),
      promptConstraint: form.promptConstraint.trim(),
      confirmAdult: form.confirmAdult,
      acceptDeclaration: form.acceptDeclaration,
      declarationVersion: legal.declaration.version,
    };
    try {
      const api = new Api(createAuthenticatedApiConfig(getAccessToken));
      if (isRegister) {
        await api.auth.registerInstitutionCreate({
          ...details,
          accountName: form.accountName.trim(),
          language: i18n.resolvedLanguage ?? i18n.language?.split("-")[0] ?? "en",
          acceptTerms: form.acceptTerms,
          acceptPrivacy: form.acceptPrivacy,
        });
      } else {
        await api.institutionApplications.institutionApplicationsCreate(details);
      }
      onSubmitted(details.email);
    } catch (error) {
      authLogger.error("Organisation application failed", { error });
      setSubmitError(organizationErrorMessage(error, t));
    } finally {
      setIsSubmitting(false);
    }
  };

  if (legalError) {
    return (
      <Container size="sm" py="xl">
        <Alert color="red" icon={<IconAlertCircle size={16} />}>
          {t("organizationRegistration.errors.loadFailed")}
        </Alert>
      </Container>
    );
  }

  if (!legal) {
    return (
      <Container size="sm" py="xl">
        <Group justify="center">
          <Loader />
        </Group>
      </Container>
    );
  }

  return (
    <Container size="sm" py={{ base: "md", sm: "xl" }}>
      <Card shadow="md" padding="xl" radius="md" withBorder>
        <form onSubmit={handleSubmit} noValidate>
          <Stack gap="lg">
            <Box ta="center">
              <SectionTitle>
                {isRegister
                  ? t("organizationRegistration.titleRegister")
                  : t("organizationRegistration.titleUpgrade")}
              </SectionTitle>
              <HelperText>{t("organizationRegistration.subtitle")}</HelperText>
            </Box>

            {/* Person */}
            <Stack gap="sm">
              <Title order={4}>{t("organizationRegistration.personSection")}</Title>
              {isRegister && (
                <TextInput
                  label={t("organizationRegistration.accountNameLabel")}
                  description={t("organizationRegistration.accountNameDescription")}
                  leftSection={<IconUser size={16} />}
                  maxLength={24}
                  value={form.accountName}
                  onChange={(e) => set("accountName", e.currentTarget.value)}
                  error={fieldError("accountName")}
                  required
                />
              )}
              <TextInput
                label={t("organizationRegistration.personNameLabel")}
                value={form.personName}
                maxLength={100}
                onChange={(e) => set("personName", e.currentTarget.value)}
                error={fieldError("personName")}
                autoComplete="name"
                required
              />
              <TextInput
                label={t("organizationRegistration.personFunctionLabel")}
                placeholder={t("organizationRegistration.personFunctionPlaceholder")}
                value={form.personFunction}
                maxLength={100}
                onChange={(e) => set("personFunction", e.currentTarget.value)}
                error={fieldError("personFunction")}
                required
              />
              <TextInput
                type="email"
                label={t("organizationRegistration.emailLabel")}
                description={
                  emailLocked
                    ? isRegister
                      ? t("organizationRegistration.emailFromLogin")
                      : t("organizationRegistration.emailFromAccount")
                    : t("organizationRegistration.emailDescription")
                }
                leftSection={<IconMail size={16} />}
                value={form.email}
                maxLength={200}
                readOnly={emailLocked}
                variant={emailLocked ? "filled" : "default"}
                onChange={(e) => set("email", e.currentTarget.value)}
                error={fieldError("email") ?? (emailInvalid ? t("organizationRegistration.errors.emailInvalid") : undefined)}
                autoComplete="email"
                required
              />
            </Stack>

            {/* Organisation */}
            <Stack gap="sm">
              <Title order={4}>{t("organizationRegistration.institutionSection")}</Title>
              <TextInput
                label={t("organizationRegistration.institutionNameLabel")}
                leftSection={<IconBuilding size={16} />}
                value={form.institutionName}
                maxLength={150}
                onChange={(e) => set("institutionName", e.currentTarget.value)}
                error={fieldError("institutionName")}
                autoComplete="organization"
                required
              />
              <TextInput
                label={t("organizationRegistration.operatorLabel")}
                description={t("organizationRegistration.operatorDescription")}
                value={form.operator}
                maxLength={150}
                onChange={(e) => set("operator", e.currentTarget.value)}
                error={fieldError("operator")}
                required
              />
              <TextInput
                label={t("organizationRegistration.streetLabel")}
                value={form.street}
                maxLength={150}
                onChange={(e) => set("street", e.currentTarget.value)}
                error={fieldError("street")}
                autoComplete="street-address"
                required
              />
              <Group grow align="flex-start">
                <TextInput
                  label={t("organizationRegistration.postalCodeLabel")}
                  value={form.postalCode}
                  maxLength={20}
                  onChange={(e) => set("postalCode", e.currentTarget.value)}
                  error={fieldError("postalCode")}
                  autoComplete="postal-code"
                  required
                  style={{ maxWidth: 140 }}
                />
                <TextInput
                  label={t("organizationRegistration.cityLabel")}
                  value={form.city}
                  maxLength={150}
                  onChange={(e) => set("city", e.currentTarget.value)}
                  error={fieldError("city")}
                  autoComplete="address-level2"
                  required
                />
              </Group>
            </Stack>

            <YouthProtectionField
              legal={legal}
              value={form.promptConstraint}
              onChange={(v) => set("promptConstraint", v)}
              showErrors={showErrors}
            />

            {/* Consents */}
            <Stack gap="sm">
              <Title order={4}>{t("organizationRegistration.consentSection")}</Title>
              <Checkbox
                label={t("organizationRegistration.confirmAdult")}
                checked={form.confirmAdult}
                onChange={(e) => set("confirmAdult", e.currentTarget.checked)}
                error={showErrors && !form.confirmAdult ? t("organizationRegistration.errors.required") : undefined}
              />
              {isRegister && (
                <>
                  <Checkbox
                    label={
                      <>
                        {t("organizationRegistration.acceptTermsPrefix")}{" "}
                        <Anchor href={HELP_LINKS.TERMS_OF_SERVICE} target="_blank" rel="noopener noreferrer">
                          {t("registerPage.termsOfService")}
                        </Anchor>
                        {t("organizationRegistration.acceptTermsSuffix")}
                      </>
                    }
                    checked={form.acceptTerms}
                    onChange={(e) => set("acceptTerms", e.currentTarget.checked)}
                    error={showErrors && !form.acceptTerms ? t("organizationRegistration.errors.required") : undefined}
                  />
                  <Checkbox
                    label={
                      <>
                        {t("organizationRegistration.acceptPrivacyPrefix")}{" "}
                        <Anchor href={HELP_LINKS.PRIVACY_POLICY} target="_blank" rel="noopener noreferrer">
                          {t("registerPage.privacyPolicy")}
                        </Anchor>
                        {t("organizationRegistration.acceptPrivacySuffix")}
                      </>
                    }
                    checked={form.acceptPrivacy}
                    onChange={(e) => set("acceptPrivacy", e.currentTarget.checked)}
                    error={showErrors && !form.acceptPrivacy ? t("organizationRegistration.errors.required") : undefined}
                  />
                </>
              )}

              {!isGermanUi && (
                <Alert color="gray" variant="light" icon={<IconInfoCircle size={16} />}>
                  {t("organizationRegistration.declarationGermanOnly")}
                </Alert>
              )}
              <DeclarationText text={legal.declaration?.text ?? ""} />
              <HelperText>
                {t("organizationRegistration.declarationVersion", {
                  version: legal.declaration?.version,
                  date: legal.declaration?.date
                    ? new Date(legal.declaration.date).toLocaleDateString(i18n.language)
                    : "",
                })}
              </HelperText>
              {/* Binding wording agreed with the JFF legal counsel; shown in German in every UI language. */}
              <Checkbox
                lang="de"
                label="Ich habe die Einverständniserklärung gelesen und stimme ihr im Namen meiner Einrichtung zu. Ich bestätige, dass ich berechtigt bin, diese Erklärung für meine Einrichtung abzugeben."
                checked={form.acceptDeclaration}
                onChange={(e) => set("acceptDeclaration", e.currentTarget.checked)}
                error={showErrors && !form.acceptDeclaration ? t("organizationRegistration.errors.required") : undefined}
              />
            </Stack>

            <Alert color="gray" variant="light" icon={<IconMail size={16} />}>
              {t("organizationRegistration.emailConfirmHint")}
            </Alert>

            {submitError && (
              <Alert color="red" icon={<IconAlertCircle size={16} />}>
                {submitError}
              </Alert>
            )}
            {showErrors && !canSubmit && !submitError && (
              <Text c="red" size="sm" ta="center">
                {t("organizationRegistration.errors.formIncomplete")}
              </Text>
            )}

            <Stack gap="xs">
              <ActionButton type="submit" fullWidth loading={isSubmitting}>
                {t("organizationRegistration.submit")}
              </ActionButton>
              {onCancel && (
                <TextButton onClick={onCancel} disabled={isSubmitting}>
                  {isRegister
                    ? t("organizationRegistration.switchToPersonal")
                    : t("organizationRegistration.cancel")}
                </TextButton>
              )}
            </Stack>
          </Stack>
        </form>
      </Card>
    </Container>
  );
}

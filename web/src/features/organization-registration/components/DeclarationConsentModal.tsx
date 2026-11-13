import { useEffect, useState } from "react";
import {
  Alert,
  Checkbox,
  Group,
  Loader,
  Modal,
  Stack,
  Text,
  TextInput,
  Title,
} from "@mantine/core";
import { IconAlertCircle, IconInfoCircle } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { ActionButton, TextButton } from "@components/buttons";
import { Api } from "@/api/generated";
import type { RoutesInstitutionDeclarationResponse } from "@/api/generated";
import { getApiConfig } from "@/api/client/http";
import { useAuthenticatedApi } from "@/api/useAuthenticatedApi";
import { useAuth } from "@/providers/AuthProvider";
import { DeclarationText } from "./DeclarationText";
import { organizationErrorMessage } from "../lib/errors";

const DISMISS_KEY = "cgl_declaration_consent_later";

/**
 * Asks an organisation head to consent to the current declaration when their organisation
 * has none yet (organisations from before self-registration) or a new version was published.
 * "Später" hides it for this browser session; the organisation stays usable.
 */
export function DeclarationConsentModal() {
  const { t, i18n } = useTranslation("auth");
  const { backendUser, refreshBackendUser } = useAuth();
  const api = useAuthenticatedApi();

  const status = backendUser?.institutionDeclaration;
  const institution = backendUser?.role?.institution;
  const version = status?.currentVersion ?? "";
  const shouldAsk =
    !!status?.pending && !!institution?.id && backendUser?.role?.role === "head";

  const [dismissedVersion, setDismissedVersion] = useState<string | null>(() => {
    try {
      return sessionStorage.getItem(DISMISS_KEY);
    } catch {
      return null;
    }
  });
  const [submitted, setSubmitted] = useState(false);
  const opened = shouldAsk && !submitted && dismissedVersion !== version;
  const [legal, setLegal] = useState<RoutesInstitutionDeclarationResponse | null>(null);
  const [form, setForm] = useState({
    personName: "",
    personFunction: "",
    email: "",
    operator: "",
    street: "",
    postalCode: "",
    city: "",
  });
  const [accepted, setAccepted] = useState(false);
  const [showErrors, setShowErrors] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Load the declaration and prefill what the organisation already has.
  useEffect(() => {
    if (!opened || !api || !institution?.id) return;
    let cancelled = false;
    new Api(getApiConfig()).legal.institutionDeclarationList().then((res) => {
      if (!cancelled) setLegal(res.data);
    });
    api.institutions.institutionsDetail(institution.id).then((res) => {
      if (cancelled) return;
      const inst = res.data;
      setForm((f) => ({
        ...f,
        email: f.email || backendUser?.email || "",
        operator: f.operator || inst.operator || "",
        street: f.street || inst.street || "",
        postalCode: f.postalCode || inst.postalCode || "",
        city: f.city || inst.city || "",
      }));
    });
    return () => {
      cancelled = true;
    };
  }, [opened, api, institution?.id, backendUser?.email]);

  if (!shouldAsk || !institution?.id) return null;

  const set = (key: keyof typeof form, value: string) => setForm((f) => ({ ...f, [key]: value }));
  const missing = (Object.keys(form) as (keyof typeof form)[]).filter((k) => !form[k].trim());
  const err = (key: keyof typeof form) =>
    showErrors && missing.includes(key) ? t("organizationRegistration.errors.required") : undefined;
  const isGermanUi = (i18n.resolvedLanguage ?? i18n.language ?? "").startsWith("de");

  const later = () => {
    try {
      sessionStorage.setItem(DISMISS_KEY, version);
    } catch {
      // ignore
    }
    setDismissedVersion(version);
  };

  const submit = async () => {
    setShowErrors(true);
    if (!api || !legal?.declaration?.version || missing.length > 0 || !accepted) return;
    setSaving(true);
    setError(null);
    try {
      await api.institutions.declarationConsentCreate(institution.id!, {
        ...Object.fromEntries(Object.entries(form).map(([k, v]) => [k, v.trim()])),
        acceptDeclaration: true,
        declarationVersion: legal.declaration.version,
      });
      setSubmitted(true);
      await refreshBackendUser();
    } catch (e) {
      setError(organizationErrorMessage(e, t));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      opened={opened}
      onClose={later}
      size="lg"
      title={<Title order={3}>{t("organizationRegistration.consentModal.title")}</Title>}
    >
      {!legal ? (
        <Group justify="center" py="lg">
          <Loader />
        </Group>
      ) : (
        <Stack gap="md">
          <Text size="sm">
            {status?.consentedVersion
              ? t("organizationRegistration.consentModal.introNewVersion", {
                  name: institution.name,
                  version: legal.declaration?.version,
                })
              : t("organizationRegistration.consentModal.introExisting", { name: institution.name })}
          </Text>
          <Text size="sm" c="dimmed">
            {t("organizationRegistration.consentModal.oneHeadIsEnough")}
          </Text>

          <Title order={5}>{t("organizationRegistration.personSection")}</Title>
          <TextInput label={t("organizationRegistration.personNameLabel")} value={form.personName}
            onChange={(e) => set("personName", e.currentTarget.value)} error={err("personName")} maxLength={100} required />
          <TextInput label={t("organizationRegistration.personFunctionLabel")}
            placeholder={t("organizationRegistration.personFunctionPlaceholder")} value={form.personFunction}
            onChange={(e) => set("personFunction", e.currentTarget.value)} error={err("personFunction")} maxLength={100} required />
          <TextInput type="email" label={t("organizationRegistration.emailLabel")}
            description={t("organizationRegistration.consentModal.emailDescription")} value={form.email}
            onChange={(e) => set("email", e.currentTarget.value)} error={err("email")} maxLength={200} required />

          <Title order={5}>{t("organizationRegistration.institutionSection")}</Title>
          <TextInput label={t("organizationRegistration.operatorLabel")}
            description={t("organizationRegistration.operatorDescription")} value={form.operator}
            onChange={(e) => set("operator", e.currentTarget.value)} error={err("operator")} maxLength={150} required />
          <TextInput label={t("organizationRegistration.streetLabel")} value={form.street}
            onChange={(e) => set("street", e.currentTarget.value)} error={err("street")} maxLength={150} required />
          <Group grow align="flex-start">
            <TextInput label={t("organizationRegistration.postalCodeLabel")} value={form.postalCode}
              onChange={(e) => set("postalCode", e.currentTarget.value)} error={err("postalCode")} maxLength={20} required
              style={{ maxWidth: 140 }} />
            <TextInput label={t("organizationRegistration.cityLabel")} value={form.city}
              onChange={(e) => set("city", e.currentTarget.value)} error={err("city")} maxLength={150} required />
          </Group>

          {!isGermanUi && (
            <Alert color="gray" variant="light" icon={<IconInfoCircle size={16} />}>
              {t("organizationRegistration.declarationGermanOnly")}
            </Alert>
          )}
          <DeclarationText text={legal.declaration?.text ?? ""} />
          <Text size="xs" c="dimmed">
            {t("organizationRegistration.declarationVersion", {
              version: legal.declaration?.version,
              date: legal.declaration?.date ? new Date(legal.declaration.date).toLocaleDateString(i18n.language) : "",
            })}
          </Text>
          {/* Binding wording agreed with the JFF legal counsel; shown in German in every UI language. */}
          <Checkbox
            lang="de"
            label="Ich habe die Einverständniserklärung gelesen und stimme ihr im Namen meiner Einrichtung zu. Ich bestätige, dass ich berechtigt bin, diese Erklärung für meine Einrichtung abzugeben."
            checked={accepted}
            onChange={(e) => setAccepted(e.currentTarget.checked)}
            error={showErrors && !accepted ? t("organizationRegistration.errors.required") : undefined}
          />

          {error && (
            <Alert color="red" icon={<IconAlertCircle size={16} />}>
              {error}
            </Alert>
          )}
          <Group justify="space-between">
            <TextButton onClick={later} disabled={saving}>
              {t("organizationRegistration.consentModal.later")}
            </TextButton>
            <ActionButton onClick={submit} loading={saving}>
              {t("organizationRegistration.consentModal.submit")}
            </ActionButton>
          </Group>
        </Stack>
      )}
    </Modal>
  );
}

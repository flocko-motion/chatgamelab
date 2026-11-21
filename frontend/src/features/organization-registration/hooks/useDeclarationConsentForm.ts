import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useConsentToDeclaration, useDeclaration, useOrganisationConsent } from "@/api/hooks";
import { useAuth } from "@/providers/AuthProvider";
import { organizationErrorMessage } from "../lib/errors";

/** The person and the organisation's details a consent confirms. */
export interface ConsentForm {
  personName: string;
  personFunction: string;
  email: string;
  operator: string;
  street: string;
  postalCode: string;
  city: string;
}

/**
 * A head's consent to the current declaration for their organisation, which
 * lifts its suspension for want of a consent. The form starts with what the
 * organisation has: its details and its restriction (or the 13+ default).
 */
export function useDeclarationConsentForm() {
  const { t } = useTranslation("auth");
  const { backendUser, refreshCaller } = useAuth();
  const institutionId = backendUser?.role?.institution?.id;
  const declaration = useDeclaration();
  const organisation = useOrganisationConsent(institutionId);
  const consentToDeclaration = useConsentToDeclaration();

  // What the head typed; the rest starts with what the organisation has.
  const [edits, setEdits] = useState<Partial<ConsentForm>>({});
  const [constraintEdit, setConstraint] = useState<string | null>(null);
  const [accepted, setAccepted] = useState(false);
  const [showErrors, setShowErrors] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const org = organisation.data;
  const form: ConsentForm = {
    personName: "",
    personFunction: "",
    email: backendUser?.email ?? "",
    operator: org?.operator ?? "",
    street: org?.street ?? "",
    postalCode: org?.postalCode ?? "",
    city: org?.city ?? "",
    ...edits,
  };
  // null until the organisation loaded: then its restriction, or the 13+ default.
  const constraint =
    org && declaration.data
      ? (constraintEdit ?? org.promptConstraints ?? declaration.data.defaultPromptConstraint ?? "")
      : null;

  const set = (key: keyof ConsentForm, value: string) => setEdits((e) => ({ ...e, [key]: value }));
  const missing = (Object.keys(form) as (keyof ConsentForm)[]).filter((key) => !form[key].trim());
  const maxConstraint = declaration.data?.maxPromptConstraintLength ?? 200;
  const constraintValue = constraint ?? "";
  const constraintInvalid = !constraintValue.trim() || constraintValue.trim().length > maxConstraint;

  /** The error a field shows once the head tried to agree. */
  const fieldError = (key: keyof ConsentForm) =>
    showErrors && missing.includes(key) ? t("organizationRegistration.errors.required") : undefined;

  const submit = async () => {
    setShowErrors(true);
    const declarationVersion = declaration.data?.declaration.version;
    if (!institutionId || !declarationVersion || missing.length > 0 || constraintInvalid || !accepted) return;
    setError(null);
    try {
      await consentToDeclaration.mutateAsync({
        institutionId,
        consent: {
          personName: form.personName.trim(),
          personFunction: form.personFunction.trim(),
          email: form.email.trim(),
          operator: form.operator.trim(),
          street: form.street.trim(),
          postalCode: form.postalCode.trim(),
          city: form.city.trim(),
          promptConstraint: constraintValue.trim(),
          acceptDeclaration: true,
          declarationVersion,
        },
      });
      await refreshCaller();
    } catch (e) {
      setError(organizationErrorMessage(e, t));
    }
  };

  return {
    loadFailed: declaration.isError || organisation.isError,
    organisationName: org?.name ?? "",
    status: org?.declaration,
    declaration: declaration.data,
    form,
    set,
    fieldError,
    constraint,
    setConstraint,
    accepted,
    setAccepted,
    showErrors,
    saving: consentToDeclaration.isPending,
    error,
    submit,
  };
}

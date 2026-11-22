import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useDeclaration } from "@/api/hooks";
import type { OrganisationSignUpReq } from "@/api/types";
import { useAuth } from "@/providers/AuthProvider";
import { AgeGroup } from "@/constants/ageGroup";
import { authLogger } from "@/config/logger";
import { organizationErrorMessage } from "../lib/errors";

/** What the organisation's form holds. */
export interface OrganisationFormState {
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
  acceptDeclaration: boolean;
}

/** The text fields the organisation's form requires. */
const REQUIRED: (keyof OrganisationFormState)[] = [
  "accountName",
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

const EMAIL = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

interface Options {
  /** Suggested display name of the new account. */
  initialName?: string;
  /** The sign-in's email; fixed when given. */
  initialEmail?: string;
  /** Called once the organisation exists and the caller is its head. */
  onSubmitted: () => void;
}

/**
 * The organisation's form: its state, what is missing, and sending it, which
 * completes the new account's registration with the organisation.
 */
export function useOrganisationForm({ initialName = "", initialEmail = "", onSubmitted }: Options) {
  const { t } = useTranslation("auth");
  const { register } = useAuth();
  const declaration = useDeclaration();

  const [fields, setFields] = useState<OrganisationFormState>({
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
    acceptTerms: false,
    acceptDeclaration: false,
  });
  const [showErrors, setShowErrors] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  // The restriction starts as the site's for 13+, until the person changes it.
  const [constraintEdited, setConstraintEdited] = useState(false);
  const form: OrganisationFormState = constraintEdited
    ? fields
    : { ...fields, promptConstraint: declaration.data?.defaultPromptConstraint ?? "" };

  const set = <K extends keyof OrganisationFormState>(key: K, value: OrganisationFormState[K]) => {
    if (key === "promptConstraint") setConstraintEdited(true);
    setFields((f) => ({ ...f, [key]: value }));
  };

  const missing = new Set(REQUIRED.filter((key) => !String(form[key]).trim()));

  const maxConstraintLength = declaration.data?.maxPromptConstraintLength ?? 200;
  const emailInvalid = form.email.trim() !== "" && !EMAIL.test(form.email.trim());
  const constraintTooLong = form.promptConstraint.trim().length > maxConstraintLength;
  const consentsMissing = !form.confirmAdult || !form.acceptTerms || !form.acceptDeclaration;
  const canSubmit =
    !!declaration.data && missing.size === 0 && !emailInvalid && !constraintTooLong && !consentsMissing;

  /** The error a field shows once the person tried to send the form. */
  const fieldError = (key: keyof OrganisationFormState) =>
    showErrors && missing.has(key) ? t("organizationRegistration.errors.required") : undefined;

  /** The error an unticked box shows once the person tried to send the form. */
  const checkError = (key: "confirmAdult" | "acceptTerms" | "acceptDeclaration") =>
    showErrors && !form[key] ? t("organizationRegistration.errors.required") : undefined;

  const signUp = (version: string): OrganisationSignUpReq => ({
    personName: form.personName.trim(),
    personFunction: form.personFunction.trim(),
    institutionName: form.institutionName.trim(),
    operator: form.operator.trim(),
    street: form.street.trim(),
    postalCode: form.postalCode.trim(),
    city: form.city.trim(),
    promptConstraint: form.promptConstraint.trim(),
    acceptDeclaration: form.acceptDeclaration,
    declarationVersion: version,
  });

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setShowErrors(true);
    const version = declaration.data?.declaration.version;
    if (!canSubmit || !version) return;

    setIsSubmitting(true);
    setSubmitError(null);
    try {
      await register({
        name: form.accountName.trim(),
        email: form.email.trim(),
        ageGroup: AgeGroup.U18, // confirmed with the 18+ box
        organisation: signUp(version),
        acceptTerms: form.acceptTerms,
      });
      onSubmitted();
    } catch (error) {
      authLogger.error("Registering an organisation failed", { error });
      setSubmitError(organizationErrorMessage(error, t));
    } finally {
      setIsSubmitting(false);
    }
  };

  return {
    declaration: declaration.data,
    loadFailed: declaration.isError,
    form,
    set,
    fieldError,
    checkError,
    emailLocked: initialEmail.trim() !== "",
    emailInvalid,
    showErrors,
    canSubmit,
    isSubmitting,
    submitError,
    submit,
  };
}

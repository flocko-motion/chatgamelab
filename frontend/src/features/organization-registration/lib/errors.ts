import type { TFunction } from "i18next";
import { ErrorCodes, extractRawErrorCode } from "@/common/types/errorCodes";

/** The codes of the organisation's forms that have a text of their own. */
const KNOWN: ReadonlySet<string> = new Set([
  ErrorCodes.INSTITUTION_AGE_NOT_ALLOWED,
  ErrorCodes.INSTITUTION_NAME_TAKEN,
  ErrorCodes.EMAIL_BELONGS_TO_OTHER_ACCOUNT,
  ErrorCodes.DECLARATION_VERSION_OUTDATED,
  ErrorCodes.INSTITUTION_CONSENTS_REQUIRED,
  ErrorCodes.DUPLICATE_NAME,
  ErrorCodes.PROFANE_NAME,
  ErrorCodes.NAME_TOO_LONG,
  ErrorCodes.VALIDATION,
  ErrorCodes.CONFLICT,
]);

/** A refusal of the organisation's forms, as a translated message. */
export function organizationErrorMessage(error: unknown, t: TFunction): string {
  const code = extractRawErrorCode(error);
  if (code && KNOWN.has(code)) {
    return t(`organizationRegistration.errors.${code}`);
  }
  return t("organizationRegistration.errors.generic");
}

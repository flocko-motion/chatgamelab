import type { TFunction } from "i18next";
import { extractRawErrorCode } from "@/common/types/errorCodes";

const KNOWN = new Set([
  "institution_age_not_allowed",
  "already_in_institution",
  "institution_name_taken",
  "email_belongs_to_other_account",
  "application_token_invalid",
  "application_expired",
  "declaration_version_outdated",
  "institution_consents_required",
  "duplicate_name",
  "profane_name",
  "name_too_long",
  "validation_error",
  "conflict",
]);

/** Maps a backend error of the organisation flow to a translated message. */
export function organizationErrorMessage(error: unknown, t: TFunction): string {
  const code = extractRawErrorCode(error);
  if (code && KNOWN.has(code)) {
    return t(`organizationRegistration.errors.${code}`);
  }
  return t("organizationRegistration.errors.generic");
}

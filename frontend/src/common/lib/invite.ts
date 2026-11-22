import type { TFunction } from "i18next";
import type { InviteView, UserRoleInvite } from "@/api/types";
import { translateError } from "@/common/lib/errorHelpers";
import { extractRawErrorCode } from "@/common/types/errorCodes";

/** Pending open invite that still admits participants; mirrors db.CreateWorkshopInvite. */
export function isUsableInvite(invite: UserRoleInvite | InviteView): boolean {
  if (invite.status !== "pending" || !invite.inviteToken) return false;
  if (invite.expiresAt && new Date(invite.expiresAt) < new Date()) return false;
  if (invite.maxUses && (invite.usesCount ?? 0) >= invite.maxUses) return false;
  return true;
}

/**
 * What to tell someone whose invite was refused: a pending invite for the
 * same person answers `conflict`; anything else is the error's own text.
 */
export function inviteErrorMessage(t: TFunction, error: unknown): string {
  if (extractRawErrorCode(error) === "conflict") return t("myOrganization.inviteAlreadyExists");
  return translateError(error).message || t("myOrganization.inviteError");
}

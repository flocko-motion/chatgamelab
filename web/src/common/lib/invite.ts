import type { ObjUserRoleInvite } from "@/api/generated";

/** Pending open invite that still admits participants; mirrors db.CreateWorkshopInvite. */
export function isUsableInvite(invite: ObjUserRoleInvite): boolean {
  if (invite.status !== "pending" || !invite.inviteToken) return false;
  if (invite.expiresAt && new Date(invite.expiresAt) < new Date()) return false;
  if (invite.maxUses && (invite.usesCount ?? 0) >= invite.maxUses) return false;
  return true;
}

import { useMembers, useRemoveMember } from "@/api/hooks";
import type { UserView } from "@/api/types";
import { MEMBER_LIST_PRESET } from "../lib/memberListPresets";

/**
 * An organisation's members, for the admins' organisation page: the
 * embedded list and removing a member.
 */
export function useOrganizationMembers(institutionId: string) {
  const list = useMembers(institutionId, MEMBER_LIST_PRESET.list);
  const removeMember = useRemoveMember();

  return {
    list,
    remove: (member: UserView) => removeMember.mutate({ institutionId, userId: member.id }),
    removing: removeMember.isPending,
  };
}

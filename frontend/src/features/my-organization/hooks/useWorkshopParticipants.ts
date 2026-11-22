import { useState } from "react";
import {
  useParticipants,
  useRemoveParticipant,
  useUpdateParticipant,
  useWorkshop,
} from "@/api/hooks";
import type { WorkshopParticipantView } from "@/api/types";
import { RoleName } from "@/common/lib/roles";
import { PARTICIPANT_LIST_PRESET } from "../lib/participantListPresets";

/**
 * A workshop's participants, for ParticipantsList to render: the list, the
 * participant whose name is being edited, and the modals for a Guest's
 * re-login link and for removing a participant.
 */
export function useWorkshopParticipants(workshopId: string) {
  const list = useParticipants(workshopId, PARTICIPANT_LIST_PRESET);
  // Whether the caller may manage who is in it: the server's flag.
  const { data: workshop } = useWorkshop(workshopId);
  const mayManage = !!workshop?.mayManageParticipants;
  const updateParticipant = useUpdateParticipant();
  const removeParticipant = useRemoveParticipant();

  // ── Editing a name in place ─────────────────────────────────────────────
  const [editing, setEditing] = useState<WorkshopParticipantView | null>(null);
  const [newName, setNewName] = useState("");

  const startEdit = (participant: WorkshopParticipantView) => {
    setEditing(participant);
    setNewName(participant.name || "");
  };
  const cancelEdit = () => {
    setEditing(null);
    setNewName("");
  };
  const saveEdit = async () => {
    if (!editing?.id || !newName.trim()) return;
    await updateParticipant.mutateAsync({ participantId: editing.id, name: newName.trim() });
    cancelEdit();
  };

  // ── Re-login link and removing ──────────────────────────────────────────
  const [linkGuest, setLinkGuest] = useState<WorkshopParticipantView | null>(null);
  const [removing, setRemoving] = useState<WorkshopParticipantView | null>(null);

  const confirmRemove = async () => {
    if (!removing?.id) return;
    await removeParticipant.mutateAsync({
      participantId: removing.id,
      workshopId,
      permanent: removing.permanent !== false,
    });
    setRemoving(null);
  };

  return {
    list,
    /** Only a Guest has a re-login link. */
    hasGuestLink: (p: WorkshopParticipantView) => p.role === RoleName.Guest,
    /** Who may manage the workshop's participants removes any of them. */
    canRemove: () => mayManage,
    edit: {
      participantId: editing?.id ?? null,
      name: newName,
      setName: setNewName,
      start: startEdit,
      cancel: cancelEdit,
      save: saveEdit,
      saving: updateParticipant.isPending,
    },
    showGuestLink: setLinkGuest,
    guestLinkModal: {
      guest: linkGuest,
      close: () => setLinkGuest(null),
    },
    requestRemove: setRemoving,
    removeModal: {
      participant: removing,
      close: () => setRemoving(null),
      confirm: confirmRemove,
      loading: removeParticipant.isPending,
    },
  };
}

export type WorkshopParticipantsState = ReturnType<typeof useWorkshopParticipants>;

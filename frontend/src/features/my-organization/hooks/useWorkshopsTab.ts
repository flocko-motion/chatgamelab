import { useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import {
  useCreateWorkshop,
  useDeleteWorkshop,
  useUpdateWorkshop,
  useWorkshopList,
} from "@/api/hooks";
import type { WorkshopView } from "@/api/types";
import { isListNarrowed } from "@/common/lib/listNarrowed";
import { ROUTES } from "@/common/routes/routes";
import { useWorkshopMode } from "@/providers/WorkshopModeProvider";
import { WORKSHOP_LIST_PRESET } from "../lib/workshopListPresets";
import { useWorkshopInvites } from "./useWorkshopInvites";

export interface UseWorkshopsTabOptions {
  institutionId: string;
  /** Open the create modal on mount. */
  autoCreate?: boolean;
}

/**
 * Everything the workshops tab does, for WorkshopsTab to render: the
 * workshop list, entering a workshop, and the modals for creating, deleting,
 * the invites and a workshop's settings.
 */
export function useWorkshopsTab({ institutionId, autoCreate }: UseWorkshopsTabOptions) {
  const navigate = useNavigate();
  const { enterWorkshopMode } = useWorkshopMode();

  // ── The list ────────────────────────────────────────────────────────────
  const list = useWorkshopList(institutionId, WORKSHOP_LIST_PRESET);

  const createWorkshop = useCreateWorkshop();
  const updateWorkshop = useUpdateWorkshop();
  const deleteWorkshop = useDeleteWorkshop();
  const invites = useWorkshopInvites();

  // ── Entering and (de)activating ─────────────────────────────────────────
  const enter = async (workshop: WorkshopView) => {
    if (!workshop.id || !workshop.name) return;
    await enterWorkshopMode(workshop.id, workshop.name);
    navigate({ to: ROUTES.MY_WORKSHOP as "/" });
  };

  const toggleActive = async (workshop: WorkshopView) => {
    await updateWorkshop.mutateAsync({
      id: workshop.id,
      name: workshop.name || "",
      active: !workshop.active,
      public: workshop.public,
    });
  };

  // ── Creating ────────────────────────────────────────────────────────────
  const [createOpened, setCreateOpened] = useState(!!autoCreate);
  const [newName, setNewName] = useState("");
  const [newActive, setNewActive] = useState(true);

  // Asked for again later (a new link to the page): open it again.
  const [seenAutoCreate, setSeenAutoCreate] = useState(autoCreate);
  if (autoCreate !== seenAutoCreate) {
    setSeenAutoCreate(autoCreate);
    if (autoCreate) setCreateOpened(true);
  }

  const submitCreate = async () => {
    if (!newName.trim()) return;
    await createWorkshop.mutateAsync({
      name: newName.trim(),
      institutionId,
      active: newActive,
      public: false,
    });
    setNewName("");
    setNewActive(true);
    setCreateOpened(false);
  };

  // ── Deleting ────────────────────────────────────────────────────────────
  const [deleting, setDeleting] = useState<WorkshopView | null>(null);
  const confirmDelete = async () => {
    if (!deleting) return;
    await deleteWorkshop.mutateAsync(deleting.id);
    setDeleting(null);
  };

  // ── A workshop's settings ───────────────────────────────────────────────
  const [settingsId, setSettingsId] = useState<string | null>(null);

  return {
    list,
    /** Narrowed by a search or a toggle, so empty means no match. */
    narrowed: isListNarrowed(list),
    enter,
    toggleActive,
    toggling: updateWorkshop.isPending,
    invites,
    requestDelete: setDeleting,
    openSettings: (workshop: WorkshopView) => setSettingsId(workshop.id),
    createModal: {
      opened: createOpened,
      open: () => setCreateOpened(true),
      close: () => setCreateOpened(false),
      name: newName,
      setName: setNewName,
      active: newActive,
      setActive: setNewActive,
      submit: submitCreate,
      loading: createWorkshop.isPending,
    },
    deleteModal: {
      workshop: deleting,
      close: () => setDeleting(null),
      confirm: confirmDelete,
      loading: deleteWorkshop.isPending,
    },
    settingsModal: {
      workshopId: settingsId,
      close: () => setSettingsId(null),
    },
  };
}

export type WorkshopsTabState = ReturnType<typeof useWorkshopsTab>;

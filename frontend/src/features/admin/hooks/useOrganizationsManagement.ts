import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  useCreateOrganisation,
  useCreateOrganisationInvite,
  useDeleteOrganisation,
  useOrganisations,
  useUpdateOrganisation,
} from "@/api/hooks";
import type { OrganisationView } from "@/api/types";
import { inviteErrorMessage } from "@/common/lib/invite";
import { ORGANISATION_LIST_PRESET } from "../lib/organisationListPresets";

/** An organisation's operator and address, as an admin enters them. */
const NO_DETAILS = { operator: "", street: "", postalCode: "", city: "" };
export type OrganisationDetails = typeof NO_DETAILS;

/**
 * Everything the admins' organisation list does, for OrganizationsManagement
 * to render: the list (the server filters, sorts and pages), what may be done
 * with each organisation (the server's flags), and the dialogs for creating,
 * renaming, inviting a head, suspending and deleting.
 */
export function useOrganizationsManagement() {
  const { t } = useTranslation("common");
  const list = useOrganisations(ORGANISATION_LIST_PRESET.list);

  // ── Creating ────────────────────────────────────────────────────────────
  const createOrganisation = useCreateOrganisation();
  const [creating, setCreating] = useState(false);
  const [newName, setNewName] = useState("");
  const [newDetails, setNewDetails] = useState<OrganisationDetails>(NO_DETAILS);
  const create = () =>
    createOrganisation.mutate(
      {
        name: newName,
        operator: newDetails.operator.trim(),
        street: newDetails.street.trim(),
        postalCode: newDetails.postalCode.trim(),
        city: newDetails.city.trim(),
      },
      {
        onSuccess: () => {
          setCreating(false);
          setNewName("");
          setNewDetails(NO_DETAILS);
        },
      },
    );

  // ── Renaming ────────────────────────────────────────────────────────────
  const updateOrganisation = useUpdateOrganisation();
  const [editing, setEditing] = useState<OrganisationView | null>(null);
  const [editName, setEditName] = useState("");
  const rename = () =>
    editing &&
    updateOrganisation.mutate(
      { id: editing.id, name: editName },
      {
        onSuccess: () => {
          setEditing(null);
          setEditName("");
        },
      },
    );

  // ── Deleting ────────────────────────────────────────────────────────────
  const deleteOrganisation = useDeleteOrganisation();
  const [deleting, setDeleting] = useState<OrganisationView | null>(null);
  const remove = () =>
    deleting && deleteOrganisation.mutate(deleting.id, { onSuccess: () => setDeleting(null) });

  // ── Inviting a head ─────────────────────────────────────────────────────
  const createInvite = useCreateOrganisationInvite();
  const [inviting, setInviting] = useState<OrganisationView | null>(null);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteError, setInviteError] = useState<string | null>(null);
  const invite = () =>
    inviting &&
    createInvite.mutate(
      { institutionId: inviting.id, role: "head", invitedEmail: inviteEmail },
      {
        onSuccess: () => {
          setInviteError(null);
          setInviting(null);
          setInviteEmail("");
        },
        onError: (error) => setInviteError(inviteErrorMessage(t, error)),
      },
    );

  // ── Suspending ──────────────────────────────────────────────────────────
  const [suspending, setSuspending] = useState<OrganisationView | null>(null);

  return {
    list,
    /** Renaming is a change of the organisation's card. */
    canEdit: (org: OrganisationView) => org.mayEditCard,
    canDelete: (org: OrganisationView) => org.mayDelete,
    canSuspend: (org: OrganisationView) => org.maySuspend,

    requestEdit: (org: OrganisationView) => {
      setEditing(org);
      setEditName(org.name || "");
    },
    requestInvite: (org: OrganisationView) => {
      setInviting(org);
      setInviteEmail("");
      setInviteError(null);
    },
    requestSuspension: setSuspending,
    requestDelete: setDeleting,
    openCreate: () => setCreating(true),

    createModal: {
      opened: creating,
      close: () => setCreating(false),
      name: newName,
      setName: setNewName,
      details: newDetails,
      setDetail: (field: keyof OrganisationDetails, value: string) =>
        setNewDetails((d) => ({ ...d, [field]: value })),
      confirm: create,
      canConfirm: !!newName.trim(),
      loading: createOrganisation.isPending,
    },
    editModal: {
      opened: !!editing,
      close: () => setEditing(null),
      name: editName,
      setName: setEditName,
      confirm: rename,
      canConfirm: !!editName.trim(),
      loading: updateOrganisation.isPending,
    },
    inviteModal: {
      org: inviting,
      opened: !!inviting,
      close: () => {
        setInviteError(null);
        setInviting(null);
      },
      email: inviteEmail,
      setEmail: setInviteEmail,
      error: inviteError,
      confirm: invite,
      canConfirm: !!inviteEmail.trim(),
      loading: createInvite.isPending,
    },
    deleteModal: {
      org: deleting,
      opened: !!deleting,
      close: () => setDeleting(null),
      confirm: remove,
      loading: deleteOrganisation.isPending,
    },
    suspendModal: { org: suspending, close: () => setSuspending(null) },
  };
}

export type OrganizationsManagementState = ReturnType<typeof useOrganizationsManagement>;

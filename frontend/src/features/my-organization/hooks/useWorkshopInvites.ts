import { useState } from "react";
import { useTranslation } from "react-i18next";
import { notifications } from "@mantine/notifications";
import {
  useCreateWorkshopEmailInvite,
  useCreateWorkshopInvite,
  useRevokeInvite,
} from "@/api/hooks";
import type { InviteView, WorkshopView } from "@/api/types";
import { isUsableInvite } from "@/common/lib/invite";

type InvitedWorkshop = Pick<WorkshopView, "id" | "name" | "invites">;

type ShownInvite = Pick<InviteView, "id" | "inviteToken" | "expiresAt" | "usesCount"> & {
  meta?: { createdAt?: string };
};

/**
 * Inviting participants to a workshop, for WorkshopInviteModals to render:
 * its open invite link (as a QR code, revocable) and an invite by email.
 */
export function useWorkshopInvites() {
  const { t } = useTranslation("common");
  const createInvite = useCreateWorkshopInvite();
  const revokeInvite = useRevokeInvite();
  const createEmailInvite = useCreateWorkshopEmailInvite();

  // ── The invite link ─────────────────────────────────────────────────────
  const [linkWorkshop, setLinkWorkshop] = useState<InvitedWorkshop | null>(null);
  const [createdInvite, setCreatedInvite] = useState<ShownInvite | null>(null);

  /** The server returns the open invite or replaces a dead one. */
  const openLink = async (workshop: InvitedWorkshop) => {
    const invite = await createInvite.mutateAsync({ workshopId: workshop.id });
    setCreatedInvite(invite);
    setLinkWorkshop(workshop);
  };
  const revoke = async (inviteId: string) => {
    await revokeInvite.mutateAsync(inviteId);
    setCreatedInvite(null);
    setLinkWorkshop(null);
  };

  // ── Inviting by email ───────────────────────────────────────────────────
  const [emailWorkshop, setEmailWorkshop] = useState<InvitedWorkshop | null>(null);
  const [emailError, setEmailError] = useState<string | null>(null);

  const openEmail = (workshop: InvitedWorkshop) => {
    setEmailWorkshop(workshop);
    setEmailError(null);
  };
  const submitEmail = async (email: string) => {
    if (!emailWorkshop) return;
    try {
      await createEmailInvite.mutateAsync({ workshopId: emailWorkshop.id, email });
      setEmailError(null);
      setEmailWorkshop(null);
      notifications.show({
        title: t("myOrganization.workshops.inviteByEmailSuccess"),
        message: t("myOrganization.workshops.inviteByEmailSuccessMessage", { email }),
        color: "green",
      });
    } catch {
      setEmailError(t("myOrganization.workshops.addIndividualError"));
    }
  };

  return {
    openLink,
    openEmail,
    creatingLink: createInvite.isPending,
    hasUsableInvite: (workshop: InvitedWorkshop) => !!workshop.invites?.find(isUsableInvite),
    linkModal: {
      workshop: linkWorkshop,
      invite: createdInvite ?? linkWorkshop?.invites?.find(isUsableInvite) ?? null,
      close: () => setLinkWorkshop(null),
      revoke,
      revoking: revokeInvite.isPending,
    },
    emailModal: {
      opened: !!emailWorkshop,
      close: () => {
        setEmailWorkshop(null);
        setEmailError(null);
      },
      submit: submitEmail,
      loading: createEmailInvite.isPending,
      error: emailError,
    },
  };
}

export type WorkshopInvites = ReturnType<typeof useWorkshopInvites>;

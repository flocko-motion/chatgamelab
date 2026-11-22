import { useTranslation } from "react-i18next";
import {
  useOrganisation,
  useSetWorkshopApiKey,
  useUpdateWorkshop,
  useWorkshop,
} from "@/api/hooks";
import { getAiQualityTierOptions } from "@/common/lib/aiQualityTier";
import { useAuth } from "@/providers/AuthProvider";
import { useOrgKeyChoice } from "./useOrgKeyChoice";
import { useWorkshopInvites } from "./useWorkshopInvites";

export interface WorkshopSettingsChange {
  showPublicGames?: boolean;
  showOtherParticipantsGames?: boolean;
  aiQualityTier?: string;
  promptConstraints?: string;
  isPaused?: boolean;
  allowGameSharing?: boolean;
}

/**
 * One workshop's settings, for SingleWorkshopSettings to render: the
 * workshop, its default key (one of the caller's own keys is first given to
 * the organisation), its settings, and the invites.
 */
export function useWorkshopSettings(workshopId: string, institutionId: string) {
  const { t } = useTranslation("common");
  const { retryBackendFetch } = useAuth();
  const { data: workshop, isLoading, isError } = useWorkshop(workshopId);
  const { data: institution } = useOrganisation(institutionId);
  const updateWorkshop = useUpdateWorkshop();
  const setWorkshopApiKey = useSetWorkshopApiKey();

  const keyChoice = useOrgKeyChoice({
    institutionId,
    apply: (apiKeyShareId) => setWorkshopApiKey.mutateAsync({ workshopId, apiKeyShareId }),
    applying: setWorkshopApiKey.isPending,
    emptyLabel: t("myOrganization.workshops.noDefaultApiKey"),
  });
  const invites = useWorkshopInvites();

  const update = async (change: WorkshopSettingsChange) => {
    if (!workshop) return;
    await updateWorkshop.mutateAsync({
      id: workshop.id,
      name: workshop.name || "",
      active: workshop.active || false,
      public: workshop.public || false,
      showPublicGames: change.showPublicGames ?? workshop.showPublicGames ?? false,
      showOtherParticipantsGames:
        change.showOtherParticipantsGames ?? workshop.showOtherParticipantsGames ?? true,
      aiQualityTier: change.aiQualityTier ?? workshop.aiQualityTier ?? undefined,
      promptConstraints: change.promptConstraints ?? workshop.promptConstraints ?? undefined,
      isPaused: change.isPaused ?? workshop.isPaused ?? false,
      allowGameSharing: change.allowGameSharing ?? workshop.allowGameSharing ?? false,
    });
    // The workshop's settings also sit in the caller's role (role.workshop).
    retryBackendFetch();
  };

  /** Saves the constraint when it changed. */
  const savePromptConstraints = (value: string) => {
    if ((workshop?.promptConstraints || "") === value) return;
    update({ promptConstraints: value });
  };

  return {
    workshop,
    isLoading,
    isError,
    update,
    savePromptConstraints,
    updating: updateWorkshop.isPending,
    /** What applies when the workshop sets no constraint of its own. */
    promptPlaceholder: institution?.promptConstraints
      ? t("myOrganization.workshops.promptConstraintsFallbackOrg", {
          constraint: institution.promptConstraints,
        })
      : t("myOrganization.workshops.promptConstraintsFallbackSite"),
    aiQualityTierOptions: getAiQualityTierOptions(t, { includeEmpty: true }),
    keyChoice,
    settingKey: setWorkshopApiKey.isPending,
    invites,
    /** The public page was saved: its settings also sit in the caller's role. */
    onPublicPageSaved: retryBackendFetch,
  };
}

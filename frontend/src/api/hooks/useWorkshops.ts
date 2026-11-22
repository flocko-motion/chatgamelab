import { api } from "../client/api";
import {
  useQuery,
  useMutation,
  useQueryClient,
  keepPreviousData,
} from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import type { PublicWorkshopLink, WorkshopView } from "../generated/api.gen";

/**
 * Parameters for listing workshops with filtering and sorting
 */
export interface UseWorkshopsParams {
  institutionId: string | undefined;
  search?: string;
  sortBy?: "name" | "createdAt" | "participantCount";
  sortDir?: "asc" | "desc";
  activeOnly?: boolean;
}

/**
 * Hook to fetch workshops for a specific institution with filtering and sorting
 */
export function useWorkshops(params: UseWorkshopsParams) {
  const { institutionId, search, sortBy, sortDir, activeOnly } = params;

  return useQuery({
    queryKey: [
      ...queryKeys.workshopsByInstitution(institutionId || ""),
      { search, sortBy, sortDir, activeOnly },
    ],
    queryFn: async () => {
      if (!institutionId) return [];
      const response = await api.getApiWorkshops({
          institutionId,
          search: search || undefined,
          sortBy: sortBy || undefined,
          sortDir: sortDir || undefined,
          activeOnly: activeOnly ? "true" : undefined,
        });
      return response.data;
    },
    enabled: !!institutionId,
    placeholderData: keepPreviousData,
  });
}

/**
 * Hook to fetch a single workshop by ID
 */
export function useWorkshop(workshopId: string | undefined) {

  return useQuery({
    queryKey: queryKeys.workshop(workshopId || ""),
    queryFn: async () => {
      if (!workshopId) return null;
      const response = await api.getApiWorkshopsId(workshopId);
      return response.data;
    },
    enabled: !!workshopId,
  });
}

/**
 * Hook to create a new workshop
 */
export function useCreateWorkshop() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: {
      name: string;
      institutionId?: string;
      active?: boolean;
      public?: boolean;
    }) => {
      const response = await api.postApiWorkshops({
        name: data.name,
        institutionId: data.institutionId,
        active: data.active ?? true,
        public: data.public ?? false,
      });
      return response.data;
    },
    onSuccess: () => {
      // Invalidate all workshop queries
      queryClient.invalidateQueries({ queryKey: ["workshops"] });
    },
  });
}

/**
 * Hook to update a workshop
 */
export function useUpdateWorkshop() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      id,
      ...data
    }: {
      id: string;
      name: string;
      active?: boolean;
      public?: boolean;
      showPublicGames?: boolean;
      showOtherParticipantsGames?: boolean;
      aiQualityTier?: string;
      promptConstraints?: string;
      isPaused?: boolean;
      allowGameSharing?: boolean;
      // Omitted keeps the current value on the server.
      publicSlug?: string;
      publicDescription?: string;
      publicLinks?: PublicWorkshopLink[];
    }) => {
      const response = await api.patchApiWorkshopsId(id, {
        publicSlug: data.publicSlug,
        publicDescription: data.publicDescription,
        publicLinks: data.publicLinks,
        name: data.name,
        active: data.active ?? true,
        public: data.public ?? false,
        showPublicGames: data.showPublicGames ?? false,
        showOtherParticipantsGames: data.showOtherParticipantsGames ?? true,
        aiQualityTier: data.aiQualityTier,
        promptConstraints: data.promptConstraints,
        isPaused: data.isPaused ?? false,
        allowGameSharing: data.allowGameSharing ?? false,
      });
      return response.data;
    },
    onSuccess: (_, variables) => {
      // Invalidate all workshop queries (including workshopsByInstitution)
      queryClient.invalidateQueries({ queryKey: ["workshops"] });
      queryClient.invalidateQueries({
        queryKey: queryKeys.workshop(variables.id),
      });
      // Workshop settings affect game visibility filtering on the backend,
      // so refetch games when settings change
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
    },
  });
}

/**
 * Hook to delete a workshop
 */
export function useDeleteWorkshop() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (workshopId: string) => {
      await api.deleteApiWorkshopsId(workshopId);
    },
    onSuccess: () => {
      // Invalidate all workshop queries
      queryClient.invalidateQueries({ queryKey: ["workshops"] });
    },
  });
}

/**
 * Hook to create a workshop invite (open invite for participants)
 */
export function useCreateWorkshopInvite() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: {
      workshopId: string;
      maxUses?: number;
      expiresAt?: string;
    }) => {
      const response = await api.postApiInvitesWorkshop({
        workshopId: data.workshopId,
        maxUses: data.maxUses,
        expiresAt: data.expiresAt,
      });
      return response.data;
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ["workshops"] });
      queryClient.invalidateQueries({
        queryKey: queryKeys.workshop(variables.workshopId),
      });
      queryClient.invalidateQueries({ queryKey: queryKeys.invites });
    },
  });
}

/**
 * Hook to set workshop default API key for participants
 */
export function useSetWorkshopApiKey() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: {
      workshopId: string;
      apiKeyShareId: string | null;
    }) => {
      const response = await api.putApiWorkshopsIdApiKey(data.workshopId, {
        apiKeyShareId: data.apiKeyShareId || undefined,
      });
      return response.data;
    },
    onSuccess: (updatedWorkshop, variables) => {
      // Update the workshop in the cache immediately with the new data
      queryClient.setQueryData<WorkshopView[]>(["workshops"], (old) => {
        if (!old) return old;
        return old.map((w) =>
          w.id === variables.workshopId ? updatedWorkshop : w,
        );
      });
      queryClient.invalidateQueries({ queryKey: ["workshops"] });
      queryClient.invalidateQueries({
        queryKey: queryKeys.workshop(variables.workshopId),
      });
    },
  });
}

/**
 * Hook to update a participant's name
 */
export function useUpdateParticipant() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { participantId: string; name: string }) => {
      const response = await api.postApiUsersId(data.participantId, {
        name: data.name,
      });
      return response.data;
    },
    onSuccess: () => {
      // Invalidate all workshop-related queries to refresh participant data
      queryClient.invalidateQueries({ queryKey: ["workshops"] });
      queryClient.invalidateQueries({ queryKey: ["workshop"] });
    },
  });
}

/**
 * Hook to replace the login token of a Guest in a workshop; earlier re-login links stop working.
 */
export function useResetGuestToken() {

  return useMutation({
    mutationFn: async (guestId: string) => {
      const response =
        await api.postApiWorkshopsGuestsGuestIdTokenReset(guestId);
      return response.data;
    },
  });
}

/**
 * Hook to remove a member from a workshop.
 * Permanent members (Guests) are HARD-deleted: db.DeleteUser removes the
 * account row and every game they created, with its sessions and shared links.
 * Non-permanent members (individuals/visiting head/staff) have their active workshop cleared.
 */
export function useRemoveParticipant() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      participantId,
      workshopId,
      permanent,
    }: {
      participantId: string;
      workshopId: string;
      permanent: boolean;
    }) => {
      if (permanent) {
        await api.deleteApiUsersId(participantId);
      } else {
        await api.deleteApiWorkshopsIdMembersUserId(workshopId, participantId);
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.workshops });
      queryClient.invalidateQueries({ queryKey: ["workshop"] });
    },
  });
}

/**
 * Hook to invite a registered user to a workshop by email
 */
export function useCreateWorkshopEmailInvite() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { workshopId: string; email: string }) => {
      const response = await api.postApiInvitesWorkshopEmail({
        workshopId: data.workshopId,
        email: data.email,
      });
      return response.data;
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ["workshops"] });
      queryClient.invalidateQueries({
        queryKey: queryKeys.workshop(variables.workshopId),
      });
      queryClient.invalidateQueries({ queryKey: queryKeys.invites });
    },
  });
}


import {
  useQuery,
  useMutation,
  useQueryClient,
  keepPreviousData,
} from "@tanstack/react-query";
import { useRequiredAuthenticatedApi } from "../useAuthenticatedApi";
import { queryKeys } from "../queryKeys";
import type { ObjWorkshop } from "../generated";

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
  const api = useRequiredAuthenticatedApi();
  const { institutionId, search, sortBy, sortDir, activeOnly } = params;

  return useQuery({
    queryKey: [
      ...queryKeys.workshopsByInstitution(institutionId || ""),
      { search, sortBy, sortDir, activeOnly },
    ],
    queryFn: async () => {
      if (!institutionId) return [];
      const response = await api.workshops.workshopsList({
        institutionId,
        search: search || undefined,
        sortBy: sortBy || undefined,
        sortDir: sortDir || undefined,
        activeOnly: activeOnly || undefined,
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
  const api = useRequiredAuthenticatedApi();

  return useQuery({
    queryKey: queryKeys.workshop(workshopId || ""),
    queryFn: async () => {
      if (!workshopId) return null;
      const response = await api.workshops.workshopsDetail(workshopId);
      return response.data;
    },
    enabled: !!workshopId,
  });
}

/**
 * Hook to create a new workshop
 */
export function useCreateWorkshop() {
  const api = useRequiredAuthenticatedApi();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: {
      name: string;
      institutionId?: string;
      active?: boolean;
      public?: boolean;
    }) => {
      const response = await api.workshops.workshopsCreate({
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
  const api = useRequiredAuthenticatedApi();
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
      showAiModelSelector?: boolean;
      showPublicGames?: boolean;
      showOtherParticipantsGames?: boolean;
      useSpecificAiModel?: string;
    }) => {
      const response = await api.workshops.workshopsPartialUpdate(id, {
        name: data.name,
        active: data.active ?? true,
        public: data.public ?? false,
        showAiModelSelector: data.showAiModelSelector ?? false,
        showPublicGames: data.showPublicGames ?? false,
        showOtherParticipantsGames: data.showOtherParticipantsGames ?? true,
        useSpecificAiModel: data.useSpecificAiModel,
      });
      return response.data;
    },
    onSuccess: (_, variables) => {
      // Invalidate all workshop queries (including workshopsByInstitution)
      queryClient.invalidateQueries({ queryKey: ["workshops"] });
      queryClient.invalidateQueries({
        queryKey: queryKeys.workshop(variables.id),
      });
    },
  });
}

/**
 * Hook to delete a workshop
 */
export function useDeleteWorkshop() {
  const api = useRequiredAuthenticatedApi();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (workshopId: string) => {
      await api.workshops.workshopsDelete(workshopId);
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
  const api = useRequiredAuthenticatedApi();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: {
      workshopId: string;
      maxUses?: number;
      expiresAt?: string;
    }) => {
      const response = await api.invites.workshopCreate({
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
  const api = useRequiredAuthenticatedApi();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: {
      workshopId: string;
      apiKeyShareId: string | null;
    }) => {
      const response = await api.workshops.apiKeyUpdate(data.workshopId, {
        apiKeyShareId: data.apiKeyShareId || undefined,
      });
      return response.data;
    },
    onSuccess: (updatedWorkshop, variables) => {
      // Update the workshop in the cache immediately with the new data
      queryClient.setQueryData<ObjWorkshop[]>(["workshops"], (old) => {
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
  const api = useRequiredAuthenticatedApi();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { participantId: string; name: string }) => {
      const response = await api.users.usersCreate(data.participantId, {
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
 * Hook to get a participant's login token (for creating individual share links)
 */
export function useGetParticipantToken() {
  const api = useRequiredAuthenticatedApi();

  return useMutation({
    mutationFn: async (participantId: string) => {
      const response = await api.workshops.participantsTokenList(participantId);
      return response.data;
    },
  });
}

/**
 * Hook to remove a participant from a workshop (soft-delete)
 */
export function useRemoveParticipant() {
  const api = useRequiredAuthenticatedApi();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (participantId: string) => {
      await api.users.usersDelete(participantId);
    },
    onSuccess: () => {
      // Invalidate workshop queries to refresh participant list
      queryClient.invalidateQueries({ queryKey: ["workshops"] });
    },
  });
}

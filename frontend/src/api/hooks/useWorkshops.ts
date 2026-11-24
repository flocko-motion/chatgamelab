import { api } from "../client/api";
import { rawRequest } from "../client/raw";
import {
  useQuery,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import type {
  Error as ApiError,
  PublicWorkshopPage,
  PublicWorkshopLink,
  ParticipantSort,
  WorkshopFilter,
  WorkshopParticipantView,
  WorkshopSort,
  WorkshopView,
} from "../generated/api.gen";
import { useEntityList, type List, type ListPreset } from "../lists";

/** One workshop, or null without an id; `useParticipants` shares the query. */
function fetchWorkshop(workshopId: string | undefined): Promise<WorkshopView | null> {
  if (!workshopId) return Promise.resolve(null);
  return api.getApiWorkshopsId(workshopId).then((response) => response.data);
}

/**
 * Hook to fetch a single workshop by ID
 */
export function useWorkshop(workshopId: string | undefined) {

  return useQuery({
    queryKey: queryKeys.workshop(workshopId || ""),
    queryFn: () => fetchWorkshop(workshopId),
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
/** Replaces a guest's sign-in token; the new one replaces the cached one. */
export function useResetGuestToken() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (guestId: string) => {
      const response =
        await api.postApiWorkshopsGuestsGuestIdTokenReset(guestId);
      return response.data;
    },
    onSuccess: (result, guestId) => {
      queryClient.setQueryData(queryKeys.guestToken(guestId), result?.token ?? null);
    },
  });
}

/**
 * Takes a participant out of a workshop.
 * Permanent participants (Guests) are HARD-deleted: db.DeleteUser removes the
 * account row and every game they created, with its sessions and shared links.
 * Others taking part (head, staff, individuals) have their active workshop
 * cleared; taking part made no one part of the organisation.
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


/** A filter of the workshop list (the server's enum): `active` keeps the active workshops. */
export type WorkshopListFilter = WorkshopFilter;

/** An order of the workshop list (the server's enum). */
export type WorkshopListSort = WorkshopSort;

/** An organisation's workshops, as a list. */
export function useWorkshopList<F extends string, O extends WorkshopListFilter = never>(
  institutionId: string | undefined,
  preset: ListPreset<F, WorkshopListSort, O>,
): List<WorkshopView, F, WorkshopListSort, O> {
  return useEntityList<WorkshopView, F, WorkshopListSort, O>(
    preset,
    {
      key: (params) => [...queryKeys.workshopsByInstitution(institutionId ?? ""), { ...params }],
      fetch: (params) =>
        api
          .getApiWorkshops({ ...(params as WorkshopListQuery), institutionId: institutionId! })
          .then((r) => r.data),
    },
    { enabled: !!institutionId },
  );
}

/** An order of the participant list (the server's enum). */
export type ParticipantListSort = ParticipantSort;

/** A workshop's participants, as a list. */
export function useParticipants<F extends string>(
  workshopId: string | undefined,
  preset: ListPreset<F, ParticipantListSort>,
): List<WorkshopParticipantView, F, ParticipantListSort> {
  return useEntityList<WorkshopParticipantView, F, ParticipantListSort>(
    preset,
    {
      key: (params) => [...queryKeys.workshop(workshopId ?? ""), "participants", { ...params }],
      fetch: (params) =>
        api
          .getApiWorkshopsIdParticipants(workshopId!, params as ParticipantListQuery)
          .then((r) => r.data),
    },
    { enabled: !!workshopId },
  );
}

// A list's parameters come from presets typed by the server's enums; the
// server refuses what it does not know.
type WorkshopListQuery = NonNullable<Parameters<typeof api.getApiWorkshops>[0]>;
type ParticipantListQuery = Parameters<typeof api.getApiWorkshopsIdParticipants>[1];

/**
 * A guest's sign-in token, read fresh each time (never cached): it is a
 * secret, shown only while the guest link is open.
 */
export function useGuestToken(guestId: string | undefined) {
  return useQuery({
    queryKey: queryKeys.guestToken(guestId ?? ""),
    queryFn: () =>
      api.getApiWorkshopsGuestsGuestIdToken(guestId!).then((r) => r.data?.token ?? null),
    enabled: !!guestId,
    staleTime: 0,
    gcTime: 0,
  });
}

/**
 * The page a workshop shows the world at /w/<slug>. Not retried: a slug that
 * is unknown or not public answers at once.
 */
export function usePublicWorkshop(slug: string) {
  return useQuery<PublicWorkshopPage, ApiError>({
    queryKey: queryKeys.publicWorkshop(slug),
    queryFn: () => api.getApiPublicWorkshopsSlug(slug).then((r) => r.data),
    retry: false,
  });
}

/** A game on a public workshop page as a .cgl file, its YAML and cover; errors are the caller's. */
export function usePublicWorkshopGameFile() {
  return useMutation<Blob, ApiError, { slug: string; gameId: string }>({
    mutationFn: ({ slug, gameId }) =>
      rawRequest(`/api/public/workshops/${encodeURIComponent(slug)}/games/${gameId}/cgl`).then((r) => r.blob()),
    retry: false,
  });
}

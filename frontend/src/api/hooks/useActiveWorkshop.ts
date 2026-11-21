import { api } from "../client/api";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import type { GetCurrentUserResp, Error as ApiError } from "../generated/api.gen";

// Mutation key for deduplication
const MUTATION_KEY = ["setActiveWorkshop"] as const;

/**
 * Hook to set the active workshop for head/staff/individual users (workshop mode)
 * This persists the workshop context in the backend
 */
export function useSetActiveWorkshop() {
  const queryClient = useQueryClient();

  return useMutation<GetCurrentUserResp, ApiError, string | null>({
    mutationKey: MUTATION_KEY,
    mutationFn: async (workshopId: string | null) => {
      const response = await api.putApiUsersMeActiveWorkshop({
        workshopId: workshopId ?? undefined,
      });
      return response.data;
    },
    // the caller is refetched by WorkshopModeProvider; the user view too
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.currentUser });
    },
  });
}

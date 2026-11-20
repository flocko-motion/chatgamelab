import { api } from "../client/api";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import type { User, Error as ApiError } from "../generated/api.gen";

// Mutation key for deduplication
const MUTATION_KEY = ["setActiveWorkshop"] as const;

/**
 * Hook to set the active workshop for head/staff/individual users (workshop mode)
 * This persists the workshop context in the backend
 */
export function useSetActiveWorkshop() {
  const queryClient = useQueryClient();

  return useMutation<User, ApiError, string | null>({
    mutationKey: MUTATION_KEY,
    mutationFn: async (workshopId: string | null) => {
      const response = await api.putApiUsersMeActiveWorkshop({
        workshopId: workshopId ?? undefined,
      });
      return response.data;
    },
    onSuccess: (updatedUser) => {
      // Update the cached backend user with the new workshop context
      queryClient.setQueryData(queryKeys.backendUser, updatedUser);
      // Also invalidate to ensure fresh data
      queryClient.invalidateQueries({ queryKey: queryKeys.backendUser });
    },
  });
}

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuthenticatedApi } from "../useAuthenticatedApi";
import { queryKeys } from "../queryKeys";
import type { ObjUser, HttpxErrorResponse } from "../generated";

/**
 * Hook to set the active workshop for head/staff/individual users (workshop mode)
 * This persists the workshop context in the backend
 */
export function useSetActiveWorkshop() {
  const api = useAuthenticatedApi();
  const queryClient = useQueryClient();

  return useMutation<ObjUser, HttpxErrorResponse, string | null>({
    mutationFn: async (workshopId: string | null) => {
      if (!api) {
        throw new Error("Not authenticated");
      }
      const response = await api.users.meActiveWorkshopUpdate({
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

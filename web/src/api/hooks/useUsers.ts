import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { handleApiError } from "@/config/queryClient";
import { useRequiredAuthenticatedApi } from "../useAuthenticatedApi";
import { queryKeys } from "../queryKeys";
import type {
  ObjUser,
  ObjUserStats,
  HttpxErrorResponse,
  RoutesUserUpdateRequest,
} from "../generated";

export function useCurrentUser() {
  const api = useRequiredAuthenticatedApi();

  return useQuery<ObjUser, HttpxErrorResponse>({
    queryKey: queryKeys.currentUser,
    queryFn: () => api.users.getUsers().then((response) => response.data),
  });
}

export function useUserStats() {
  const api = useRequiredAuthenticatedApi();

  return useQuery<ObjUserStats, HttpxErrorResponse>({
    queryKey: [...queryKeys.currentUser, "stats"],
    queryFn: () => api.users.meStatsList().then((response) => response.data),
  });
}

export function useUpdateUser() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();

  return useMutation<
    ObjUser,
    HttpxErrorResponse,
    { id: string; request: RoutesUserUpdateRequest }
  >({
    mutationFn: ({ id, request }) =>
      api.users.usersCreate(id, request).then((response) => response.data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.users });
      queryClient.invalidateQueries({ queryKey: [...queryKeys.users, id] });
      // Always invalidate currentUser - settings updates affect the logged-in user
      queryClient.invalidateQueries({ queryKey: queryKeys.currentUser });
    },
    onError: handleApiError,
  });
}


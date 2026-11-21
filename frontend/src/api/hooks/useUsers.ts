import { api } from "../client/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { handleApiError } from "@/config/queryClient";
import { queryKeys } from "../queryKeys";
import type {
  UserView,
  UserStatsView,
  Error as ApiError,
  UserSecretsView,
  UpdateUserReq,
} from "../generated/api.gen";

export function useCurrentUser() {

  return useQuery<UserView, ApiError>({
    queryKey: queryKeys.currentUser,
    queryFn: () =>
      api.getApiUsersMe().then((response) => {
        if (!response.data.user) throw new Error("not registered");
        return response.data.user;
      }),
  });
}

export function useUserStats() {

  return useQuery<UserStatsView, ApiError>({
    queryKey: [...queryKeys.currentUser, "stats"],
    queryFn: () => api.getApiUsersMeStats().then((response) => response.data),
  });
}

export function useOwnGuestCode() {

  return useQuery<UserSecretsView, ApiError>({
    queryKey: [...queryKeys.currentUser, "guestCode"],
    queryFn: () => api.getApiUsersMeSecrets().then((response) => response.data),
  });
}

export function useUpdateUser() {
  const queryClient = useQueryClient();

  return useMutation<
    UserView,
    ApiError,
    { id: string; request: UpdateUserReq }
  >({
    mutationFn: ({ id, request }) =>
      api.postApiUsersId(id, request).then((response) => response.data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.users });
      queryClient.invalidateQueries({ queryKey: [...queryKeys.users, id] });
      // Always invalidate currentUser - settings updates affect the logged-in user
      queryClient.invalidateQueries({ queryKey: queryKeys.currentUser });
    },
    onError: handleApiError,
  });
}

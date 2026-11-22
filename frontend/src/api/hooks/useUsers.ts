import { api } from "../client/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { handleApiError } from "@/config/queryClient";
import { apiLogger } from "@/config/logger";
import { queryKeys } from "../queryKeys";
import type {
  UserView,
  UserStatsView,
  Error as ApiError,
  UserSecretsView,
  UpdateUserReq,
  SetUserRoleReq,
  UserFilter,
  UserSort,
} from "../generated/api.gen";
import { useEntityList, type List, type ListPreset } from "../lists";

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
      invalidateUsers(queryClient);
      queryClient.invalidateQueries({ queryKey: [...queryKeys.users, id] });
      // Always invalidate currentUser - settings updates affect the logged-in user
      queryClient.invalidateQueries({ queryKey: queryKeys.currentUser });
    },
    onError: handleApiError,
  });
}

/**
 * Stores the caller's language preference. Best-effort: not retried, and a
 * failure is only logged, never shown.
 */
export function useSetLanguage() {
  return useMutation<UserView, ApiError, string>({
    mutationFn: (language) =>
      api.patchApiUsersMeLanguage({ language }).then((response) => response.data),
    retry: false,
    onError: (error) => {
      apiLogger.warning("Failed to persist language preference", { error });
    },
  });
}

/** A filter of the user list (the server's enum): `noGuests` leaves out Guests and users without a role. */
export type UserListFilter = UserFilter;

/** An order of the user list (the server's enum). */
export type UserListSort = UserSort;

/** Every user the caller may see (the admins' user list), as a list. */
export function useUsers<F extends string, O extends UserListFilter = never>(
  preset: ListPreset<F, UserListSort, O>,
): List<UserView, F, UserListSort, O> {
  return useEntityList<UserView, F, UserListSort, O>(preset, {
    key: (params) => [...queryKeys.adminUsers, { ...params }],
    fetch: (params) => api.getApiUsers(params as UserListQuery).then((r) => r.data),
  });
}

// A list's parameters come from presets typed by the server's enums; the
// server refuses what it does not know.
type UserListQuery = Parameters<typeof api.getApiUsers>[0];

/** After a user's account or role changed: every list and view that shows users. */
function invalidateUsers(queryClient: ReturnType<typeof useQueryClient>) {
  queryClient.invalidateQueries({ queryKey: queryKeys.adminUsers });
  queryClient.invalidateQueries({ queryKey: queryKeys.users });
  queryClient.invalidateQueries({ queryKey: queryKeys.institutionMembersBase });
  queryClient.invalidateQueries({ queryKey: queryKeys.institutions });
}

/** Deletes a user's account. */
export function useDeleteUser() {
  const queryClient = useQueryClient();
  return useMutation<unknown, ApiError, string>({
    mutationFn: (userId) => api.deleteApiUsersId(userId).then((r) => r.data),
    onSuccess: () => invalidateUsers(queryClient),
    onError: handleApiError,
  });
}

/** Gives a user a role (admin; head or staff in an organisation). */
export function useSetUserRole() {
  const queryClient = useQueryClient();
  return useMutation<unknown, ApiError, { userId: string } & SetUserRoleReq>({
    mutationFn: ({ userId, ...role }) => api.postApiUsersIdRole(userId, role).then((r) => r.data),
    onSuccess: () => invalidateUsers(queryClient),
    onError: handleApiError,
  });
}

/** Takes a user's role away (e.g. admin). */
export function useRemoveUserRole() {
  const queryClient = useQueryClient();
  return useMutation<unknown, ApiError, string>({
    mutationFn: (userId) => api.deleteApiUsersIdRole(userId).then((r) => r.data),
    onSuccess: () => invalidateUsers(queryClient),
    onError: handleApiError,
  });
}

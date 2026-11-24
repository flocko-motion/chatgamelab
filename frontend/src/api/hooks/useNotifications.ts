import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { handleApiError } from "@/config/queryClient";
import { api } from "../client/api";
import { queryKeys } from "../queryKeys";
import type {
  AcknowledgeNotificationReq,
  Error as ApiError,
  NotificationView,
  NotificationsResp,
} from "../generated/api.gen";

/** How often the bell asks again while the tab is visible. */
const NOTIFICATIONS_POLL_MS = 60_000;

/**
 * The caller's notifications, newest first: the facts that are true now and
 * wait for their acknowledgement. Asked on load and every minute while the
 * tab is visible (a hidden tab asks nothing).
 */
export function useNotifications() {
  return useQuery<NotificationView[], ApiError>({
    queryKey: queryKeys.notifications,
    queryFn: () => api.getApiNotifications().then((r) => r.data.notifications),
    refetchInterval: NOTIFICATIONS_POLL_MS,
    refetchIntervalInBackground: false,
  });
}

/** Acknowledges one notification up to the entry the caller saw; the answer is what remains. */
export function useAcknowledgeNotification() {
  const queryClient = useQueryClient();
  return useMutation<NotificationsResp, ApiError, AcknowledgeNotificationReq>({
    mutationFn: (req) => api.postApiNotificationsAcknowledge(req).then((r) => r.data),
    onSuccess: (resp) => queryClient.setQueryData(queryKeys.notifications, resp.notifications),
    onError: handleApiError,
  });
}

/** Acknowledges every notification the caller has; the answer is what remains. */
export function useAcknowledgeAllNotifications() {
  const queryClient = useQueryClient();
  return useMutation<NotificationsResp, ApiError, void>({
    mutationFn: () => api.postApiNotificationsAcknowledgeAll().then((r) => r.data),
    onSuccess: (resp) => queryClient.setQueryData(queryKeys.notifications, resp.notifications),
    onError: handleApiError,
  });
}

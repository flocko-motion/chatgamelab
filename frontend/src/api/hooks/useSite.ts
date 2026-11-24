import { api } from "../client/api";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { handleApiError } from "@/config/queryClient";
import { queryKeys } from "../queryKeys";
import type { Error as ApiError, SiteResp } from "../generated/api.gen";

/** How the site presents itself: its branding, and each field's default. Public. */
export function useSite() {
  return useQuery<SiteResp, ApiError>({
    queryKey: queryKeys.site,
    queryFn: () => api.getApiSite().then((r) => r.data),
    staleTime: Infinity,
  });
}

/**
 * Changes the site's branding (admin only): a value sets a field, null gives
 * it back its default; fields not sent stay.
 */
export function useUpdateSiteBranding() {
  const queryClient = useQueryClient();
  return useMutation<SiteResp, ApiError, Record<string, string | null>>({
    mutationFn: (branding) =>
      api
        .patchApiSystemSettingsBranding({ branding: branding as Record<string, string> })
        .then((r) => r.data),
    onSuccess: (site) => {
      queryClient.setQueryData(queryKeys.site, site);
    },
    onError: handleApiError,
  });
}

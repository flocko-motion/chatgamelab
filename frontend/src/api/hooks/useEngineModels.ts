import { api } from "../client/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { handleApiError } from "@/config/queryClient";
import { queryKeys } from "../queryKeys";
import type {
  AssignEngineModelReq,
  EngineModelsResp,
  Error as ApiError,
  ProbeEngineModelReq,
  ProbeEngineModelResp,
  SetEngineModelEntryReq,
} from "../generated/api.gen";

/** The engine's models and model table, per platform (admin only). */
export function useEngineModels() {
  return useQuery<EngineModelsResp, ApiError>({
    queryKey: queryKeys.engineModels,
    queryFn: () => api.getApiSystemEngineModels().then((r) => r.data),
  });
}

/** Writes answer the engine's models as they now are; the platforms list follows. */
function useEngineModelsWrite<Req>(write: (req: Req) => Promise<{ data: EngineModelsResp }>) {
  const queryClient = useQueryClient();
  return useMutation<EngineModelsResp, ApiError, Req>({
    mutationFn: (req) => write(req).then((r) => r.data),
    onSuccess: (models) => {
      queryClient.setQueryData(queryKeys.engineModels, models);
      queryClient.invalidateQueries({ queryKey: queryKeys.platforms });
    },
  });
}

/** Assigns a model to a tier's task; an empty model gives back the default. */
export function useAssignEngineModel() {
  return useEngineModelsWrite((req: AssignEngineModelReq) => api.putApiSystemEngineModelsAssignment(req));
}

/** Sets a model's entry in the model table; no facts reset it. */
export function useSetEngineModelEntry() {
  return useEngineModelsWrite((req: SetEngineModelEntryReq) => api.putApiSystemEngineModelsEntry(req));
}

/** Asks whether a model answers; paid by the admin's own key. */
export function useProbeEngineModel() {
  return useMutation<ProbeEngineModelResp, ApiError, ProbeEngineModelReq>({
    mutationFn: (req) => api.postApiSystemEngineModelsProbe(req).then((r) => r.data),
    onError: handleApiError,
  });
}

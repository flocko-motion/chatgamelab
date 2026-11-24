import { useState } from "react";
import {
  useAssignEngineModel,
  useEngineModels,
  useProbeEngineModel,
  useSetEngineModelEntry,
} from "@/api/hooks";
import type {
  EngineModelsPlatform,
  Medium,
  ModelFacts,
  ModelFactsOverride,
  ModelSlotView,
  ModelTableRowView,
  ModelType,
  Price,
  ProbeEngineModelResp,
  Tier,
} from "@/api/types";
import { extractRawErrorMessage } from "@/common/types/errorCodes";

/** The tasks a tier runs a model on, in the engine's order. */
export const MODEL_TYPES: ModelType[] = ["tooling", "story", "image", "live", "speech", "hearing"];

/** The tiers, cheapest first. */
export const TIERS: Tier[] = ["low", "medium", "high"];

/** The kinds of content a model takes or makes. */
export const MEDIA: Medium[] = ["text", "image", "audio"];

/** A model table entry's price fields, in the order the form shows them. */
export const PRICE_FIELDS: (keyof Price)[] = [
  "inputPerMTok",
  "cachedInputPerMTok",
  "outputPerMTok",
  "audioInputPerMTok",
  "cachedAudioInputPerMTok",
  "audioOutputPerMTok",
  "audioPerMinute",
  "perImage",
];

/**
 * What a task needs its model to take and make, and whether it needs a
 * context window: the same rule the engine checks every assignment by, so
 * a choice offers only models it would take.
 */
const NEEDS: Record<ModelType, { in: Medium; out: Medium; window: boolean }> = {
  tooling: { in: "text", out: "text", window: false },
  story: { in: "text", out: "text", window: true },
  image: { in: "text", out: "image", window: false },
  live: { in: "audio", out: "audio", window: false },
  speech: { in: "text", out: "audio", window: true },
  hearing: { in: "audio", out: "text", window: false },
};

/** The task types with a probe: the others answer "no probe yet". */
const PROBED: ModelType[] = ["tooling", "story", "image"];

/** Whether a model's entry fits a task. */
export function fits(facts: ModelFacts, type: ModelType): boolean {
  const need = NEEDS[type];
  return facts.in.includes(need.in) && facts.out.includes(need.out) && (!need.window || (facts.window ?? 0) > 0);
}

/** The entry being edited: a model's row, or a new model. */
export interface EntryDraft {
  platform: string;
  /** The row edited; none for a model being added. */
  row?: ModelTableRowView;
}

/** The cell being assigned. */
export interface SlotDraft {
  platform: string;
  slot: ModelSlotView;
}

/** A probe's result, by platform, task and model. */
type ProbeKey = string;
const probeKey = (platform: string, type: ModelType, model: string): ProbeKey => `${platform}/${type}/${model}`;

/**
 * The engine's models for the admin: per platform the model each tier runs
 * each task on and the model table, as the server sends them; assigning,
 * setting and resetting entries, and probing a model. The server checks
 * every change; a refusal comes back as its message.
 */
export function useEngineModelsPage() {
  const { data, isLoading } = useEngineModels();
  const assign = useAssignEngineModel();
  const setEntry = useSetEngineModelEntry();
  const probe = useProbeEngineModel();
  const [probes, setProbes] = useState<Record<ProbeKey, ProbeEngineModelResp>>({});
  const [probing, setProbing] = useState<ProbeKey | null>(null);

  const platforms: EngineModelsPlatform[] = data?.platforms ?? [];

  /** The cell of a tier's task, if the platform has the task. */
  const slot = (p: EngineModelsPlatform, tier: Tier, type: ModelType) =>
    p.assignments.find((a) => a.tier === tier && a.type === type);

  /** The tasks a platform has, in order. */
  const types = (p: EngineModelsPlatform) => MODEL_TYPES.filter((type) => p.assignments.some((a) => a.type === type));

  /** The models of the table a task may run on. */
  const choices = (p: EngineModelsPlatform, type: ModelType) =>
    p.models.filter((row) => fits(row.facts, type)).map((row) => row.model);

  /** A model a tier runs a task on without an entry: costs unpriced, no compaction. */
  const unpriced = (p: EngineModelsPlatform, model: string) => !p.models.some((row) => row.model === model);

  return {
    loading: isLoading,
    platforms,
    slot,
    types,
    choices,
    unpriced,
    saving: assign.isPending || setEntry.isPending,
    /** Whether the server refused the last change, and its message, for detail. */
    assignError: assign.error ? (extractRawErrorMessage(assign.error) ?? "") : null,
    entryError: setEntry.error ? (extractRawErrorMessage(setEntry.error) ?? "") : null,
    clearErrors: () => {
      assign.reset();
      setEntry.reset();
    },

    assign: (platform: string, s: ModelSlotView, model: string) =>
      assign.mutateAsync({ platform: platform as never, tier: s.tier, type: s.type, model }),
    resetSlot: (platform: string, s: ModelSlotView) =>
      assign.mutateAsync({ platform: platform as never, tier: s.tier, type: s.type, model: "" }),

    saveEntry: (platform: string, model: string, facts: Omit<ModelFactsOverride, "since">) =>
      setEntry.mutateAsync({ platform: platform as never, model, facts: { ...facts, since: "" } }),
    resetEntry: (platform: string, model: string) =>
      setEntry.mutateAsync({ platform: platform as never, model, facts: undefined }),

    probeable: (type: ModelType) => PROBED.includes(type),
    probe: async (platform: string, type: ModelType, model: string) => {
      const key = probeKey(platform, type, model);
      setProbing(key);
      try {
        const result = await probe.mutateAsync({ platform: platform as never, type, model });
        setProbes((all) => ({ ...all, [key]: result }));
      } catch {
        // The mutation shows the error (no key of one's own on the platform).
      } finally {
        setProbing(null);
      }
    },
    probeResult: (platform: string, type: ModelType, model: string) => probes[probeKey(platform, type, model)],
    isProbing: (platform: string, type: ModelType, model: string) => probing === probeKey(platform, type, model),
  };
}

export type EngineModelsPage = ReturnType<typeof useEngineModelsPage>;

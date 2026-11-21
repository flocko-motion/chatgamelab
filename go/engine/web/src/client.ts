/**
 * The engine's routes, through the client generated from the platform's
 * OpenAPI spec. The platform and the dev server both serve them at
 * /api/engine/…, so the client's base is the page's origin either way.
 */
import { Api, type HttpResponse } from "../../../../frontend/src/api/generated/api.gen.js";
import { PROBLEMS, type Problem, type Refusal } from "./protocol.js";

export type {
  Cell,
  Choice,
  CreateRequest,
  EdgeDetail,
  NodeDetail,
  OfferOption,
  Options,
  Sample,
  Settled,
  Snapshot,
  TierOption,
  Toggles,
  Topology,
  TopologyEdge,
  TopologyNode,
  UsageRecord,
  UsageReport,
} from "../../../../frontend/src/api/generated/api.gen.js";

/** Every route the platform serves; the playback-engine calls the engine's. */
export type EngineClient = Api<unknown>["api"];

/** The client for the engine served at an origin, such as the page's. */
export function engineClient(origin: string | URL): EngineClient {
  const baseUrl = new URL(origin).origin;
  return new Api({ baseUrl }).api;
}

/** A session, and the client that reaches it. */
export interface SessionRef {
  readonly client: EngineClient;
  readonly id: string;
}

/** The address of a session's live WebSocket. */
export function liveURL(session: SessionRef, origin: string | URL): URL {
  const url = new URL(`/api/engine/sessions/${encodeURIComponent(session.id)}/live`, origin);
  url.protocol = url.protocol.replace("http", "ws");
  return url;
}

/**
 * What a route's failure says to the player. The generated client throws the
 * response; its error body carries the problem as its code. Anything else
 * thrown, such as a fetch that could not reach the server, is null.
 */
export async function refusalOf(thrown: unknown): Promise<Refusal | null> {
  if (!(thrown instanceof Response)) return null;
  let body: unknown = (thrown as HttpResponse<unknown, unknown>).error;
  if (!isErrorBody(body)) {
    // A route without a response body leaves its error body unread.
    try {
      body = await thrown.clone().json();
    } catch {
      // Not JSON: a proxy, or something other than the engine, answered.
    }
  }
  if (isErrorBody(body)) return { problem: problemOf(body.code), message: body.error };
  return { problem: "failed", message: `The game server answered ${thrown.status}.` };
}

interface ErrorBody {
  readonly error: string;
  readonly code: string;
}

function isErrorBody(value: unknown): value is ErrorBody {
  if (typeof value !== "object" || value === null) return false;
  const candidate = value as Partial<ErrorBody>;
  return typeof candidate.code === "string" && typeof candidate.error === "string";
}

function problemOf(code: string): Problem {
  return PROBLEMS.find((problem) => problem === code) ?? "failed";
}

/** What a failed call says, for a notice's detail. */
export async function detailOf(thrown: unknown): Promise<string> {
  const refusal = await refusalOf(thrown);
  if (refusal) return refusal.message;
  return thrown instanceof Error ? thrown.message : String(thrown);
}

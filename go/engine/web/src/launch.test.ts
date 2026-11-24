import assert from "node:assert/strict";
import { test } from "node:test";
import type { EngineClient } from "./client.js";
import { create, opening, transientMemory } from "./launch.js";

/** A client whose create answers the session id, recording each request. */
function creating(id: string): { client: EngineClient; onces: string[] } {
  const onces: string[] = [];
  const client = {
    postApiEngineSessions: async (body: { once?: string }) => {
      onces.push(body.once ?? "");
      return { data: { id } };
    },
  } as unknown as EngineClient;
  return { client, onces };
}

test("a session id opens that session", () => {
  assert.deepEqual(opening(new URLSearchParams("session=s1")), { kind: "resume", id: "s1" });
});

test("new with a game or a share token creates a session for it", () => {
  assert.deepEqual(opening(new URLSearchParams("new&game=g1")), {
    kind: "new",
    selector: { kind: "game", id: "g1" },
  });
  assert.deepEqual(opening(new URLSearchParams("new&share=t1")), {
    kind: "new",
    selector: { kind: "share", id: "t1" },
  });
  assert.deepEqual(opening(new URLSearchParams("")), { kind: "nothing" });
});

test("a share embedded on a verified page carries the host's player and request ids", () => {
  assert.deepEqual(opening(new URLSearchParams("new&share=t1&player=p1&request=r1")), {
    kind: "new",
    selector: { kind: "share", id: "t1", player: "p1", request: "r1" },
  });
});

test("creating from an embedded share sends the host's ids along", async () => {
  const bodies: unknown[] = [];
  const client = {
    postApiEngineSessions: async (body: unknown) => {
      bodies.push(body);
      return { data: { id: "s1" } };
    },
  } as unknown as EngineClient;

  await create(client, { kind: "share", id: "t1", player: "p1", request: "r1" }, "en", transientMemory(), "k1");

  assert.deepEqual(bodies, [{ share: "t1", player: "p1", request: "r1", language: "en", once: "k1" }]);
});

test("creating keeps no session memory, and lets the one-time key go once created", async () => {
  const pending = transientMemory();
  const { client } = creating("s1");

  const created = await create(client, { kind: "game", id: "g1" }, "en", pending);

  assert.deepEqual(created, { id: "s1" });
  assert.deepEqual(pending.keys(), []);
});

test("a creation cut short repeats its one-time key", async () => {
  const pending = transientMemory();
  const { client, onces } = creating("s1");
  pending.set("cgl.once.game:g1", "k1");

  await create(client, { kind: "game", id: "g1" }, "en", pending);

  assert.deepEqual(onces, ["k1"]);
});

test("an embed may name live or rounds, which skips the choice", () => {
  assert.deepEqual(opening(new URLSearchParams("new&share=t1&mode=rounds")), {
    kind: "new",
    selector: { kind: "share", id: "t1", mode: "rounds" },
  });
  assert.deepEqual(opening(new URLSearchParams("new&game=g1&mode=live")), {
    kind: "new",
    selector: { kind: "game", id: "g1", mode: "live" },
  });
  // Anything else is no mode, and the player chooses where there is a choice.
  assert.deepEqual(opening(new URLSearchParams("new&game=g1&mode=loud")), {
    kind: "new",
    selector: { kind: "game", id: "g1" },
  });
});

test("a game that plays live and in rounds asks which, then creates in the mode chosen", async () => {
  const bodies: Array<{ mode?: string; once?: string }> = [];
  const client = {
    postApiEngineSessions: async (body: { mode?: string; once?: string }) => {
      bodies.push(body);
      if (!body.mode) {
        throw new Response(JSON.stringify({ error: "choose", code: "choose_mode" }), { status: 422 });
      }
      return { data: { id: "s1" } };
    },
  } as unknown as EngineClient;
  const pending = transientMemory();

  const asked = await create(client, { kind: "game", id: "g1" }, "en", pending);
  assert.deepEqual(asked, { choose: true });

  const created = await create(client, { kind: "game", id: "g1", mode: "rounds" }, "en", pending);
  assert.deepEqual(created, { id: "s1" });
  // One intent: the choice carries on with the same one-time key.
  assert.equal(bodies[1]?.mode, "rounds");
  assert.equal(bodies[0]?.once, bodies[1]?.once);
});

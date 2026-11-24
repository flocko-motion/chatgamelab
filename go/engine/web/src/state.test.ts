import assert from "node:assert/strict";
import { test } from "node:test";
import type { Topology } from "./client.js";
import { emptyState, preparing } from "./state.js";

const topology = {
  name: "adventure",
  nodes: [
    { name: "cast", role: "block" },
    { name: "condense-scenario", role: "block" },
    { name: "image-style", role: "block" },
    { name: "start-game", role: "gate" },
  ],
  edges: [],
} as unknown as Topology;

// While the game prepares, the steps still running are shown, in the
// wiring's order, without the start itself; each drops out as it finishes,
// and none show once the game has begun.
test("preparing lists the blocks still running before the start", () => {
  const state = emptyState();
  state.topology = topology;
  state.phases = { cast: "working", "condense-scenario": "ready", "image-style": "working", "start-game": "working" };
  assert.deepEqual(preparing(state), ["cast", "image-style"]);

  state.phases.cast = "ready";
  assert.deepEqual(preparing(state), ["image-style"]);

  state.started = true;
  state.phases.cast = "working";
  assert.deepEqual(preparing(state), []);
});

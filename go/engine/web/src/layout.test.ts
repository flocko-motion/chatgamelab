import assert from "node:assert/strict";
import { test } from "node:test";
import { layoutCode, parked } from "./layout.js";

test("the copied layout is the Go literal, shifted to the origin and in reading order", () => {
  const code = layoutCode(
    "npc-live",
    new Map([
      ["out-text", { x: 160.4, y: 300 }],
      ["start-game", { x: 330, y: 100 }],
      ["live-session", { x: 160, y: 199.6 }],
      ["portrait", { x: 40, y: 100 }],
    ]),
  );
  assert.equal(
    code,
    [
      "var npcLiveLayout = ports.Layout{",
      '\t"portrait":     {X: 0, Y: 0},',
      '\t"start-game":   {X: 290, Y: 0},',
      '\t"live-session": {X: 120, Y: 100},',
      '\t"out-text":     {X: 120, Y: 200},',
      "}",
      "",
    ].join("\n"),
  );
});

test("unplaced blocks wait in a row of their own below the drawing", () => {
  const places = parked(new Map([["outline", { x: 300, y: 256 }]]), ["new-block", "other"]);
  assert.deepEqual(places.get("new-block"), { x: 0, y: 384 });
  assert.deepEqual(places.get("other"), { x: 150, y: 384 });
});

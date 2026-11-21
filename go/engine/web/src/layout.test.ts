import assert from "node:assert/strict";
import { test } from "node:test";
import { layoutCode, parked } from "./layout.js";

test("the copied layout is the Go literal, shifted to the origin and in reading order", () => {
  const code = layoutCode(
    "npc-live",
    new Map([
      ["out-text", { row: 3, col: 2 }],
      ["start-game", { row: 1, col: 3 }],
      ["live-session", { row: 2, col: 2 }],
      ["portrait", { row: 1, col: 1 }],
    ]),
  );
  assert.equal(
    code,
    [
      "var npcLiveLayout = ports.Layout{",
      '\t"portrait":     {Row: 0, Col: 0},',
      '\t"start-game":   {Row: 0, Col: 2},',
      '\t"live-session": {Row: 1, Col: 1},',
      '\t"out-text":     {Row: 2, Col: 1},',
      "}",
      "",
    ].join("\n"),
  );
});

test("unplaced blocks wait in a row of their own below the drawing", () => {
  const cells = parked(new Map([["outline", { row: 4, col: 2 }]]), ["new-block", "other"]);
  assert.deepEqual(cells.get("new-block"), { row: 6, col: 0 });
  assert.deepEqual(cells.get("other"), { row: 6, col: 1 });
});

import assert from "node:assert/strict";
import { test } from "node:test";
import { handedCSS, sessionAddress, stylesheetURL } from "./embed.js";

test("the session's address keeps the stylesheet and the language", () => {
  const params = new URLSearchParams("new&share=t1&player=p1&request=r1&css=https%3A%2F%2Fa.org%2Fs.css&lang=de");
  assert.equal(sessionAddress("s1", params), "?session=s1&css=https%3A%2F%2Fa.org%2Fs.css&lang=de");
  assert.equal(sessionAddress("s1", new URLSearchParams("new&game=g1")), "?session=s1");
});

test("a host hands CSS over as a cgl-css message of bounded size", () => {
  assert.equal(handedCSS({ type: "cgl-css", css: "body{color:red}" }), "body{color:red}");
  assert.equal(handedCSS({ type: "cgl-size", css: "x" }), null);
  assert.equal(handedCSS({ type: "cgl-css", css: 42 }), null);
  assert.equal(handedCSS("body{}"), null);
  assert.equal(handedCSS({ type: "cgl-css", css: "a".repeat(100_001) }), null);
});

test("a stylesheet from the address must be https", () => {
  assert.equal(stylesheetURL(new URLSearchParams("css=https%3A%2F%2Fa.org%2Fs.css")), "https://a.org/s.css");
  assert.equal(stylesheetURL(new URLSearchParams("css=http%3A%2F%2Fa.org%2Fs.css")), null);
  assert.equal(stylesheetURL(new URLSearchParams("css=javascript%3Aalert(1)")), null);
  assert.equal(stylesheetURL(new URLSearchParams("")), null);
});

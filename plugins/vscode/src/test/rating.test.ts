import { test } from "node:test";
import * as assert from "node:assert/strict";
import { DAY_MS, answered, asked, initial, load, recordProgress, reviewTarget, shouldAsk } from "../rating";

const t0 = Date.UTC(2026, 8, 1, 12);
const day = (n: number) => t0 + n * DAY_MS;
function activeUser() {
  let s = initial(t0);
  for (const n of [1, 3, 5]) s = recordProgress(s, day(n));
  return s;
}

void test("asks only real users, after a week and 3 active days", () => {
  let s = initial(t0);
  s = recordProgress(recordProgress(s, day(1)), day(1)); // same day twice counts once
  assert.equal(s.activeDays.length, 1);
  assert.equal(shouldAsk(activeUser(), day(6)), false, "not before a week");
  assert.equal(shouldAsk(activeUser(), day(7)), true);
  s = recordProgress(recordProgress(initial(t0), day(1)), day(2));
  assert.equal(shouldAsk(s, day(30)), false, "two active days are not enough");
});

void test("rated or never: never again", () => {
  for (const a of ["rate", "never"] as const) {
    const s = answered(asked(activeUser(), day(7)), a);
    assert.equal(shouldAsk(s, day(60)), false, a);
  }
});

void test("later or dismissed: again in a week, at most 3 times", () => {
  let s = asked(activeUser(), day(7));      // shown, closed without an answer
  assert.equal(shouldAsk(s, day(10)), false);
  assert.equal(shouldAsk(s, day(14)), true);
  s = answered(asked(s, day(14)), "later");
  s = asked(s, day(21));                    // third time
  assert.equal(shouldAsk(s, day(60)), false);
});

void test("damaged stored state starts over instead of crashing", () => {
  assert.deepEqual(load(undefined, t0), initial(t0));
  assert.deepEqual(load({ firstSeen: "x" }, t0), initial(t0));
  const s = load({ firstSeen: t0, activeDays: ["2026-09-01", 5], status: "weird", asks: 1 }, t0);
  assert.deepEqual(s.activeDays, ["2026-09-01"]);
  assert.equal(s.status, "pending");
});

void test("reviews go where the editor reads them", () => {
  assert.equal(reviewTarget("VSCodium").store, "Open VSX");
  assert.equal(reviewTarget("Cursor").store, "Open VSX");
  // VS Code goes to Open VSX until the Visual Studio Marketplace listing exists
  assert.ok(reviewTarget("Visual Studio Code").url.startsWith("https://"));
});

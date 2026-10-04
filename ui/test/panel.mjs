// Loads the panel the way the IDE plugins do (same CSP, a nonce for the
// script) with a fake host that records what the panel sends.
import { test as base, expect } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { FIXTURES } from "./fixtures.mjs";

export { expect };
export const COVERAGE = path.join(path.dirname(fileURLToPath(import.meta.url)), ".coverage");

/**
 * Every test records which parts of the panel's script ran (Chromium's
 * block coverage); test/coverage-report.mjs adds them up after the run.
 */
export const test = base.extend({
  panelCoverage: [async ({ page }, use, testInfo) => {
    await page.coverage.startJSCoverage({ resetOnNavigation: false, reportAnonymousScripts: true });
    await use();
    const panel = (await page.coverage.stopJSCoverage()).filter(e => e.source && e.source.includes("window.commitHike = {"));
    if (panel.length) {
      fs.mkdirSync(COVERAGE, { recursive: true });
      fs.writeFileSync(path.join(COVERAGE, `${testInfo.testId}.json`),
        JSON.stringify(panel.map(e => ({ length: e.source.length, functions: e.functions }))));
    }
  }, { auto: true }],
});

const here = path.dirname(fileURLToPath(import.meta.url));
const PANEL = path.join(here, "..", "panel", "panel.html");
const CSP = "default-src 'none'; style-src 'unsafe-inline'; script-src 'nonce-test'; img-src data:";

/** The core's status from the fixtures, a fresh copy each time. */
export function status(lang = "en") {
  return JSON.parse(fs.readFileSync(path.join(FIXTURES, `status-${lang}.json`), "utf8"));
}

/** What a host sends to the panel. */
export function data(st, extra = {}) {
  return { type: "update", state: "ok", repo: "/work/repo", locale: st.locale || "en", status: st, build: { version: "0.4.0", flavor: "prod" }, ...extra };
}

/** Opens the panel with the given data; fails the test on any page error. */
export async function openPanel(page, d, { width = 380, height = 900 } = {}) {
  const errors = [];
  page.on("pageerror", e => errors.push(e.message));
  page.on("console", m => { if (m.type() === "error") errors.push(m.text()); });
  await page.setViewportSize({ width, height });
  await page.clock.setFixedTime(new Date("2026-09-28T13:00:00Z"));
  const html = fs.readFileSync(PANEL, "utf8").replace("{{CSP}}", CSP).replace(/\{\{NONCE\}\}/g, "test");
  await page.setContent(html);
  await page.evaluate(() => {
    window.__msgs = [];
    window.commitHikeHost = { send: m => window.__msgs.push(JSON.parse(m)) };
  });
  await page.evaluate(x => window.commitHike.update(x), d);
  return {
    errors,
    update: next => page.evaluate(x => window.commitHike.update(x), next),
    /** Messages the panel sent to the host, without the "ready" handshake. */
    sent: async () => (await page.evaluate(() => window.__msgs)).filter(m => m.command !== "ready"),
    tab: i => page.getByRole("tab").nth(i).click(),
  };
}

/** Width and height of a PNG from its data URL (read from the IHDR chunk). */
export function pngSize(dataUrl) {
  const b = Buffer.from(dataUrl.split(",")[1], "base64");
  return { png: b.subarray(1, 4).toString() === "PNG", width: b.readUInt32BE(16), height: b.readUInt32BE(20) };
}

// Which Chromium the panel tests run in. Playwright's own build when it's
// installed; otherwise a Chrome or Chromium already on this computer, so the
// tests work even where Playwright's download doesn't (unsupported Linux
// versions, offline machines). PLAYWRIGHT_CHROMIUM=/path/to/chrome picks one.
import { chromium } from "@playwright/test";
import { execFileSync } from "node:child_process";
import fs from "node:fs";

export function bundledChromium() {
  try {
    const p = chromium.executablePath();
    return p && fs.existsSync(p) ? p : null;
  } catch {
    return null;
  }
}

export function systemChrome() {
  const env = process.env.PLAYWRIGHT_CHROMIUM;
  if (env) return fs.existsSync(env) ? env : null;
  const candidates = {
    darwin: ["/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "/Applications/Chromium.app/Contents/MacOS/Chromium"],
    win32: [`${process.env.PROGRAMFILES}\\Google\\Chrome\\Application\\chrome.exe`, `${process.env.LOCALAPPDATA}\\Google\\Chrome\\Application\\chrome.exe`],
  }[process.platform];
  if (candidates) return candidates.find(p => fs.existsSync(p)) || null;
  // Linux: a snap Chromium comes last, its sandbox sometimes gets in the way
  for (const name of ["google-chrome-stable", "google-chrome", "chromium", "chromium-browser"]) {
    try {
      const p = execFileSync("which", [name], { encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] }).trim();
      if (p) return p;
    } catch { /* not installed */ }
  }
  return null;
}

/** The browser to use, or null when there is none yet. */
export function browserPath() {
  return process.env.PLAYWRIGHT_CHROMIUM ? systemChrome() : bundledChromium() || systemChrome();
}

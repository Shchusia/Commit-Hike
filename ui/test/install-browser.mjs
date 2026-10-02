// `task ui:deps`: makes sure there is a Chromium for the panel tests.
// Uses one already installed when possible; downloads Playwright's at most
// once, and never waits forever.
import { spawnSync } from "node:child_process";
import { browserPath, bundledChromium } from "./browser.mjs";

const found = browserPath();
if (found) {
  console.log(`Panel tests run in ${found}`);
  process.exit(0);
}
console.log("Downloading Chromium for the panel tests (once, about 170 MB)…");
const args = ["playwright", "install", ...(process.env.CI ? ["--with-deps"] : []), "chromium"];
const r = spawnSync("npx", args, { stdio: "inherit", timeout: 5 * 60 * 1000, shell: process.platform === "win32" });
if (r.status === 0 && bundledChromium()) process.exit(0);
console.error(`
${r.error && r.error.code === "ETIMEDOUT" ? "The Chromium download didn't finish in 5 minutes." : "Installing Chromium for the panel tests failed."}
Either install Google Chrome or Chromium (for example: sudo apt install chromium),
or point PLAYWRIGHT_CHROMIUM at a Chrome binary, then run task ui:test again.`);
process.exit(1);

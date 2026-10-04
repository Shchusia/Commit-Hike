// Runs once before the tests: asks the real core for statuses, so the panel is
// tested against what the core actually sends (a protocol change shows up here).
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
export const FIXTURES = path.join(here, ".fixtures");

export default function setup() {
  const bin = process.env.COMMIT_HIKE_BIN || path.join(here, "..", "..", ".sandbox", "bin", "commit-hike");
  if (!fs.existsSync(bin)) throw new Error(`core binary not found at ${bin}: run through \`task ui:test\`, which builds it`);
  const home = fs.mkdtempSync(path.join(os.tmpdir(), "ch-ui-home-"));
  const repo = fs.mkdtempSync(path.join(os.tmpdir(), "ch-ui-repo-"));
  const env = { ...process.env, COMMIT_HIKE_HOME: home, GIT_CONFIG_NOSYSTEM: "1", HOME: home };
  const git = (args, extra = {}) => execFileSync("git", ["-C", repo, ...args], { env: { ...env, ...extra }, stdio: "pipe" });
  const core = (...args) => JSON.parse(execFileSync(bin, args, { env, encoding: "utf8" }));

  // A week of commits, a few a day, ending today: mid-route on Chornohora.
  git(["init", "-q"]);
  const now = Math.floor(Date.now() / 1000);
  let n = 0;
  for (let d = 6; d >= 0; d--) {
    for (let i = 0; i < 2 + (d % 2); i++) {
      const t = now - d * 86400 - 3 * 3600 + i * 900;
      fs.appendFileSync(path.join(repo, `f${i}.go`), Array.from({ length: 12 + ((d * 7 + i * 5) % 20) }, (_, k) => `line ${n} ${k}\n`).join(""));
      git(["add", "-A"]);
      git(["-c", "user.email=me@x.io", "-c", "user.name=Me", "commit", "-qm", `c${n++}`], { GIT_AUTHOR_DATE: `@${t} +0000`, GIT_COMMITTER_DATE: `@${t} +0000` });
    }
  }
  core("init", "--email", "me@x.io", "--route", "chornohora-ridge");
  core("scan", "--repo", repo);
  fs.rmSync(FIXTURES, { recursive: true, force: true });
  fs.mkdirSync(FIXTURES, { recursive: true });
  for (const lang of ["en", "uk", "pl", "de", "es"]) {
    const status = core("status", "--repo", repo, "--lang", lang);
    if (!status.ok) throw new Error("status failed: " + JSON.stringify(status));
    fs.writeFileSync(path.join(FIXTURES, `status-${lang}.json`), JSON.stringify(status.data));
  }
  fs.rmSync(home, { recursive: true, force: true });
  fs.rmSync(repo, { recursive: true, force: true });
}

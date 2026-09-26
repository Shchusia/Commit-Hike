// Runs with `npm test` (node --test). Uses the real core binary from ./bin.
import { test } from "node:test";
import * as assert from "node:assert/strict";
import { execFileSync } from "child_process";
import * as fs from "fs";
import * as os from "os";
import * as path from "path";
import { Cli, CliError, bundledBinary, ensureExecutable, formatDistance } from "../cli";

const ext = path.join(__dirname, "..", "..");

void test("bundledBinary maps Node platform/arch to Go names", () => {
  assert.equal(path.basename(bundledBinary(ext, "win32", "x64")), "commit-hike-windows-amd64.exe");
  assert.equal(path.basename(bundledBinary(ext, "darwin", "arm64")), "commit-hike-darwin-arm64");
  assert.equal(path.basename(bundledBinary(ext, "linux", "x64")), "commit-hike-linux-amd64");
  assert.throws(() => bundledBinary(ext, "freebsd", "x64"), /doesn't support/);
});

void test("formatDistance", () => {
  assert.equal(formatDistance(42.4), "42 m");
  assert.equal(formatDistance(8400), "8.4 km");
  assert.equal(formatDistance(250000), "250 km");
});

void test("Cli talks to the real core end to end", async () => {
  const home = fs.mkdtempSync(path.join(os.tmpdir(), "ct-home-"));
  const repo = fs.mkdtempSync(path.join(os.tmpdir(), "ct-repo-"));
  process.env.COMMIT_HIKE_HOME = home;
  const git = (...a: string[]) => execFileSync("git", ["-C", repo, ...a], {
    env: { ...process.env, GIT_AUTHOR_NAME: "T", GIT_AUTHOR_EMAIL: "me@x.io", GIT_COMMITTER_NAME: "T", GIT_COMMITTER_EMAIL: "me@x.io" },
  });
  git("init", "-q");
  fs.writeFileSync(path.join(repo, "a.go"), "x\ny\nz\n");
  git("add", "-A");
  git("commit", "-qm", "c");

  const bin = bundledBinary(ext);
  fs.chmodSync(bin, 0o644); // simulate a VSIX that lost the exec bit
  ensureExecutable(bin);
  const cli = new Cli(bin);

  await assert.rejects(cli.status(), (e: CliError) => e.notInitialized);
  await cli.init({ emails: ["me@x.io"], mode: "all", fromHistory: true });

  // concurrent calls are queued, not failing on the lock
  const [a, b] = await Promise.all([cli.scan(repo), cli.scan(repo)]);
  assert.equal(a.new_commits + b.new_commits, 1);
  assert.equal(a.total_m, 50);

  cli.lang = "uk";
  const st = await cli.status(repo);
  assert.equal(st.global?.distance_m, 50);
  assert.equal(st.locale, "uk");
  assert.equal(st.global?.route.name, "Демо-стежка");

  const routes = await cli.routes();
  const ridge = routes.find(r => r.id === "chornohora-ridge");
  assert.ok(ridge && ridge.waypoints.length > 0);
  await cli.setJourney("project", ridge.id, true, repo);
  assert.equal((await cli.status(repo)).project?.route.name, "Чорногірський хребет");

  await assert.rejects(cli.setJourney("global", "atlantis", false), (e: CliError) => e.code === "unknown_route");

  const bad = await cli.scan(os.tmpdir());
  assert.equal(bad.tracked, false);
});

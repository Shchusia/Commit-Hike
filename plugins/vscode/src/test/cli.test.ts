// Runs with `npm test` (node --test). Uses the real core binary from ./bin.
import { test } from "node:test";
import * as assert from "node:assert/strict";
import { execFileSync } from "child_process";
import * as fs from "fs";
import * as os from "os";
import * as path from "path";
import { Cli, CliError, bundledBinary, ensureExecutable, formatDistance } from "../cli";

const ext = path.join(__dirname, "..", "..");
// Every git call gets its own minute: commits within one second share a dedup
// key (author email + time) and would count as one.
let clock = Math.floor(Date.now() / 1000) - 6 * 3600;
const dates = () => { clock += 60; return { GIT_AUTHOR_DATE: `@${clock} +0000`, GIT_COMMITTER_DATE: `@${clock} +0000` }; };

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
    env: { ...process.env, ...dates(), GIT_AUTHOR_NAME: "T", GIT_AUTHOR_EMAIL: "me@x.io", GIT_COMMITTER_NAME: "T", GIT_COMMITTER_EMAIL: "me@x.io" },
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
  assert.equal(a.total_m, 650); // 3 lines = 50 base points × 13 (medium)

  cli.lang = "uk";
  const st = await cli.status(repo);
  assert.equal(st.global?.distance_m, 650);
  assert.equal(st.locale, "uk");
  assert.equal(st.global?.route.name, "Демо-стежка");

  const routes = await cli.routes();
  const ridge = routes.find(r => r.id === "chornohora-ridge");
  assert.ok(ridge && ridge.waypoints.length > 0);
  await cli.setJourney("project", ridge.id, true, repo);
  assert.equal((await cli.status(repo)).project?.route.name, "Чорногірський хребет");

  await assert.rejects(cli.setJourney("global", "atlantis", false), (e: CliError) => e.code === "unknown_route");

  // Map data and daily stats
  const proj = (await cli.status(repo)).project!;
  const hoverla = proj.route.waypoints.find(w => w.id === "hoverla")!;
  assert.equal(hoverla.kind, "peak");
  assert.equal(hoverla.elevation_m, 2061);
  assert.ok((proj.route.biomes ?? []).length > 0);
  assert.equal(proj.daily.length, 14);

  // Custom routes: template -> import -> remove
  const work = fs.mkdtempSync(path.join(os.tmpdir(), "ch-routes-"));
  const dir = await cli.routeTemplate("my-trail", work);
  const imported = await cli.importRoute(dir, false);
  assert.equal(imported.name, "Моя стежка");
  await assert.rejects(cli.importRoute(dir, false), (e: CliError) => e.code === "route_exists");
  await cli.importRoute(dir, true);
  await cli.removeRoute("my-trail");

  // Hiker icon: the default figure shipped with the panel is itself a valid custom icon.
  assert.equal((await cli.avatar()).custom, false);
  const icon = path.join(__dirname, "..", "..", "..", "..", "ui", "panel", "hiker-default.png");
  const av = await cli.setAvatar(icon);
  assert.ok(av.custom && av.data_url?.startsWith("data:image/png;base64,"));
  await assert.rejects(cli.setAvatar(path.join(work, "my-trail", "route.json")), (e: CliError) => e.code === "invalid_image");
  await cli.resetAvatar();
  assert.equal((await cli.avatar()).custom, false);
  assert.ok(!(await cli.routes()).some(r => r.id === "my-trail"));

  const bad = await cli.scan(os.tmpdir());
  assert.equal(bad.tracked, false);
});

void test("Cli: rewrites, team, language and route assets", async () => {
  const home = fs.mkdtempSync(path.join(os.tmpdir(), "ct-home2-"));
  const repo = fs.mkdtempSync(path.join(os.tmpdir(), "ct-repo2-"));
  process.env.COMMIT_HIKE_HOME = home;
  const git = (email: string, ...a: string[]) => execFileSync("git", ["-C", repo, ...a], {
    env: { ...process.env, ...dates(), GIT_AUTHOR_NAME: email.split("@")[0], GIT_AUTHOR_EMAIL: email, GIT_COMMITTER_NAME: "T", GIT_COMMITTER_EMAIL: email },
  }).toString().trim();
  const commit = (email: string, file: string, lines: number) => {
    fs.appendFileSync(path.join(repo, file), "x\n".repeat(lines));
    git(email, "add", "-A");
    git(email, "commit", "-qm", "c");
  };
  git("me@x.io", "init", "-q");
  commit("other@x.io", "README", 1);
  commit("me@x.io", "a.go", 50);
  commit("me@x.io", "b.go", 50);
  commit("anna@x.io", "c.go", 3);

  const cli = new Cli(bundledBinary(ext));
  await cli.init({ emails: ["me@x.io"], mode: "all", fromHistory: true });
  await cli.scan(repo);
  const before = git("me@x.io", "rev-parse", "HEAD");
  git("me@x.io", "reset", "-q", "--hard", "HEAD~3"); // back to the root: my commits are gone
  const res = await cli.scan(repo, before);
  assert.equal(res.rewritten, true);
  assert.equal(res.total_m, 0);

  commit("me@x.io", "d.go", 3);
  commit("anna@x.io", "e.go", 3);
  await cli.setTeam(repo, true);
  assert.equal((await cli.status(repo)).team, true);
  const team = await cli.team(repo);
  assert.equal(team.members.length, 3); // me, anna and the root commit's author
  assert.ok(team.members.some(m => m.me) && team.members.some(m => m.name === "anna"));

  const loc = await cli.locale("uk");
  assert.equal(loc.effective, "uk");
  assert.equal((await cli.status(repo)).locale, "uk");
  assert.equal((await cli.locale("auto")).locale, "");

  const assets = await cli.routeAssets("molfar-path");
  assert.ok(assets.images["assets/campfire.png"].startsWith("data:image/png;base64,"));
  assert.ok(assets.html["assets/fireflies.html"].includes("glade"));
});

void test("Cli: difficulty", async () => {
  process.env.COMMIT_HIKE_HOME = fs.mkdtempSync(path.join(os.tmpdir(), "ct-home3-"));
  const cli = new Cli(bundledBinary(ext));
  await cli.init({ emails: ["me@x.io"], mode: "all", fromHistory: true, difficulty: "hard" });
  const info = await cli.difficulty();
  assert.equal(info.level, "hard");
  assert.ok(info.typical_day_m.easy > info.typical_day_m.medium && info.typical_day_m.medium > info.typical_day_m.hard);
  assert.equal((await cli.difficulty("easy")).level, "easy");
  assert.equal((await cli.status()).difficulty, "easy");
});

void test("Cli: days off, history and backups", async () => {
  process.env.COMMIT_HIKE_HOME = fs.mkdtempSync(path.join(os.tmpdir(), "ct-home4-"));
  const cli = new Cli(bundledBinary(ext));
  await cli.init({ emails: ["me@x.io"], mode: "all", fromHistory: true });
  assert.deepEqual((await cli.restDays("sat,sun")).days, [0, 6]);
  const st = await cli.status() as { rest_days?: number[]; history?: unknown[] };
  assert.deepEqual(st.rest_days, [0, 6]);
  assert.ok(Array.isArray(st.history ?? []));

  const file = path.join(fs.mkdtempSync(path.join(os.tmpdir(), "ct-backup-")), "backup.json");
  assert.equal((await cli.exportBackup(file)).path, file);
  await assert.rejects(cli.importBackup(file, false), (e: unknown) => e instanceof CliError && e.code === "data_exists");
  process.env.COMMIT_HIKE_HOME = fs.mkdtempSync(path.join(os.tmpdir(), "ct-home5-"));
  const fresh = new Cli(bundledBinary(ext));
  await fresh.importBackup(file, false);
  assert.deepEqual((await fresh.restDays()).days, [0, 6]);
});

void test("Cli: settings", async () => {
  process.env.COMMIT_HIKE_HOME = fs.mkdtempSync(path.join(os.tmpdir(), "ct-home6-"));
  const cli = new Cli(bundledBinary(ext));
  await cli.init({ emails: ["me@x.io"], mode: "all", fromHistory: true });
  assert.deepEqual(await cli.settings(), { reduce_motion: "auto", high_contrast: "auto", notifications: "all" });
  assert.equal((await cli.settings({ notifications: "off", high_contrast: "on" })).notifications, "off");
  assert.deepEqual((await cli.status()).settings, { reduce_motion: "auto", high_contrast: "on", notifications: "off" });
  await assert.rejects(cli.settings({ reduce_motion: "sometimes" as never }), (e: unknown) => e instanceof CliError && e.code === "invalid_argument");
});

void test("Cli: badge and diagnostics", async () => {
  process.env.COMMIT_HIKE_HOME = fs.mkdtempSync(path.join(os.tmpdir(), "ct-home7-"));
  const cli = new Cli(bundledBinary(ext));
  await cli.init({ emails: ["me@x.io"], mode: "all", fromHistory: true });
  const b = await cli.badge();
  assert.ok(b.svg.startsWith("<svg") && b.svg.includes("Commit Hike"));
  assert.equal(b.file_name, "commit-hike-badge.svg");
  const d = await cli.diagnostics();
  assert.equal(d.initialized, true);
  assert.ok(!JSON.stringify(d).includes("me@x.io"), "nothing personal in diagnostics");
});

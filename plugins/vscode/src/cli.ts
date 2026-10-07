// Thin wrapper around the commit-hike core binary. No VS Code APIs here, so this
// file is unit-testable with plain Node.
import { execFile } from "child_process";
import * as fs from "fs";
import * as path from "path";

// Types mirror core/internal/protocol (api 1). Unknown fields are ignored,
// so newer cores with extra fields keep working.
export const API_VERSION = 1;

export interface Waypoint {
  id: string; name: string; text?: string; at_m: number; kind?: string; elevation_m?: number;
  lat?: number; lon?: number; // real position, on routes with a GPS track
  x?: number; y?: number;     // position on a drawn map (0..1), on routes with a path
}
export interface Biome { at_m: number; type: string }
export interface Day { date: string; m: number }
export interface Story { id: string; text: string; at_m: number }
export interface Achievement { id: string; name?: string; description?: string; hidden?: boolean; unlocked_at?: number }
export type Level = "easy" | "medium" | "hard";
export interface DifficultyInfo { level: Level; levels: Level[]; typical_day_m: Record<Level, number> }
export interface RestDaysInfo { days: number[] } // 0 = Sunday … 6 = Saturday
export type AutoOnOff = "auto" | "on" | "off";
/** Panel and notification choices, shared by every IDE on this computer. */
export interface Settings { reduce_motion: AutoOnOff; high_contrast: AutoOnOff; notifications: "all" | "milestones" | "off"; festive: "on" | "off"; ambient: "on" | "off" }
export interface BackupResult { path: string; created_at: number; commits: number; routes: number; avatar: boolean }
export interface Span { from_m: number; to_m: number }
export interface Danger { id: string; at_m: number; text: string }
export interface ProfilePoint { at_m: number; elevation_m: number }
export interface Fact { id: string; at_m: number; text: string }
export interface RouteObject {
  id: string; at_m: number; asset: string; height_px: number; lift_px?: number; offset_m?: number; fade_m: number;
  layer: "trail" | "far"; caption?: string;
}
export interface Route {
  id: string; name: string; description: string; length_m: number;
  locales: string[]; builtin: boolean; waypoints: Waypoint[]; biomes?: Biome[]; achievements: number;
  profile?: ProfilePoint[]; ascent_m?: number; min_elevation_m?: number; max_elevation_m?: number;
  facts?: Fact[]; objects?: RouteObject[]; path?: [number, number][]; map_image?: string;
  underground?: Span[]; dangers?: Danger[]; track?: number[][]; loop?: boolean;
}
export interface NextRoute { id: string; name: string; length_m: number; series_id: string; series_name: string }
export interface Journey {
  next_route?: NextRoute; // the next route of this route's series
  scope: "global" | "project"; route: Route; distance_m: number; percent: number; finished: boolean;
  last_waypoint?: Waypoint; next_waypoint?: Waypoint; to_next_m?: number; story?: Story;
  achievements: Achievement[]; commits: number; streak_days: number; daily: Day[]; day: number;
  elevation_m?: number; ascent_m?: number; max_elevation_m?: number; to_next_climb_m?: number; underground?: boolean;
}
export interface Status {
  tracked: boolean; reason?: string; reason_code?: "not_a_repo" | "not_enabled"; team?: boolean;
  difficulty?: Level; typical_day_m?: number;
  locale: string; global?: Journey; project?: Journey; today_m: number; total_m: number;
  settings?: Settings;
}
export type EventType = "waypoint" | "story" | "achievement" | "finished" | "fact" | "danger";
export interface JourneyEvent {
  type: EventType; journey: "global" | "project"; waypoint?: Waypoint; story?: Story; achievement?: Achievement; fact?: Fact;
  danger?: Danger;
}
export interface ScanResult extends Status {
  new_commits: number; updated_commits: number; removed_commits?: number; rewritten?: boolean; added_m: number; events?: JourneyEvent[];
}
export interface Avatar { custom: boolean; data_url?: string }
export interface VerifyResult { added: number; updated: number; removed: number }
export interface Config { mode: "all" | "selected"; emails: string[]; locale?: string }
export interface Member {
  id: string; name: string; me?: boolean; distance_m: number; percent: number; finished?: boolean;
  commits: number; today_m: number; last_commit_at: number; elevation_m?: number;
}
export interface WeekMember { id: string; name: string; me?: boolean; distance_m: number; commits: number; active_days: number }
export interface TeamWeek {
  from: string; to: string; total_m: number; commits: number; active_days: number; prev_total_m: number;
  best_day?: string; best_day_m?: number; members: WeekMember[]; hidden?: number; days_elapsed: number;
}
export interface GoalMember { id: string; name: string; me?: boolean; distance_m: number }
export interface TeamGoal {
  route_id: string; route_name: string; length_m: number; since: number; distance_m: number; percent: number; finished?: boolean;
  last_waypoint?: Waypoint; next_waypoint?: Waypoint; to_next_m?: number; pace_m: number; days_left?: number; members: GoalMember[];
}
export interface RouteChoice { id: string; name: string; length_m: number }
export interface Team {
  scope: "global" | "project"; route_id: string; length_m: number; members: Member[]; hidden?: number;
  week: TeamWeek; goal?: TeamGoal; routes: RouteChoice[];
}
export interface LocaleInfo { locale: string; effective: string; available: string[] }
export interface RouteAssets { id: string; images: Record<string, string>; html: Record<string, string> }

/** Every core response: {"api":1,"ok":true,"data":…} or {"api":1,"ok":false,"error":…}. */
interface Envelope { api?: number; ok?: boolean; data?: unknown; error?: { code: string; message: string } }

export class CliError extends Error {
  constructor(message: string, readonly code?: string) { super(message); }
  get notInitialized(): boolean { return this.code === "not_initialized"; }
}

/** Picks the bundled binary for this OS/CPU, e.g. bin/commit-hike-darwin-arm64. */
export function bundledBinary(extensionPath: string, platform = process.platform, arch = process.arch): string {
  const os = platform === "win32" ? "windows" : platform;
  const cpu = arch === "x64" ? "amd64" : arch;
  if (!["linux", "darwin", "windows"].includes(os) || !["amd64", "arm64"].includes(cpu)) {
    throw new Error(`Commit Hike doesn't support ${platform}/${arch} yet.`);
  }
  return path.join(extensionPath, "bin", `commit-hike-${os}-${cpu}${os === "windows" ? ".exe" : ""}`);
}

/** VSIX archives don't keep the executable bit, so restore it once. */
export function ensureExecutable(file: string): void {
  if (!fs.existsSync(file)) throw new Error(`Commit Hike binary not found at ${file}`);
  if (process.platform !== "win32") {
    const mode = fs.statSync(file).mode;
    if ((mode & 0o111) === 0) fs.chmodSync(file, 0o755);
  }
}

export class Cli {
  // Calls are queued so one window never contends with itself for the core's
  // file lock; other windows/IDEs are handled by the core's lock.
  private queue: Promise<unknown> = Promise.resolve();

  constructor(private readonly binary: string, private readonly timeoutMs = 60_000) {}

  run<T>(args: string[]): Promise<T> {
    const job = this.queue.then(() => this.exec<T>(args));
    this.queue = job.catch(() => undefined);
    return job;
  }

  private exec<T>(args: string[]): Promise<T> {
    return new Promise((resolve, reject) => {
      execFile(this.binary, args, { timeout: this.timeoutMs, maxBuffer: 16 * 1024 * 1024, windowsHide: true },
        (err, stdout, stderr) => {
          let env: Envelope;
          try {
            env = JSON.parse(stdout) as Envelope;
          } catch {
            return reject(new CliError(err ? `${err.message} ${stderr}`.trim() : "The core returned invalid output."));
          }
          if (env.api !== API_VERSION) {
            return reject(new CliError(`Core speaks protocol ${env.api}, this plugin expects ${API_VERSION}.`, "protocol_mismatch"));
          }
          if (!env.ok) return reject(new CliError(env.error?.message ?? "Unknown error", env.error?.code));
          resolve(env.data as T);
        });
    });
  }

  /** Language for route texts; VS Code gives tags like "uk" or "en-us". */
  lang = "en";

  private withLang(args: string[]): string[] { return [...args, "--lang", this.lang]; }

  config() { return this.run<Config>(["config"]); }
  routes() { return this.run<Route[]>(this.withLang(["routes"])); }
  status(repo?: string) { return this.run<Status>(this.withLang(repo ? ["status", "--repo", repo] : ["status"])); }
  /** prevHead: the HEAD seen before this change; if it's no longer an ancestor the core recounts in full. */
  scan(repo: string, prevHead?: string) {
    return this.run<ScanResult>(this.withLang(["scan", "--repo", repo, ...(prevHead ? ["--prev-head", prevHead] : [])]));
  }
  team(repo: string) { return this.run<Team>(this.withLang(["team", "--repo", repo])); }
  /** The route the project's team walks together, from now; "" removes it. */
  setTeamGoal(repo: string, route: string) {
    return this.run<{ goal: string }>(route ? ["project", "goal", "--repo", repo, "--route", route] : ["project", "goal-off", "--repo", repo]);
  }
  setTeam(repo: string, on: boolean) { return this.run<{ team: boolean }>(["project", on ? "team-on" : "team-off", "--repo", repo]); }
  /** Reads the language setting; with set ("auto", "en", "uk"…) changes it for every IDE. */
  locale(set?: string) { return this.run<LocaleInfo>(this.withLang(set === undefined ? ["locale"] : ["locale", "--set", set])); }
  routeAssets(id: string) { return this.run<RouteAssets>(["route", "assets", "--id", id]); }
  /** Reads the difficulty; with set, changes it for commits from now on. */
  difficulty(set?: Level) { return this.run<DifficultyInfo>(set ? ["difficulty", "--set", set] : ["difficulty"]); }
  /** Weekdays off; set is like "sat,sun" or "none". */
  restDays(set?: string) { return this.run<RestDaysInfo>(set ? ["rest-days", "--set", set] : ["rest-days"]); }
  /** Changes the given settings and returns them all. */
  settings(change: Partial<Settings> = {}) {
    const args = Object.entries(change).filter(([, v]) => v).map(([k, v]) => `--${k.replace(/_/g, "-")}=${v}`);
    return this.run<Settings>(["settings", ...args]);
  }
  /** An SVG badge for a README, for the journey the panel shows. */
  badge(repo?: string) {
    return this.run<{ svg: string; file_name: string; markdown: string }>(this.withLang(["badge", ...(repo ? ["--repo", repo] : [])]));
  }
  /** Versions and counts for a bug report: nothing personal. */
  diagnostics() { return this.run<Record<string, unknown>>(["diagnostics"]); }
  exportBackup(path: string) { return this.run<BackupResult>(["backup", "export", "--path", path]); }
  /** Rejects with CliError code "data_exists" when there is progress and replace is false. */
  importBackup(path: string, replace: boolean) {
    return this.run<BackupResult>(["backup", "import", "--path", path, ...(replace ? ["--replace"] : [])]);
  }
  verify(repo: string) { return this.run<VerifyResult>(["verify", "--repo", repo]); }

  init(o: { emails: string[]; mode: "all" | "selected"; fromHistory: boolean; difficulty?: Level }) {
    return this.run<Config>(["init", "--email", o.emails.join(","), "--mode", o.mode, `--from-history=${o.fromHistory}`,
      ...(o.difficulty ? ["--difficulty", o.difficulty] : [])]);
  }

  setJourney(scope: "global" | "project", route: string, fromHistory: boolean, repo?: string) {
    const args = ["journey", "--scope", scope, "--route", route, `--from-history=${fromHistory}`];
    if (repo) args.push("--repo", repo);
    return this.run<Status>(this.withLang(args));
  }

  importRoute(path: string, replace: boolean) {
    return this.run<Route>(this.withLang(["route", "import", "--path", path, ...(replace ? ["--replace"] : [])]));
  }
  removeRoute(id: string) { return this.run<{ removed: string }>(["route", "remove", "--id", id]); }

  /** Searches the routes website. Outside the queue: it waits on the network, not on the core's data. */
  siteRoutes(args: string[]) { return this.exec<unknown>(this.withLang(args)); }
  /** Installs (or updates) a route from the website. */
  siteInstall(id: string, replaceLocal: boolean) {
    return this.exec<{ route: Route; version: number; page: string }>(
      this.withLang(["site", "install", "--id", id, ...(replaceLocal ? ["--replace-local"] : [])]));
  }
  /** Writes a template route pack to dir/id and returns its folder. */
  async routeTemplate(id: string, dir: string) {
    return (await this.run<{ path: string }>(["route", "template", "--id", id, "--path", dir])).path;
  }

  avatar() { return this.run<Avatar>(["avatar", "get"]); }
  setAvatar(pngPath: string) { return this.run<Avatar>(["avatar", "set", "--path", pngPath]); }
  resetAvatar() { return this.run<{ custom: boolean }>(["avatar", "reset"]); }

  setProjectEnabled(repo: string, on: boolean) {
    return this.run<{ enabled: boolean }>(["project", on ? "enable" : "disable", "--repo", repo]);
  }
}

/** `git config --global user.email`, or "" if git or the setting is missing. */
export function gitGlobalEmail(): Promise<string> {
  return new Promise(resolve => {
    execFile("git", ["config", "--global", "user.email"], { windowsHide: true },
      (err, stdout) => resolve(err ? "" : stdout.trim()));
  });
}

export function formatDistance(m: number): string {
  if (m < 1000) return `${Math.round(m)} m`;
  const km = m / 1000;
  return `${km < 100 ? km.toFixed(1) : Math.round(km)} km`;
}

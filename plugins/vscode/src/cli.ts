// Thin wrapper around the commit-hike core binary. No VS Code APIs here, so this
// file is unit-testable with plain Node.
import { execFile } from "child_process";
import * as fs from "fs";
import * as path from "path";

// Types mirror core/internal/protocol (api 1). Unknown fields are ignored,
// so newer cores with extra fields keep working.
export const API_VERSION = 1;

export interface Waypoint { id: string; name: string; text?: string; at_m: number }
export interface Story { id: string; text: string; at_m: number }
export interface Achievement { id: string; name?: string; description?: string; hidden?: boolean; unlocked_at?: number }
export interface Route {
  id: string; name: string; description: string; length_m: number;
  locales: string[]; builtin: boolean; waypoints: Waypoint[]; achievements: number;
}
export interface Journey {
  scope: "global" | "project"; route: Route; distance_m: number; percent: number; finished: boolean;
  last_waypoint?: Waypoint; next_waypoint?: Waypoint; to_next_m?: number; story?: Story;
  achievements: Achievement[]; commits: number; streak_days: number;
}
export interface Status {
  tracked: boolean; reason?: string; locale: string; global?: Journey; project?: Journey; today_m: number; total_m: number;
}
export type EventType = "waypoint" | "story" | "achievement" | "finished";
export interface JourneyEvent {
  type: EventType; journey: "global" | "project"; waypoint?: Waypoint; story?: Story; achievement?: Achievement;
}
export interface ScanResult extends Status {
  new_commits: number; updated_commits: number; added_m: number; events?: JourneyEvent[];
}
export interface VerifyResult { added: number; updated: number; removed: number }
export interface Config { mode: "all" | "selected"; emails: string[]; locale?: string }

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
  scan(repo: string) { return this.run<ScanResult>(this.withLang(["scan", "--repo", repo])); }
  verify(repo: string) { return this.run<VerifyResult>(["verify", "--repo", repo]); }

  init(o: { emails: string[]; mode: "all" | "selected"; fromHistory: boolean }) {
    return this.run<Config>(["init", "--email", o.emails.join(","), "--mode", o.mode, `--from-history=${o.fromHistory}`]);
  }

  setJourney(scope: "global" | "project", route: string, fromHistory: boolean, repo?: string) {
    const args = ["journey", "--scope", scope, "--route", route, `--from-history=${fromHistory}`];
    if (repo) args.push("--repo", repo);
    return this.run<Status>(this.withLang(args));
  }

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

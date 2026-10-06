import * as fs from "fs";
import * as os from "os";
import * as path from "path";
import * as vscode from "vscode";
import {
  Avatar, Cli, CliError, Level, LocaleInfo, Route, RouteAssets, ScanResult, Settings, Status, Team, bundledBinary, ensureExecutable, gitGlobalEmail,
} from "./cli";
import { LANGUAGE_NAMES, formatDistanceL as formatDistance, language, setLanguage, t } from "./i18n";
import { PanelMessage, TrailPanel } from "./panel";
import * as rating from "./rating";
import { RepoTracker } from "./repos";
import { scrubReport } from "./report";
import { isShareUrl, pngBytes } from "./share";
import { isRouteId, isSitePage, siteSearchArgs } from "./site";

export async function activate(ctx: vscode.ExtensionContext): Promise<void> {
  let binary: string;
  try {
    binary = vscode.workspace.getConfiguration("commitHike").get<string>("binaryPath") || bundledBinary(ctx.extensionPath);
    ensureExecutable(binary);
  } catch (e) {
    setLanguage(vscode.env.language);
    void vscode.window.showErrorMessage(t("cantStart", (e as Error).message));
    return;
  }
  const cli = new Cli(binary, 5 * 60_000); // first scan of a huge repo can be slow
  cli.lang = vscode.env.language; // route texts in the editor's language
  const app = new App(ctx, cli);
  await app.start();
}

export function deactivate(): void {}

class App {
  private readonly log = vscode.window.createOutputChannel("Commit Hike");
  private readonly statusItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 50);
  private readonly panel: TrailPanel;
  private readonly tracker: RepoTracker;
  private routes: Record<string, Route> = {};
  private avatar: Avatar = { custom: false };
  private initialized = false;
  private currentRepo?: string;
  private lastStatus?: Status;
  private flashTimer?: NodeJS.Timeout;
  private localeInfo?: LocaleInfo;
  private readonly assets = new Map<string, RouteAssets>(); // pictures of route objects and maps, per route
  private team?: { repo: string; data?: Team; error?: string; at: number; loading?: boolean };

  constructor(private readonly ctx: vscode.ExtensionContext, private readonly cli: Cli) {
    setLanguage(vscode.env.language);
    this.panel = new TrailPanel(ctx.extensionPath, m => void this.onPanelMessage(m));
    // The last HEAD seen per repository survives restarts, so a rebase done
    // while VS Code was closed is still recognised as a rewrite.
    const heads = "commitHike.heads";
    this.tracker = new RepoTracker((root, prev) => void this.scan(root, prev), {
      get: root => ctx.workspaceState.get<Record<string, string>>(heads)?.[root],
      set: (root, head) => void ctx.workspaceState.update(heads, { ...ctx.workspaceState.get<Record<string, string>>(heads), [root]: head }),
    });
  }

  async start(): Promise<void> {
    const { ctx } = this;
    this.statusItem.command = "commitHike.showTrail";
    this.statusItem.name = "Commit Hike";
    this.statusItem.show();

    ctx.subscriptions.push(
      this.log, this.statusItem, this.tracker,
      vscode.window.registerWebviewViewProvider(TrailPanel.viewId, this.panel),
      vscode.commands.registerCommand("commitHike.showTrail", () => this.showTrail()),
      vscode.commands.registerCommand("commitHike.setup", () => this.guard(() => this.setup())),
      vscode.commands.registerCommand("commitHike.chooseRoute", () => this.guard(() => this.chooseRoute("global"))),
      vscode.commands.registerCommand("commitHike.chooseProjectRoute", () => this.guard(() => this.chooseRoute("project"))),
      vscode.commands.registerCommand("commitHike.enableProject", () => this.guard(() => this.setProjectEnabled(true))),
      vscode.commands.registerCommand("commitHike.disableProject", () => this.guard(() => this.setProjectEnabled(false))),
      vscode.commands.registerCommand("commitHike.verify", () => this.guard(() => this.verify())),
      vscode.commands.registerCommand("commitHike.importRoute", () => this.guard(() => this.importRoute())),
      vscode.commands.registerCommand("commitHike.findRoutes", () => this.openSite()),
      vscode.commands.registerCommand("commitHike.createRouteTemplate", () => this.guard(() => this.createRouteTemplate())),
      vscode.commands.registerCommand("commitHike.removeRoute", () => this.guard(() => this.removeRoute())),
      vscode.commands.registerCommand("commitHike.setHikerIcon", () => this.guard(() => this.setHikerIcon())),
      vscode.commands.registerCommand("commitHike.resetHikerIcon", () => this.guard(() => this.resetHikerIcon())),
      vscode.commands.registerCommand("commitHike.changeLanguage", () => this.guard(() => this.changeLanguage())),
      vscode.commands.registerCommand("commitHike.changeDifficulty", () => this.guard(() => this.changeDifficulty())),
      vscode.commands.registerCommand("commitHike.setRestDays", () => this.guard(() => this.setRestDays())),
      vscode.commands.registerCommand("commitHike.openSettings", () => this.openSettings()),
      vscode.commands.registerCommand("commitHike.exportProgress", () => this.guard(() => this.exportProgress())),
      vscode.commands.registerCommand("commitHike.importProgress", () => this.guard(() => this.importProgress())),
      vscode.commands.registerCommand("commitHike.toggleTeam", () => this.guard(() => this.setTeam(!this.lastStatus?.team))),
      vscode.window.onDidChangeActiveTextEditor(() => this.followActiveRepo()),
      this.tracker.onDidChangeRepos(() => this.followActiveRepo(true)),
    );

    try {
      this.routes = Object.fromEntries((await this.cli.routes()).map(r => [r.id, r]));
      this.avatar = await this.cli.avatar();
      await this.cli.config();
      this.initialized = true;
      await this.loadLocale();
    } catch (e) {
      if (!(e instanceof CliError && e.notInitialized)) this.report(e);
    }
    await this.tracker.start();
    await this.refresh();
    if (!this.initialized) void this.offerSetupOnce();
  }

  // ---------- scanning & rendering ----------

  private async scan(root: string, prevHead?: string): Promise<void> {
    if (!this.initialized) return;
    try {
      const res = await this.cli.scan(root, prevHead);
      if (this.team?.repo === root) this.team = undefined; // teammates may have moved too
      this.celebrate(res);
    } catch (e) {
      this.report(e);
    }
    await this.refresh();
  }

  private async loadLocale(): Promise<void> {
    try {
      this.localeInfo = await this.cli.locale();
      setLanguage(this.localeInfo.effective);
    } catch (e) {
      this.report(e);
    }
  }

  private celebrate(res: ScanResult): void {
    const cfg = vscode.workspace.getConfiguration("commitHike");
    // all | milestones (stops, achievements, the finish) | off: chosen on the settings page
    const notifications = this.lastStatus?.settings?.notifications ?? "all";
    if (res.added_m > 0 && notifications === "all" && cfg.get<boolean>("showDistanceAfterCommit", true)) {
      clearTimeout(this.flashTimer);
      this.statusItem.text = `🥾 +${formatDistance(res.added_m)}`;
      this.flashTimer = setTimeout(() => { this.flashTimer = undefined; this.renderStatusBar(); }, 4000);
    }
    if (res.rewritten && (res.removed_commits ?? 0) > 0) this.log.appendLine(t("rewritten"));
    if (res.added_m > 0) this.maybeAskForRating();
    if (notifications === "off" || !cfg.get<boolean>("notifyOnWaypoints", true)) return;
    const events = res.events ?? [];
    // Achievements are always worth a notification of their own.
    for (const e of events.filter(e => e.type === "achievement" && e.achievement)) {
      void vscode.window.showInformationMessage(`🏅 ${e.achievement!.name}: ${e.achievement!.description ?? ""}`, t("showTrail"))
        .then(a => { if (a) this.showTrail(); });
    }
    // Importing history can pass many stops at once: announce only the latest per journey.
    for (const journey of ["global", "project"] as const) {
      const evs = events.filter(e => e.journey === journey);
      const j = journey === "global" ? res.global : res.project;
      if (j && evs.some(e => e.type === "finished")) {
        const next = j.next_route;
        if (next) { // a series goes on: offer the next route
          void vscode.window.showInformationMessage(`${t("finished", j.route.name)} ${t("seriesNextText", next.series_name, next.name)}`, t("walkOn", next.name), t("chooseTrail"))
            .then(a => {
              if (a === t("walkOn", next.name)) void this.guard(() => this.walkRoute(next.id));
              else if (a) void this.chooseRoute(journey);
            });
        } else {
          void vscode.window.showInformationMessage(t("finished", j.route.name), t("chooseTrail"))
            .then(a => { if (a) void this.chooseRoute(journey); });
        }
        continue;
      }
      // An encounter on the road is announced when it's the only news of this commit.
      const danger = evs.filter(e => e.type === "danger").pop()?.danger;
      if (danger && evs.length <= 3 && notifications === "all") {
        void vscode.window.showWarningMessage(`⚠ ${danger.text}`, t("showTrail")).then(a => { if (a) this.showTrail(); });
      }
      const w = evs.filter(e => e.type === "waypoint").pop()?.waypoint;
      if (w) {
        void vscode.window.showInformationMessage(w.text ? `${w.name}: ${w.text}` : t("reached", w.name), t("showTrail"))
          .then(a => { if (a) this.showTrail(); });
      }
    }
  }

  /**
   * After a commit that moved the user along: once they've used Commit Hike for
   * a while, ask for a rating (see rating.ts for when and how often).
   */
  private maybeAskForRating(): void {
    const key = "commitHike.rating";
    const now = Date.now();
    let st = rating.recordProgress(rating.load(this.ctx.globalState.get(key), now), now);
    if (!rating.shouldAsk(st, now)) {
      void this.ctx.globalState.update(key, st);
      return;
    }
    st = rating.asked(st, now); // closing the message without an answer counts as "later"
    void this.ctx.globalState.update(key, st);
    const target = rating.reviewTarget(vscode.env.appName);
    // A few seconds later, so it doesn't pile onto the distance/waypoint notifications.
    setTimeout(() => {
      void vscode.window.showInformationMessage(t("rateAsk", target.store), t("rateYes"), t("rateLater"), t("rateNever"))
        .then(async choice => {
          const answer = choice === t("rateYes") ? "rate" : choice === t("rateNever") ? "never" : "later";
          await this.ctx.globalState.update(key, rating.answered(st, answer));
          if (answer === "rate") await vscode.env.openExternal(vscode.Uri.parse(target.url));
        });
    }, 4000);
  }

  private async refresh(): Promise<void> {
    if (!this.initialized) {
      this.lastStatus = undefined;
      this.renderStatusBar();
      this.panel.update({ type: "update", state: "not_initialized", locale: vscode.env.language });
      return;
    }
    try {
      this.lastStatus = await this.cli.status(this.currentRepo);
      await this.loadAssets(this.lastStatus);
      this.renderStatusBar();
      this.postPanel();
      if (this.lastStatus.team && this.currentRepo) void this.loadTeam(this.currentRepo);
    } catch (e) {
      if (e instanceof CliError && e.notInitialized) { this.initialized = false; return this.refresh(); }
      this.report(e);
      this.panel.update({ type: "update", state: "error", locale: language(), error: (e as Error).message });
    }
  }

  /** Version and flavor written by scripts/prepare.js at build time. */
  private buildInfo?: BuildInfo;
  private get build(): BuildInfo { return (this.buildInfo ??= readBuildInfo(this.ctx.extensionPath)); }

  private openView?: { view: string; token: number };

  /** Commit Hike: Settings…: show the trail view on its settings page. */
  private openSettings(): void {
    this.openView = { view: "settings", token: Date.now() };
    this.showTrail();
    this.postPanel();
  }

  private postPanel(): void {
    const s = this.lastStatus;
    if (!s) return;
    const team = this.team && this.team.repo === this.currentRepo ? this.team : undefined;
    const ids = [s.global?.route.id, s.project?.route.id].filter((x): x is string => !!x);
    this.panel.update({
      type: "update", state: "ok", repo: this.currentRepo, locale: s.locale, status: s,
      avatar: this.avatar.data_url, avatar_custom: this.avatar.custom,
      assets: Object.fromEntries(ids.filter(id => this.assets.has(id)).map(id => [id, this.assets.get(id)!])),
      team: s.team ? team?.data : undefined, team_error: s.team ? team?.error : undefined,
      locale_setting: this.localeInfo,
      // A dev build (task ... FLAVOR=dev), F5/tests or COMMIT_HIKE_DEV=1 may look ahead along the trail.
      dev: this.build.flavor === "dev" || this.ctx.extensionMode !== vscode.ExtensionMode.Production || process.env.COMMIT_HIKE_DEV === "1",
      build: this.build,
      open_view: this.openView?.view, open_token: this.openView?.token,
    });
  }

  /** Route pictures change only on import, so they're fetched once per route. */
  private async loadAssets(s: Status): Promise<void> {
    for (const id of [s.global?.route.id, s.project?.route.id]) {
      if (!id || this.assets.has(id)) continue;
      try {
        this.assets.set(id, await this.cli.routeAssets(id));
      } catch (e) {
        this.report(e);
      }
    }
  }

  /** Teammates are counted from the whole history: cached until the next scan or for five minutes. */
  /** Sets (or with "" removes) the team goal from the Team tab, then recounts the team. */
  private async setTeamGoal(route: unknown): Promise<void> {
    const repo = this.currentRepo;
    if (!repo || typeof route !== "string" || (route !== "" && !/^[a-z0-9]+(-[a-z0-9]+)*$/.test(route))) return;
    await this.cli.setTeamGoal(repo, route);
    await this.loadTeam(repo, true);
  }

  private async loadTeam(repo: string, force = false): Promise<void> {
    const cur = this.team;
    if (!force && cur && cur.repo === repo && (cur.loading || Date.now() - cur.at < 5 * 60_000)) return;
    this.team = { repo, at: Date.now(), loading: true, data: cur?.repo === repo ? cur.data : undefined };
    try {
      this.team = { repo, at: Date.now(), data: await this.cli.team(repo) };
    } catch (e) {
      this.report(e);
      this.team = { repo, at: Date.now(), error: (e as Error).message };
    }
    if (repo === this.currentRepo) this.postPanel();
  }

  private renderStatusBar(): void {
    if (this.flashTimer) return; // "+86 m" is still showing
    const s = this.lastStatus;
    const g = s?.global;
    if (!this.initialized || !s || !g) {
      this.statusItem.text = "🥾 Commit Hike";
      this.statusItem.tooltip = t("barSetup");
      return;
    }
    this.statusItem.text = g.finished
      ? `🥾 ${formatDistance(g.distance_m)} ✓`
      : `🥾 ${formatDistance(g.distance_m)} / ${formatDistance(g.route.length_m)}`;
    const md = new vscode.MarkdownString();
    md.appendMarkdown(`**${escape(g.route.name)}**: ${t("barOf", formatDistance(g.distance_m), formatDistance(g.route.length_m))} (${g.percent}%)\n\n`);
    if (g.finished) md.appendMarkdown(t("barCompleted") + "\n\n");
    else if (g.next_waypoint) md.appendMarkdown(t("barNext", escape(g.next_waypoint.name), formatDistance(g.to_next_m ?? 0)) + "\n\n");
    if (g.elevation_m !== undefined) md.appendMarkdown(t("barAltitude", Math.round(g.elevation_m), Math.round(g.ascent_m ?? 0)) + "\n\n");
    md.appendMarkdown(t("barToday", formatDistance(s.today_m)));
    if (this.currentRepo && s.project) {
      md.appendMarkdown("\n\n" + t("barProject", `**${escape(s.project.route.name)}**`, `${formatDistance(s.project.distance_m)} (${s.project.percent}%)`));
    } else if (this.currentRepo && !s.tracked) {
      md.appendMarkdown("\n\n" + t("barNotCounted"));
    }
    this.statusItem.tooltip = md;
  }

  private followActiveRepo(force = false): void {
    const repo = this.tracker.repoFor(vscode.window.activeTextEditor?.document.uri)
      ?? (this.currentRepo && this.tracker.repositories.includes(this.currentRepo) ? this.currentRepo : this.tracker.repositories[0]);
    if (force || repo !== this.currentRepo) {
      this.currentRepo = repo;
      void this.refresh();
    }
  }

  // ---------- commands ----------

  private showTrail(): void {
    void vscode.commands.executeCommand(`${TrailPanel.viewId}.focus`);
  }

  private async offerSetupOnce(): Promise<void> {
    const key = "commitHike.setupOffered";
    if (this.ctx.globalState.get<boolean>(key)) return;
    await this.ctx.globalState.update(key, true);
    const a = await vscode.window.showInformationMessage(
      t("offer"), t("setUp"), t("notNow"));
    if (a === t("setUp")) await this.guard(() => this.setup());
  }

  private async setup(): Promise<void> {
    const title = t("setupTitle");
    const mode = await vscode.window.showQuickPick([
      { label: t("modeAll"), detail: t("modeAllDetail"), value: "all" as const },
      { label: t("modeSelected"), detail: t("modeSelectedDetail"), value: "selected" as const },
    ], { title: `${title} (1/4)`, placeHolder: t("modeQuestion"), ignoreFocusOut: true });
    if (!mode) return;

    const history = await vscode.window.showQuickPick([
      { label: t("histYes"), detail: t("histYesDetail"), value: true },
      { label: t("histNo"), detail: t("histNoDetail"), value: false },
    ], { title: `${title} (2/4)`, placeHolder: t("histQuestion"), ignoreFocusOut: true });
    if (!history) return;

    const level = await vscode.window.showQuickPick(this.levelItems("medium"),
      { title: `${title} (3/4)`, placeHolder: t("diffQuestion"), ignoreFocusOut: true });
    if (!level) return;

    const emails = await vscode.window.showInputBox({
      title: `${title} (4/4)`,
      prompt: t("emailsPrompt"),
      value: await gitGlobalEmail(),
      ignoreFocusOut: true,
      validateInput: v => (v.split(",").some(e => e.includes("@")) ? undefined : t("emailsInvalid")),
    });
    if (emails === undefined) return;

    await this.cli.init({ emails: emails.split(",").map(e => e.trim()).filter(Boolean), mode: mode.value, fromHistory: history.value, difficulty: level.value });
    this.initialized = true;
    await this.loadLocale();
    this.routes = Object.fromEntries((await this.cli.routes()).map(r => [r.id, r]));

    if (mode.value === "selected" && this.currentRepo) {
      const a = await vscode.window.showInformationMessage(t("countThis"), t("countIt"), t("notNow"));
      if (a === t("countIt")) await this.cli.setProjectEnabled(this.currentRepo, true);
    }
    await vscode.window.withProgress(
      { location: vscode.ProgressLocation.Window, title: t("counting") },
      async () => { for (const r of this.tracker.repositories) await this.scan(r); });
    await this.refresh();
    this.showTrail();
  }

  private async chooseRoute(scope: "global" | "project"): Promise<void> {
    if (!this.initialized) return this.setup();
    const repo = scope === "project" ? this.currentRepo : undefined;
    if (scope === "project" && !repo) {
      void vscode.window.showInformationMessage(t("openRepoForTrail"));
      return;
    }
    type Item = vscode.QuickPickItem & { id: string };
    const items: Item[] = Object.values(this.routes).map(r => ({
      id: r.id, label: r.name, description: formatDistance(r.length_m), detail: r.description,
    }));
    if (scope === "project" && this.lastStatus?.project) {
      items.push({ id: "none", label: t("noProjectTrail"), detail: t("noProjectTrailDetail") });
    }
    items.push(
      { id: "", label: "", kind: vscode.QuickPickItemKind.Separator },
      { id: "@site", label: t("siteItem") },
      { id: "@import", label: t("importItem") },
      { id: "@template", label: t("templateItem") },
    );
    const pick = await vscode.window.showQuickPick(items, {
      title: scope === "global" ? t("trailAll") : t("trailProject"), placeHolder: t("chooseTrail"),
    });
    if (!pick) return;
    if (pick.id === "@site") return this.openSite();
    if (pick.id === "@import") return this.importRoute();
    if (pick.id === "@template") return this.createRouteTemplate();
    let fromHistory = false;
    if (pick.id !== "none") {
      const h = await vscode.window.showQuickPick([
        { label: t("histYes"), value: true },
        { label: t("histNow"), value: false },
      ], { title: pick.label, placeHolder: t("journeyStart") });
      if (!h) return;
      fromHistory = h.value;
    }
    await this.cli.setJourney(scope, pick.id, fromHistory, repo);
    await this.refresh();
  }

  private async setProjectEnabled(on: boolean): Promise<void> {
    if (!this.initialized) return this.setup();
    const repo = this.currentRepo;
    if (!repo) {
      void vscode.window.showInformationMessage(t("openRepoFirst"));
      return;
    }
    const cfg = await this.cli.config();
    if (cfg.mode === "all") {
      void vscode.window.showInformationMessage(t("allCounted"));
      return;
    }
    await this.cli.setProjectEnabled(repo, on);
    if (on) await this.scan(repo);
    else await this.refresh();
  }

  private async verify(): Promise<void> {
    const repo = this.currentRepo;
    if (!this.initialized || !repo) return;
    const r = await vscode.window.withProgress(
      { location: vscode.ProgressLocation.Notification, title: t("recounting") },
      () => this.cli.verify(repo));
    await this.refresh();
    const changes = r.added + r.updated + r.removed;
    this.team = undefined;
    void vscode.window.showInformationMessage(changes === 0 ? t("recountSame") : t("recountDone", r.added, r.updated, r.removed));
  }

  // ---------- custom routes ----------

  private async importRoute(): Promise<void> {
    if (!this.initialized) return this.setup();
    // Linux and Windows dialogs can pick files or folders, not both: ask first.
    const kind = await vscode.window.showQuickPick([
      { label: t("fromZip"), folders: false },
      { label: t("fromFolder"), folders: true },
    ], { title: t("importTitle"), placeHolder: t("importWhere") });
    if (!kind) return;
    const picked = await vscode.window.showOpenDialog({
      title: t("importTitle"), openLabel: t("importBtn"), canSelectMany: false,
      canSelectFiles: !kind.folders, canSelectFolders: kind.folders,
      filters: kind.folders ? undefined : { [t("routePack")]: ["zip"] },
    });
    const src = picked?.[0]?.fsPath;
    if (!src) return;
    let route;
    try {
      route = await this.cli.importRoute(src, false);
    } catch (e) {
      if (!(e instanceof CliError) || e.code !== "route_exists") throw e;
      const a = await vscode.window.showWarningMessage(t("replaceQ", e.message), { modal: true }, t("replace"));
      if (a !== t("replace")) return;
      route = await this.cli.importRoute(src, true);
    }
    await this.reloadRoutes();
    const a = await vscode.window.showInformationMessage(
      t("imported", route.name, formatDistance(route.length_m), route.waypoints.length), t("walkNow"));
    if (a) {
      await this.cli.setJourney("global", route.id, false);
      await this.refresh();
      this.showTrail();
    }
  }

  private async createRouteTemplate(): Promise<void> {
    const folder = await vscode.window.showOpenDialog({
      title: t("whereCreate"), openLabel: t("createHere"), canSelectFolders: true, canSelectFiles: false,
    });
    if (!folder?.[0]) return;
    const id = await vscode.window.showInputBox({
      title: t("newRoute"), prompt: t("routeIdPrompt"), value: "my-trail",
      validateInput: v => (/^[a-z0-9]+(-[a-z0-9]+)*$/.test(v.trim()) ? undefined : t("routeIdInvalid")),
    });
    if (!id) return;
    const dir = await this.cli.routeTemplate(id.trim(), folder[0].fsPath);
    await vscode.window.showTextDocument(vscode.Uri.file(`${dir}/route.json`));
    const a = await vscode.window.showInformationMessage(
      t("templateCreated"), t("importNow"));
    if (a) await this.importRoute();
  }

  private async removeRoute(): Promise<void> {
    const custom = Object.values(this.routes).filter(r => !r.builtin);
    if (custom.length === 0) {
      void vscode.window.showInformationMessage(t("noCustom"));
      return;
    }
    const pick = await vscode.window.showQuickPick(
      custom.map(r => ({ label: r.name, description: formatDistance(r.length_m), id: r.id })),
      { title: t("removeTitle"), placeHolder: t("removeHint") });
    if (!pick) return;
    await this.cli.removeRoute(pick.id);
    await this.reloadRoutes();
    void vscode.window.showInformationMessage(t("removed", pick.label));
  }

  // ---------- hiker icon ----------

  private async setHikerIcon(): Promise<void> {
    const picked = await vscode.window.showOpenDialog({
      title: t("iconTitle"),
      openLabel: t("useAsHiker"), canSelectMany: false, filters: { [t("pngImage")]: ["png"] },
    });
    if (!picked?.[0]) return;
    this.avatar = await this.cli.setAvatar(picked[0].fsPath);
    await this.refresh();
  }

  private async resetHikerIcon(): Promise<void> {
    await this.cli.resetAvatar();
    this.avatar = { custom: false };
    await this.refresh();
  }

  private async reloadRoutes(): Promise<void> {
    this.routes = Object.fromEntries((await this.cli.routes()).map(r => [r.id, r]));
    this.assets.clear(); // an imported route may have replaced its pictures
  }

  // ---------- language & team ----------

  private async changeLanguage(): Promise<void> {
    if (!this.initialized) return this.setup();
    const info = this.localeInfo ?? await this.cli.locale();
    type Item = vscode.QuickPickItem & { value: string };
    const langs = [...new Set(["en", "uk", ...info.available])];
    const items: Item[] = [{ label: t("langAuto"), value: "auto", description: info.locale === "" ? t("langCurrent") : undefined },
      ...langs.map(l => ({ label: LANGUAGE_NAMES[l] ?? l, value: l, description: info.locale === l ? t("langCurrent") : undefined }))];
    const pick = await vscode.window.showQuickPick(items, { title: t("langTitle") });
    if (pick) await this.setLocale(pick.value);
  }

  private async setLocale(value: string): Promise<void> {
    this.localeInfo = await this.cli.locale(value);
    setLanguage(this.localeInfo.effective);
    await this.reloadRoutes(); // route names come translated
    await this.refresh();
  }

  private levelItems(current?: Level): (vscode.QuickPickItem & { value: Level })[] {
    const factor: Record<Level, number> = { easy: 1.25, medium: 1, hard: 0.8 };
    const day = 10030; // a typical day on medium, see score.TypicalDay in the core
    return (["easy", "medium", "hard"] as Level[]).map(l => ({
      label: t(`diff_${l}`), value: l, description: current === l ? t("langCurrent") : undefined,
      detail: t("diffDetail", formatDistance(day * factor[l])),
    }));
  }

  private async changeDifficulty(): Promise<void> {
    if (!this.initialized) return this.setup();
    const cur = (await this.cli.difficulty()).level;
    const pick = await vscode.window.showQuickPick(this.levelItems(cur), { title: t("diffTitle"), placeHolder: t("diffHint") });
    if (pick) await this.setDifficulty(pick.value);
  }

  private async setRestDays(): Promise<void> {
    const current = (await this.cli.restDays()).days;
    const names = ["sun", "mon", "tue", "wed", "thu", "fri", "sat"];
    const items = [1, 2, 3, 4, 5, 6, 0].map(d => ({
      // 2026-10-04 is a Sunday: day d of that week, named in the user's language
      label: new Date(Date.UTC(2026, 9, 4 + d)).toLocaleDateString(language(), { weekday: "long", timeZone: "UTC" }),
      picked: current.includes(d), day: d,
    }));
    items.forEach(i => { i.label = i.label.charAt(0).toUpperCase() + i.label.slice(1); });
    const picked = await vscode.window.showQuickPick(items, { canPickMany: true, title: t("restTitle"), placeHolder: t("restNone") });
    if (!picked) return;
    if (picked.length === 7) {
      void vscode.window.showWarningMessage(t("restAllDays"));
      return;
    }
    await this.cli.restDays(picked.length ? picked.map(p => names[p.day]).join(",") : "none");
    await this.refresh();
  }

  private async exportProgress(): Promise<void> {
    const target = await vscode.window.showSaveDialog({
      title: t("backupSaveTitle"), filters: { [t("backupFile")]: ["json"] },
      defaultUri: vscode.Uri.file(path.join(os.homedir(), `commit-hike-backup-${new Date().toISOString().slice(0, 10)}.json`)),
    });
    if (!target) return;
    const res = await this.cli.exportBackup(target.fsPath);
    void vscode.window.showInformationMessage(t("backupSaved", res.path));
  }

  private async importProgress(): Promise<void> {
    const picked = await vscode.window.showOpenDialog({ title: t("backupOpenTitle"), canSelectMany: false, filters: { [t("backupFile")]: ["json"] } });
    if (!picked?.[0]) return;
    let res;
    try {
      res = await this.cli.importBackup(picked[0].fsPath, false);
    } catch (e) {
      if (!(e instanceof CliError) || e.code !== "data_exists") throw e;
      const ok = await vscode.window.showWarningMessage(t("backupReplaceQ"), { modal: true }, t("replaceBtn"));
      if (ok !== t("replaceBtn")) return;
      res = await this.cli.importBackup(picked[0].fsPath, true);
    }
    await this.reloadRoutes();
    this.avatar = await this.cli.avatar();
    await this.refresh();
    void vscode.window.showInformationMessage(t("backupRestored", res.commits));
  }

  private async setDifficulty(level: Level): Promise<void> {
    await this.cli.difficulty(level);
    await this.refresh();
  }

  private async setTeam(on: boolean): Promise<void> {
    const repo = this.currentRepo;
    if (!this.initialized || !repo) {
      void vscode.window.showInformationMessage(t("teamNeedsRepo"));
      return;
    }
    await this.cli.setTeam(repo, on);
    this.team = undefined;
    await this.refresh();
    void vscode.window.showInformationMessage(on ? t("teamOn") : t("teamOff"));
  }

  private async onPanelMessage(m: PanelMessage): Promise<void> {
    switch (m.command) {
      case "setup": return this.guard(() => this.setup());
      case "refresh": return this.refresh();
      case "enableProject": return this.guard(() => this.setProjectEnabled(true));
      case "setAvatar": return this.guard(() => this.setHikerIcon());
      case "resetAvatar": return this.guard(() => this.resetHikerIcon());
      case "chooseRoute": return this.guard(() => this.chooseRoute(m.scope));
      case "setLocale": return this.guard(() => this.setLocale(m.locale));
      case "setTeam": return this.guard(() => this.setTeam(m.on));
      case "setDifficulty": return this.guard(() => this.setDifficulty(m.level));
      case "setTeamGoal": return this.guard(() => this.setTeamGoal(m.route));
      case "requestTeam": if (this.currentRepo && this.lastStatus?.team) void this.loadTeam(this.currentRepo); return;
      case "importRoute": return this.guard(() => this.importRoute());
      case "createRouteTemplate": return this.guard(() => this.createRouteTemplate());
      case "verify": return this.guard(() => this.verify());
      case "savePostcard": return this.guard(() => this.savePostcard(m.name, m.data));
      case "setRestDays": return this.guard(async () => {
        const names = ["sun", "mon", "tue", "wed", "thu", "fri", "sat"];
        const days = [...new Set((m.days ?? []).filter(d => Number.isInteger(d) && d >= 0 && d <= 6))];
        await this.cli.restDays(days.length ? days.map(d => names[d]).join(",") : "none");
        await this.refresh();
      });
      case "setSettings": return this.guard(async () => {
        await this.cli.settings({ reduce_motion: m.reduce_motion, high_contrast: m.high_contrast, notifications: m.notifications } as Partial<Settings>);
        await this.refresh();
      });
      case "exportProgress": return this.guard(() => this.exportProgress());
      case "copyDiagnostics": return this.guard(() => this.copyDiagnostics());
      case "saveBadge": return this.guard(() => this.saveBadge());
      case "openUrl": return this.guard(() => this.openShareUrl(m.url));
      case "copyText": return this.guard(async () => { await vscode.env.clipboard.writeText(String(m.text).slice(0, 4000)); });
      case "copyImage": return this.guard(() => this.postcardForPost(m.data));
      case "panelError": this.noteError("panel: " + String(m.message).slice(0, 500)); return;
      case "walkRoute": return this.guard(() => this.walkRoute(String(m.id)));
      case "siteSearch": return void this.siteSearch(m.query);
      case "siteInstall": return void this.siteInstall(m.id, m.replaceLocal === true);
      case "startRoute": return this.guard(() => this.startRoute(m.id));
      case "importProgress": return this.guard(() => this.importProgress());
    }
  }

  /** Opens a network's sharing page; only the ones isShareUrl knows. */
  /** The panel's "Routes from the site": search through the core, answer the panel. */
  private async siteSearch(query: unknown): Promise<void> {
    try {
      this.panel.site({ kind: "page", page: await this.cli.siteRoutes(siteSearchArgs(query)) });
    } catch (e) {
      this.panel.site({ kind: "error", error: { code: e instanceof CliError ? e.code : "internal", message: String((e as Error).message ?? e) } });
    }
  }

  private async siteInstall(id: unknown, replaceLocal: boolean): Promise<void> {
    if (!isRouteId(id)) return;
    try {
      const got = await this.cli.siteInstall(id, replaceLocal);
      this.panel.site({ kind: "installed", id, version: got.version });
      await this.refresh();
    } catch (e) {
      this.panel.site({ kind: "installError", id, error: { code: e instanceof CliError ? e.code : "internal", message: String((e as Error).message ?? e) } });
    }
  }

  /** Walks a route just installed from the site: like choosing a trail, asks whether to count past commits. */
  private async startRoute(id: unknown): Promise<void> {
    if (!isRouteId(id)) return;
    const h = await vscode.window.showQuickPick([
      { label: t("histYes"), value: true },
      { label: t("histNow"), value: false },
    ], { title: this.routes[id]?.name ?? id, placeHolder: t("journeyStart") });
    if (!h) return;
    await this.cli.setJourney("global", id, h.value);
    await this.refresh();
    this.showTrail();
  }

  /** Commit Hike: Routes from the Site…: the trail view on its site page. */
  private openSite(): void {
    this.openView = { view: "site", token: Date.now() };
    this.showTrail();
    this.postPanel();
  }

  private async openShareUrl(url: unknown): Promise<void> {
    if (isSitePage(url)) {
      await vscode.env.openExternal(vscode.Uri.parse(url, true));
      return;
    }
    if (!isShareUrl(url)) {
      this.noteError("share: refused a URL that isn't a sharing page");
      return;
    }
    await vscode.env.openExternal(vscode.Uri.parse(url, true));
  }

  /**
   * The postcard for a post, when the panel can't put it into the clipboard.
   * VS Code's clipboard API takes text only, so the picture goes into a
   * temporary file, shown in the file manager, ready to attach.
   */
  private async postcardForPost(dataUrl: unknown): Promise<void> {
    const png = pngBytes(dataUrl);
    if (!png) return;
    const dir = await fs.promises.mkdtemp(path.join(os.tmpdir(), "commit-hike-"));
    const file = path.join(dir, "commit-hike-postcard.png");
    await fs.promises.writeFile(file, png);
    const show = t("showFile");
    if (await vscode.window.showInformationMessage(t("postcardForPost", file), show) === show) {
      await vscode.commands.executeCommand("revealFileInOS", vscode.Uri.file(file));
    }
  }

  /** Saves the postcard the panel drew (a PNG data URL) where the user chooses. */
  private async savePostcard(name: string, dataUrl: string): Promise<void> {
    const bytes = pngBytes(dataUrl);
    if (!bytes) return;
    const file = path.basename(String(name || "commit-hike.png")).replace(/[^\w.-]/g, "-");
    const target = await vscode.window.showSaveDialog({
      title: t("postcardSaveTitle"), filters: { PNG: ["png"] },
      defaultUri: vscode.Uri.file(path.join(os.homedir(), file.endsWith(".png") ? file : file + ".png")),
    });
    if (!target) return;
    await fs.promises.writeFile(target.fsPath, bytes);
    if (await vscode.window.showInformationMessage(t("postcardSaved", target.fsPath), t("openBtn"))) await vscode.env.openExternal(target);
  }

  // ---------- errors ----------

  private async guard(fn: () => Promise<void>): Promise<void> {
    try {
      await fn();
    } catch (e) {
      this.report(e);
      void vscode.window.showErrorMessage(t("error", (e as Error).message));
    }
  }

  private report(e: unknown): void {
    const msg = (e as Error)?.message ?? String(e);
    this.log.appendLine(`[${new Date().toISOString()}] ${msg}`);
    this.noteError(msg);
  }

  /** The last errors, for the developer report. */
  private readonly recentErrors: string[] = [];
  private noteError(msg: string): void {
    this.recentErrors.push(`${new Date().toISOString()} ${msg}`.slice(0, 600));
    if (this.recentErrors.length > 10) this.recentErrors.shift();
  }

  /** Walks on to [id], usually the next route of a series. */
  private async walkRoute(id: string): Promise<void> {
    if (!/^[a-z0-9]+(-[a-z0-9]+)*$/.test(id)) return;
    await this.cli.setJourney("global", id, false);
    await this.refresh();
    this.showTrail();
  }

  /** Saves an SVG badge with the trail, for a README such as a GitHub profile. */
  private async saveBadge(): Promise<void> {
    const badge = await this.cli.badge(this.currentRepo);
    const target = await vscode.window.showSaveDialog({
      title: t("badgeSaveTitle"), filters: { SVG: ["svg"] },
      defaultUri: vscode.Uri.file(path.join(os.homedir(), badge.file_name)),
    });
    if (!target) return;
    await fs.promises.writeFile(target.fsPath, badge.svg);
    const markdown = badge.markdown.replace(badge.file_name, path.basename(target.fsPath));
    if (await vscode.window.showInformationMessage(t("badgeSaved", target.fsPath), t("copyMarkdown"))) {
      await vscode.env.clipboard.writeText(markdown);
    }
  }

  /** "Copy a report for the developer": versions, settings and recent errors, nothing personal. */
  private async copyDiagnostics(): Promise<void> {
    const core = await this.cli.diagnostics().catch((e: unknown) => ({ error: (e as Error)?.message ?? String(e) }));
    const report = scrubReport([
      "Commit Hike: report for the developer",
      `Plugin: ${this.build.version} (${this.build.flavor}), VS Code extension`,
      `Editor: ${vscode.env.appName} ${vscode.version} (${vscode.env.uiKind === vscode.UIKind.Web ? "web" : "desktop"}${vscode.env.remoteName ? ", remote: " + vscode.env.remoteName : ""})`,
      `OS: ${process.platform} ${process.arch}`,
      "", "Core:", "```json", JSON.stringify(core, null, 2), "```",
      "", "Recent errors:", ...(this.recentErrors.length ? this.recentErrors.map(e => "- " + e) : ["- none"]),
    ].join("\n"), os.homedir());
    await vscode.env.clipboard.writeText(report);
    const issues = (this.ctx.extension.packageJSON as { bugs?: { url?: string } }).bugs?.url;
    if (await vscode.window.showInformationMessage(t("reportCopied"), ...(issues ? [t("openIssues")] : [])) && issues) {
      await vscode.env.openExternal(vscode.Uri.parse(issues));
    }
  }
}

function escape(s: string): string {
  return s.replace(/[\\`*_{}[\]()#+\-.!|<>]/g, "\\$&");
}

interface BuildInfo { version: string; flavor: "dev" | "prod" }

function readBuildInfo(extensionPath: string): BuildInfo {
  try {
    const raw = JSON.parse(fs.readFileSync(path.join(extensionPath, "media", "build.json"), "utf8")) as Partial<BuildInfo>;
    return { version: raw.version ?? "dev", flavor: raw.flavor === "dev" ? "dev" : "prod" };
  } catch {
    return { version: "dev", flavor: "prod" };
  }
}

import * as vscode from "vscode";
import {
  Avatar, Cli, CliError, Route, ScanResult, Status, bundledBinary, ensureExecutable, formatDistance, gitGlobalEmail,
} from "./cli";
import { PanelMessage, TrailPanel } from "./panel";
import { RepoTracker } from "./repos";

export async function activate(ctx: vscode.ExtensionContext): Promise<void> {
  let binary: string;
  try {
    binary = vscode.workspace.getConfiguration("commitHike").get<string>("binaryPath") || bundledBinary(ctx.extensionPath);
    ensureExecutable(binary);
  } catch (e) {
    void vscode.window.showErrorMessage(`Commit Hike can't start: ${(e as Error).message}`);
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

  constructor(private readonly ctx: vscode.ExtensionContext, private readonly cli: Cli) {
    this.panel = new TrailPanel(ctx.extensionPath, m => void this.onPanelMessage(m));
    this.tracker = new RepoTracker(root => void this.scan(root));
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
      vscode.commands.registerCommand("commitHike.createRouteTemplate", () => this.guard(() => this.createRouteTemplate())),
      vscode.commands.registerCommand("commitHike.removeRoute", () => this.guard(() => this.removeRoute())),
      vscode.commands.registerCommand("commitHike.setHikerIcon", () => this.guard(() => this.setHikerIcon())),
      vscode.commands.registerCommand("commitHike.resetHikerIcon", () => this.guard(() => this.resetHikerIcon())),
      vscode.window.onDidChangeActiveTextEditor(() => this.followActiveRepo()),
      this.tracker.onDidChangeRepos(() => this.followActiveRepo(true)),
    );

    try {
      this.routes = Object.fromEntries((await this.cli.routes()).map(r => [r.id, r]));
      this.avatar = await this.cli.avatar();
      await this.cli.config();
      this.initialized = true;
    } catch (e) {
      if (!(e instanceof CliError && e.notInitialized)) this.report(e);
    }
    await this.tracker.start();
    await this.refresh();
    if (!this.initialized) void this.offerSetupOnce();
  }

  // ---------- scanning & rendering ----------

  private async scan(root: string): Promise<void> {
    if (!this.initialized) return;
    try {
      const res = await this.cli.scan(root);
      this.celebrate(res);
    } catch (e) {
      this.report(e);
    }
    await this.refresh();
  }

  private celebrate(res: ScanResult): void {
    const cfg = vscode.workspace.getConfiguration("commitHike");
    if (res.added_m > 0 && cfg.get<boolean>("showDistanceAfterCommit", true)) {
      clearTimeout(this.flashTimer);
      this.statusItem.text = `🥾 +${formatDistance(res.added_m)}`;
      this.flashTimer = setTimeout(() => { this.flashTimer = undefined; this.renderStatusBar(); }, 4000);
    }
    if (!cfg.get<boolean>("notifyOnWaypoints", true)) return;
    const events = res.events ?? [];
    // Achievements are always worth a notification of their own.
    for (const e of events.filter(e => e.type === "achievement" && e.achievement)) {
      void vscode.window.showInformationMessage(`🏅 ${e.achievement!.name}: ${e.achievement!.description ?? ""}`, "Show trail")
        .then(a => { if (a) this.showTrail(); });
    }
    // Importing history can pass many stops at once: announce only the latest per journey.
    for (const journey of ["global", "project"] as const) {
      const evs = events.filter(e => e.journey === journey);
      const j = journey === "global" ? res.global : res.project;
      if (j && evs.some(e => e.type === "finished")) {
        void vscode.window.showInformationMessage(`You finished ${j.route.name}! Pick your next trail.`, "Choose trail")
          .then(a => { if (a) void this.chooseRoute(journey); });
        continue;
      }
      const w = evs.filter(e => e.type === "waypoint").pop()?.waypoint;
      if (w) {
        void vscode.window.showInformationMessage(w.text ? `${w.name}: ${w.text}` : `You reached ${w.name}.`, "Show trail")
          .then(a => { if (a) this.showTrail(); });
      }
    }
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
      this.renderStatusBar();
      this.panel.update({
        type: "update", state: "ok", repo: this.currentRepo, locale: this.lastStatus.locale, status: this.lastStatus,
        avatar: this.avatar.data_url, avatar_custom: this.avatar.custom,
      });
    } catch (e) {
      if (e instanceof CliError && e.notInitialized) { this.initialized = false; return this.refresh(); }
      this.report(e);
      this.panel.update({ type: "update", state: "error", locale: vscode.env.language, error: (e as Error).message });
    }
  }

  private renderStatusBar(): void {
    if (this.flashTimer) return; // "+86 m" is still showing
    const s = this.lastStatus;
    const g = s?.global;
    if (!this.initialized || !s || !g) {
      this.statusItem.text = "🥾 Commit Hike";
      this.statusItem.tooltip = "Set up Commit Hike to turn your commits into a journey.";
      return;
    }
    this.statusItem.text = g.finished
      ? `🥾 ${formatDistance(g.distance_m)} ✓`
      : `🥾 ${formatDistance(g.distance_m)} / ${formatDistance(g.route.length_m)}`;
    const md = new vscode.MarkdownString();
    md.appendMarkdown(`**${escape(g.route.name)}**: ${formatDistance(g.distance_m)} of ${formatDistance(g.route.length_m)} (${g.percent}%)\n\n`);
    if (g.finished) md.appendMarkdown("Trail completed.\n\n");
    else if (g.next_waypoint) md.appendMarkdown(`Next stop: ${escape(g.next_waypoint.name)}, in ${formatDistance(g.to_next_m ?? 0)}\n\n`);
    md.appendMarkdown(`Today: ${formatDistance(s.today_m)}`);
    if (this.currentRepo && s.project) {
      md.appendMarkdown(`\n\nThis project: **${escape(s.project.route.name)}**, ${formatDistance(s.project.distance_m)} (${s.project.percent}%)`);
    } else if (this.currentRepo && !s.tracked) {
      md.appendMarkdown("\n\nThis project isn't counted.");
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
      "Commit Hike turns your commits into a hiking journey. It never writes to your projects and keeps everything on this computer.",
      "Set up", "Not now");
    if (a === "Set up") await this.guard(() => this.setup());
  }

  private async setup(): Promise<void> {
    const title = "Set up Commit Hike";
    const mode = await vscode.window.showQuickPick([
      { label: "Count all my projects", detail: "Commits in any repository you open move you forward.", value: "all" as const },
      { label: "Only projects I choose", detail: "You turn counting on for each project.", value: "selected" as const },
    ], { title: `${title} (1/3)`, placeHolder: "Which projects should count?", ignoreFocusOut: true });
    if (!mode) return;

    const history = await vscode.window.showQuickPick([
      { label: "Include commits I already made", detail: "Your history counts, so you may start partway along the trail.", value: true },
      { label: "Start from today", detail: "Only new commits count.", value: false },
    ], { title: `${title} (2/3)`, placeHolder: "Where does your journey start?", ignoreFocusOut: true });
    if (!history) return;

    const emails = await vscode.window.showInputBox({
      title: `${title} (3/3)`,
      prompt: "Commits with these author emails count as yours. Separate several with commas.",
      value: await gitGlobalEmail(),
      ignoreFocusOut: true,
      validateInput: v => (v.split(",").some(e => e.includes("@")) ? undefined : "Enter at least one email address."),
    });
    if (emails === undefined) return;

    await this.cli.init({ emails: emails.split(",").map(e => e.trim()).filter(Boolean), mode: mode.value, fromHistory: history.value });
    this.initialized = true;

    if (mode.value === "selected" && this.currentRepo) {
      const a = await vscode.window.showInformationMessage("Count commits in this project?", "Count it", "Not now");
      if (a === "Count it") await this.cli.setProjectEnabled(this.currentRepo, true);
    }
    await vscode.window.withProgress(
      { location: vscode.ProgressLocation.Window, title: "Commit Hike: counting your commits" },
      async () => { for (const r of this.tracker.repositories) await this.scan(r); });
    await this.refresh();
    this.showTrail();
  }

  private async chooseRoute(scope: "global" | "project"): Promise<void> {
    if (!this.initialized) return this.setup();
    const repo = scope === "project" ? this.currentRepo : undefined;
    if (scope === "project" && !repo) {
      void vscode.window.showInformationMessage("Open a file from a git repository to choose a trail for that project.");
      return;
    }
    type Item = vscode.QuickPickItem & { id: string };
    const items: Item[] = Object.values(this.routes).map(r => ({
      id: r.id, label: r.name, description: formatDistance(r.length_m), detail: r.description,
    }));
    if (scope === "project" && this.lastStatus?.project) {
      items.push({ id: "none", label: "No trail for this project", detail: "Commits here still count toward your main journey." });
    }
    items.push(
      { id: "", label: "", kind: vscode.QuickPickItemKind.Separator },
      { id: "@import", label: "$(cloud-download) Import a route…" },
      { id: "@template", label: "$(new-file) Create a route template…" },
    );
    const pick = await vscode.window.showQuickPick(items, {
      title: scope === "global" ? "Trail for all projects" : "Trail for this project", placeHolder: "Choose a trail",
    });
    if (!pick) return;
    if (pick.id === "@import") return this.importRoute();
    if (pick.id === "@template") return this.createRouteTemplate();
    let fromHistory = false;
    if (pick.id !== "none") {
      const h = await vscode.window.showQuickPick([
        { label: "Include commits I already made", value: true },
        { label: "Start from now", value: false },
      ], { title: pick.label, placeHolder: "Where does this journey start?" });
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
      void vscode.window.showInformationMessage("Open a file from a git repository first.");
      return;
    }
    const cfg = await this.cli.config();
    if (cfg.mode === "all") {
      void vscode.window.showInformationMessage(
        "All your projects are already counted. To choose projects one by one, run “Commit Hike: Set Up”.");
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
      { location: vscode.ProgressLocation.Notification, title: "Commit Hike: recounting this project from git history" },
      () => this.cli.verify(repo));
    await this.refresh();
    const changes = r.added + r.updated + r.removed;
    void vscode.window.showInformationMessage(changes === 0
      ? "Recount finished: everything already matched your git history."
      : `Recount finished: ${r.added} added, ${r.updated} corrected, ${r.removed} removed.`);
  }

  // ---------- custom routes ----------

  private async importRoute(): Promise<void> {
    if (!this.initialized) return this.setup();
    // Linux and Windows dialogs can pick files or folders, not both: ask first.
    const kind = await vscode.window.showQuickPick([
      { label: "From a .zip file", folders: false },
      { label: "From a folder", folders: true },
    ], { title: "Import a route", placeHolder: "Where is the route?" });
    if (!kind) return;
    const picked = await vscode.window.showOpenDialog({
      title: "Import a route", openLabel: "Import", canSelectMany: false,
      canSelectFiles: !kind.folders, canSelectFolders: kind.folders,
      filters: kind.folders ? undefined : { "Route pack": ["zip"] },
    });
    const src = picked?.[0]?.fsPath;
    if (!src) return;
    let route;
    try {
      route = await this.cli.importRoute(src, false);
    } catch (e) {
      if (!(e instanceof CliError) || e.code !== "route_exists") throw e;
      const a = await vscode.window.showWarningMessage(`${e.message}. Replace it?`, { modal: true }, "Replace");
      if (a !== "Replace") return;
      route = await this.cli.importRoute(src, true);
    }
    await this.reloadRoutes();
    const a = await vscode.window.showInformationMessage(
      `Imported ${route.name}: ${formatDistance(route.length_m)}, ${route.waypoints.length} stops.`, "Walk it now");
    if (a) {
      await this.cli.setJourney("global", route.id, false);
      await this.refresh();
      this.showTrail();
    }
  }

  private async createRouteTemplate(): Promise<void> {
    const folder = await vscode.window.showOpenDialog({
      title: "Where to create the route", openLabel: "Create here", canSelectFolders: true, canSelectFiles: false,
    });
    if (!folder?.[0]) return;
    const id = await vscode.window.showInputBox({
      title: "New route", prompt: "Route id: lowercase letters, digits and dashes.", value: "my-trail",
      validateInput: v => (/^[a-z0-9]+(-[a-z0-9]+)*$/.test(v.trim()) ? undefined : "Use lowercase letters, digits and single dashes."),
    });
    if (!id) return;
    const dir = await this.cli.routeTemplate(id.trim(), folder[0].fsPath);
    await vscode.window.showTextDocument(vscode.Uri.file(`${dir}/route.json`));
    const a = await vscode.window.showInformationMessage(
      "Route template created. Edit route.json and locales/*.json, then import the folder.", "Import now");
    if (a) await this.importRoute();
  }

  private async removeRoute(): Promise<void> {
    const custom = Object.values(this.routes).filter(r => !r.builtin);
    if (custom.length === 0) {
      void vscode.window.showInformationMessage("You haven't imported any routes.");
      return;
    }
    const pick = await vscode.window.showQuickPick(
      custom.map(r => ({ label: r.name, description: formatDistance(r.length_m), id: r.id })),
      { title: "Remove a route", placeHolder: "Progress you made stays; you just can't choose the route anymore." });
    if (!pick) return;
    await this.cli.removeRoute(pick.id);
    await this.reloadRoutes();
    void vscode.window.showInformationMessage(`${pick.label} was removed.`);
  }

  // ---------- hiker icon ----------

  private async setHikerIcon(): Promise<void> {
    const picked = await vscode.window.showOpenDialog({
      title: "Choose a hiker icon (PNG with a transparent background, up to 512×512, facing right)",
      openLabel: "Use as hiker", canSelectMany: false, filters: { "PNG image": ["png"] },
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
  }

  private async onPanelMessage(m: PanelMessage): Promise<void> {
    switch (m.command) {
      case "setup": return this.guard(() => this.setup());
      case "refresh": return this.refresh();
      case "enableProject": return this.guard(() => this.setProjectEnabled(true));
      case "setAvatar": return this.guard(() => this.setHikerIcon());
      case "resetAvatar": return this.guard(() => this.resetHikerIcon());
      case "chooseRoute": return this.guard(() => this.chooseRoute(m.scope));
    }
  }

  // ---------- errors ----------

  private async guard(fn: () => Promise<void>): Promise<void> {
    try {
      await fn();
    } catch (e) {
      this.report(e);
      void vscode.window.showErrorMessage(`Commit Hike: ${(e as Error).message}`);
    }
  }

  private report(e: unknown): void {
    this.log.appendLine(`[${new Date().toISOString()}] ${(e as Error)?.message ?? String(e)}`);
  }
}

function escape(s: string): string {
  return s.replace(/[\\`*_{}[\]()#+\-.!|<>]/g, "\\$&");
}

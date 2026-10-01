// Hosts ui/panel.html (shared with JetBrains) in a sidebar webview.
import * as crypto from "crypto";
import * as fs from "fs";
import * as path from "path";
import * as vscode from "vscode";
import { LocaleInfo, RouteAssets, Status, Team } from "./cli";

export interface PanelData {
  type: "update";
  state: "ok" | "not_initialized" | "error";
  error?: string;
  repo?: string;
  locale?: string;
  status?: Status;
  avatar?: string; // custom hiker PNG as a data URL; absent = the panel's default
  avatar_custom?: boolean;
  assets?: Record<string, RouteAssets>; // pictures for route objects and custom maps, per route id
  team?: Team;
  team_error?: string;
  locale_setting?: LocaleInfo;
  dev?: boolean; // development build: the trail view may look ahead
  build?: { version: string; flavor: "dev" | "prod" }; // shown at the bottom of the stats tab
  open_view?: string; open_token?: number; // a page the panel opens once per token, e.g. "settings"
}

export type PanelMessage =
  | { command: "ready" | "setup" | "refresh" | "enableProject" | "setAvatar" | "resetAvatar"
    | "importRoute" | "createRouteTemplate" | "verify" | "requestTeam" }
  | { command: "chooseRoute"; scope: "global" | "project" }
  | { command: "setLocale"; locale: string }
  | { command: "setTeam"; on: boolean }
  | { command: "setDifficulty"; level: "easy" | "medium" | "hard" }
  | { command: "savePostcard"; name: string; data: string }
  | { command: "setRestDays"; days: number[] }
  | { command: "setSettings"; reduce_motion?: string; high_contrast?: string; notifications?: string }
  | { command: "exportProgress" | "importProgress" };

export class TrailPanel implements vscode.WebviewViewProvider {
  static readonly viewId = "commitHike.trail";
  private view?: vscode.WebviewView;
  private last?: PanelData;

  constructor(private readonly extensionPath: string, private readonly onMessage: (m: PanelMessage) => void) {}

  resolveWebviewView(view: vscode.WebviewView): void {
    this.view = view;
    view.webview.options = { enableScripts: true, localResourceRoots: [] };
    const nonce = crypto.randomBytes(16).toString("base64");
    const csp = [
      "default-src 'none'",
      `style-src ${view.webview.cspSource} 'unsafe-inline'`,
      "img-src data:", // the hiker icon is passed as a data URL
      `script-src 'nonce-${nonce}'`,
    ].join("; ");
    const html = fs.readFileSync(path.join(this.extensionPath, "media", "panel.html"), "utf8");
    view.webview.html = html.replace("{{CSP}}", csp).replace(/\{\{NONCE\}\}/g, nonce);
    view.webview.onDidReceiveMessage((m: PanelMessage) => {
      if (m.command === "ready") { if (this.last) void view.webview.postMessage(this.last); }
      else this.onMessage(m);
    });
    view.onDidChangeVisibility(() => { if (view.visible && this.last) void view.webview.postMessage(this.last); });
    view.onDidDispose(() => { this.view = undefined; });
  }

  update(data: PanelData): void {
    this.last = data;
    if (this.view?.visible) void this.view.webview.postMessage(data);
  }
}

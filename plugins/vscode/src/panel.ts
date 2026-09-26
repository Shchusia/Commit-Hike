// Hosts ui/panel.html (shared with JetBrains) in a sidebar webview.
import * as crypto from "crypto";
import * as fs from "fs";
import * as path from "path";
import * as vscode from "vscode";
import { Status } from "./cli";

export interface PanelData {
  type: "update";
  state: "ok" | "not_initialized" | "error";
  error?: string;
  repo?: string;
  locale?: string;
  status?: Status;
  avatar?: string; // custom hiker PNG as a data URL; absent = the panel's default
  avatar_custom?: boolean;
}

export type PanelMessage =
  | { command: "ready" | "setup" | "refresh" | "enableProject" | "setAvatar" | "resetAvatar" }
  | { command: "chooseRoute"; scope: "global" | "project" };

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

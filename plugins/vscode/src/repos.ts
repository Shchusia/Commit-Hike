// Finds git repositories in the workspace and reports when HEAD moves
// (commit, amend, pull, rebase). Read-only: nothing is written to any repo.
//
// Primary source: VS Code's built-in git extension, which already knows every
// repository (nested ones, worktrees, submodules) and tracks HEAD.
// Fallback (git extension disabled): watch .git/logs and rescan on focus.
import { execFile } from "child_process";
import * as fs from "fs";
import * as path from "path";
import * as vscode from "vscode";

// Minimal typing of the public API of the built-in `vscode.git` extension.
interface GitRepository {
  rootUri: vscode.Uri;
  state: { HEAD?: { commit?: string }; onDidChange: vscode.Event<void> };
}
interface GitAPI {
  state: "uninitialized" | "initialized";
  onDidChangeState: vscode.Event<"uninitialized" | "initialized">;
  repositories: GitRepository[];
  onDidOpenRepository: vscode.Event<GitRepository>;
  onDidCloseRepository: vscode.Event<GitRepository>;
}

export class RepoTracker implements vscode.Disposable {
  private readonly roots = new Set<string>();
  private readonly heads = new Map<string, string | undefined>();
  private readonly timers = new Map<string, NodeJS.Timeout>();
  private readonly disposables: vscode.Disposable[] = [];
  private readonly watchers: fs.FSWatcher[] = [];
  private readonly changed = new vscode.EventEmitter<void>();
  /** Fires when the set of repositories changes. */
  readonly onDidChangeRepos = this.changed.event;

  constructor(private readonly onHeadMoved: (root: string) => void) {}

  async start(): Promise<void> {
    const api = await this.gitApi();
    if (api) this.useGitApi(api);
    else await this.useFallback();
  }

  get repositories(): string[] { return [...this.roots]; }

  /** The repository a file belongs to (deepest root wins for nested repos). */
  repoFor(uri: vscode.Uri | undefined): string | undefined {
    if (!uri || uri.scheme !== "file") return undefined;
    let best: string | undefined;
    for (const r of this.roots) {
      const rel = path.relative(r, uri.fsPath);
      if (!rel.startsWith("..") && !path.isAbsolute(rel) && (!best || r.length > best.length)) best = r;
    }
    return best;
  }

  private async gitApi(): Promise<GitAPI | undefined> {
    try {
      const ext = vscode.extensions.getExtension<{ getAPI(v: 1): GitAPI }>("vscode.git");
      if (!ext) return undefined;
      const api = (ext.isActive ? ext.exports : await ext.activate()).getAPI(1);
      if (api.state === "initialized") return api;
      return await new Promise<GitAPI | undefined>(resolve => {
        const d = api.onDidChangeState(s => { if (s === "initialized") { d.dispose(); resolve(api); } });
        setTimeout(() => { d.dispose(); resolve(undefined); }, 15_000);
      });
    } catch {
      return undefined; // git extension disabled or failed
    }
  }

  private useGitApi(api: GitAPI): void {
    const add = (repo: GitRepository) => {
      const root = repo.rootUri.fsPath;
      if (this.roots.has(root)) return;
      this.roots.add(root);
      this.heads.set(root, repo.state.HEAD?.commit);
      this.disposables.push(repo.state.onDidChange(() => {
        const head = repo.state.HEAD?.commit;
        if (head !== this.heads.get(root)) {
          this.heads.set(root, head);
          this.schedule(root);
        }
      }));
      this.schedule(root); // initial scan picks up commits made while VS Code was closed
      this.changed.fire();
    };
    api.repositories.forEach(add);
    this.disposables.push(
      api.onDidOpenRepository(add),
      api.onDidCloseRepository(repo => { this.roots.delete(repo.rootUri.fsPath); this.changed.fire(); }),
    );
  }

  private async useFallback(): Promise<void> {
    for (const folder of vscode.workspace.workspaceFolders ?? []) {
      const info = await gitDirs(folder.uri.fsPath);
      if (!info || this.roots.has(info.root)) continue;
      this.roots.add(info.root);
      const logs = path.join(info.gitDir, "logs");
      try {
        this.watchers.push(fs.watch(fs.existsSync(logs) ? logs : info.gitDir, () => this.schedule(info.root)));
      } catch { /* unwatchable filesystem: the focus rescan below still works */ }
      this.schedule(info.root);
    }
    this.disposables.push(vscode.window.onDidChangeWindowState(s => {
      if (s.focused) this.roots.forEach(r => this.schedule(r));
    }));
    this.changed.fire();
  }

  // git writes several files per commit; coalesce them into one scan.
  private schedule(root: string): void {
    clearTimeout(this.timers.get(root));
    this.timers.set(root, setTimeout(() => { this.timers.delete(root); this.onHeadMoved(root); }, 800));
  }

  dispose(): void {
    this.timers.forEach(t => clearTimeout(t));
    this.watchers.forEach(w => w.close());
    this.disposables.forEach(d => { d.dispose(); }); // vscode types dispose() as any
    this.changed.dispose();
  }
}

function gitDirs(dir: string): Promise<{ root: string; gitDir: string } | undefined> {
  return new Promise(resolve => {
    execFile("git", ["-C", dir, "rev-parse", "--show-toplevel", "--absolute-git-dir"], { windowsHide: true },
      (err, stdout) => {
        const [root, gitDir] = stdout.trim().split(/\r?\n/);
        resolve(err || !root || !gitDir ? undefined : { root: path.normalize(root), gitDir: path.normalize(gitDir) });
      });
  });
}

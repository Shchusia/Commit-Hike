// One-time setup of who publishes Commit Hike. Run through Task:
//
//   task release:configure OWNER=your-github-name NAME="Your Name" EMAIL=you@example.com
//
// Optional: PLUGIN_ID (default io.github.<owner>.commithike), PUBLISHER (VS Code
// publisher id, default <owner>), REPO (default commit-hike).
// Safe to run again: it rewrites the same fields with the new values.
import fs from "node:fs";
import path from "node:path";

const root = path.resolve(path.dirname(new URL(import.meta.url).pathname), "..");
const env = process.env;
const fail = msg => { console.error("release:configure: " + msg); process.exit(1); };

const owner = (env.OWNER || "").trim();
const name = (env.NAME || "").trim();
const email = (env.EMAIL || "").trim();
const repo = (env.REPO || "commit-hike").trim();
if (!/^[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})$/.test(owner)) fail("OWNER must be your GitHub user or organisation name");
if (!name) fail("NAME is required (shown as the vendor on JetBrains Marketplace)");
if (!/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(email)) fail("EMAIL must be an email address");
const pluginId = (env.PLUGIN_ID || `io.github.${owner.toLowerCase().replace(/-/g, "")}.commithike`).trim();
if (!/^[a-z][a-z0-9_]*(\.[a-z0-9_]+)+$/.test(pluginId)) fail(`PLUGIN_ID ${pluginId} must look like io.github.name.commithike`);
const publisher = (env.PUBLISHER || owner).trim().toLowerCase();
if (!/^[a-z0-9][a-z0-9-]*$/.test(publisher)) fail("PUBLISHER may contain lowercase letters, digits and dashes");

const repoUrl = `https://github.com/${owner}/${repo}`;
const esc = s => s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
const edit = (rel, fn) => {
  const p = path.join(root, rel);
  const before = fs.readFileSync(p, "utf8");
  const after = fn(before);
  if (after !== before) { fs.writeFileSync(p, after); console.log("updated " + rel); }
};

// JetBrains: plugin id and vendor. The id can never change after the first upload.
edit("plugins/jetbrains/src/main/resources/META-INF/plugin.xml", s => s
  .replace(/<id>[^<]*<\/id>/, `<id>${pluginId}</id>`)
  .replace(/<vendor[^>]*>[^<]*<\/vendor>/, `<vendor email="${esc(email)}" url="${esc(repoUrl)}">${esc(name)}</vendor>`));
edit("plugins/jetbrains/src/main/kotlin/dev/commithike/CommitHikeApp.kt", s =>
  s.replace(/private const val PLUGIN_ID = "[^"]*"/, `private const val PLUGIN_ID = "${pluginId}"`));

// VS Code: publisher and links.
edit("plugins/vscode/package.json", s => {
  const d = JSON.parse(s);
  d.publisher = publisher;
  d.repository = { type: "git", url: `${repoUrl}.git`, directory: "plugins/vscode" };
  d.homepage = `${repoUrl}#readme`;
  d.bugs = { url: `${repoUrl}/issues` };
  d.scripts.package = d.scripts.package.replace(" --allow-missing-repository", "");
  return JSON.stringify(d, null, 2) + "\n";
});

// Go: the module path follows the repository (imports, go.mod, lint config).
const newPrefix = `github.com/${owner}/${repo}`;
const oldPrefix = fs.readFileSync(path.join(root, "core/go.mod"), "utf8").match(/^module (\S+)/m)[1].replace(/\/core$/, "");
if (oldPrefix !== newPrefix) {
  const walk = dir => fs.readdirSync(dir, { withFileTypes: true }).flatMap(e =>
    e.isDirectory() ? walk(path.join(dir, e.name)) : [path.join(dir, e.name)]);
  for (const f of walk(path.join(root, "core")).filter(f => f.endsWith(".go") || f.endsWith("go.mod") || f.endsWith(".golangci.yml"))) {
    edit(path.relative(root, f), s => s.split(oldPrefix).join(newPrefix));
  }
}

// Remember the answers for the release tasks.
fs.writeFileSync(path.join(root, "release.env"),
  `# Written by task release:configure. Public information only: never put tokens here.\n` +
  `OWNER=${owner}\nREPO=${repo}\nPLUGIN_ID=${pluginId}\nPUBLISHER=${publisher}\n`);
console.log(`\nConfigured: JetBrains plugin ${pluginId} by ${name}, VS Code publisher ${publisher}, ${repoUrl}`);

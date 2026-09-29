// Checks that a release is ready to be built and published:
//   task release:check
import fs from "node:fs";
import path from "node:path";
import { execSync } from "node:child_process";

const root = path.resolve(path.dirname(new URL(import.meta.url).pathname), "..");
const read = rel => fs.readFileSync(path.join(root, rel), "utf8");
const problems = [];

const v = read("VERSION").trim();
if (!/^\d+\.\d+\.\d+(-[0-9A-Za-z.]+)?$/.test(v)) problems.push(`VERSION "${v}" is not a semantic version`);
const pkg = JSON.parse(read("plugins/vscode/package.json"));
if (pkg.version !== v) problems.push(`plugins/vscode/package.json has ${pkg.version}, VERSION has ${v}: run task version:set V=${v}`);

const cl = read("CHANGELOG.md");
const sec = cl.match(new RegExp(`## \\[${v.replace(/\./g, "\\.")}\\][^\\n]*\\n([\\s\\S]*?)(?=\\n## |\\s*$)`));
if (!sec || !sec[1].trim()) problems.push(`CHANGELOG.md has no notes for ${v}: run task version:set V=${v} after writing them under [Unreleased]`);

if (!fs.existsSync(path.join(root, "release.env"))) problems.push("publisher not configured: run task release:configure OWNER=... NAME=... EMAIL=...");
if (/<id>dev\.commithike<\/id>/.test(read("plugins/jetbrains/src/main/resources/META-INF/plugin.xml"))) {
  problems.push("the JetBrains plugin still has the placeholder id dev.commithike");
}
// README images on the marketplace pages must be absolute URLs to this
// repository's default branch (BRANCH in release.env, master if unset), and
// the files must exist. Not the current branch: the marketplace page keeps
// these links long after a release, and a feature branch is deleted after the
// merge (a wrong branch = broken images).
if (fs.existsSync(path.join(root, "release.env"))) {
  const env = Object.fromEntries(read("release.env").split("\n").filter(l => l.includes("=") && !l.startsWith("#")).map(l => l.split("=").map(x => x.trim())));
  const branch = env.BRANCH || "master";
  const git = args => execSync("git " + args, { cwd: root, stdio: ["ignore", "pipe", "ignore"] }).toString().trim();
  let remote = ""; // origin/<branch> if this checkout knows it: then the image must already be there
  try { git(`rev-parse --verify --quiet refs/remotes/origin/${branch}`); remote = `origin/${branch}`; } catch { /* no remote ref: check the working tree only */ }
  const prefix = `https://raw.githubusercontent.com/${env.OWNER}/${env.REPO}/${branch}/`;
  for (const src of [...read("plugins/vscode/README.md").matchAll(/src="([^"]+)"/g)].map(m => m[1])) {
    if (!src.startsWith("https://")) problems.push(`plugins/vscode/README.md: image ${src} must be an absolute https URL (marketplaces can't show relative ones)`);
    else if (src.startsWith("https://raw.githubusercontent.com/") && !src.startsWith(prefix)) {
      problems.push(`plugins/vscode/README.md: image ${src} doesn't point to ${prefix} (the repository or the default branch "${branch}" differ; set BRANCH in release.env if your default branch has another name)`);
    } else if (src.startsWith(prefix)) {
      const file = src.slice(prefix.length);
      if (!fs.existsSync(path.join(root, file))) problems.push(`plugins/vscode/README.md: image ${file} doesn't exist in the repository`);
      else if (remote) {
        try { git(`cat-file -e ${remote}:${file}`); } catch {
          problems.push(`plugins/vscode/README.md: image ${file} isn't on ${remote} yet: merge (or push) it first, or the marketplace page shows a broken image`);
        }
      }
    }
  }
}
if ((process.env.FLAVOR || "prod") !== "prod") problems.push("releases are prod builds: drop FLAVOR=dev");

for (const r of fs.readdirSync(path.join(root, "core/content/routes"))) {
  const en = JSON.parse(read(`core/content/routes/${r}/locales/en.json`));
  if (/tolkien|middle-earth|bag end|frodo|bilbo|mordor/i.test(JSON.stringify(en))) {
    problems.push(`built-in route ${r} uses Tolkien's world: marketplaces will reject it (see docs/publishing.md)`);
  }
}

if (problems.length) {
  console.error("Not ready to release " + v + ":\n" + problems.map(p => "  - " + p).join("\n"));
  process.exit(1);
}
console.log(`Ready to release ${v}.`);

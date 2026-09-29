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
// repository and branch, and the files must exist (a wrong branch = broken images).
if (fs.existsSync(path.join(root, "release.env"))) {
  const env = Object.fromEntries(read("release.env").split("\n").filter(l => l.includes("=") && !l.startsWith("#")).map(l => l.split("=")));
  let branch = "";
  try { branch = execSync("git rev-parse --abbrev-ref HEAD", { cwd: root, stdio: ["ignore", "pipe", "ignore"] }).toString().trim(); } catch { /* not a git checkout */ }
  const prefix = `https://raw.githubusercontent.com/${env.OWNER}/${env.REPO}/${branch}/`;
  for (const src of [...read("plugins/vscode/README.md").matchAll(/src="([^"]+)"/g)].map(m => m[1])) {
    if (!src.startsWith("https://")) problems.push(`plugins/vscode/README.md: image ${src} must be an absolute https URL (marketplaces can't show relative ones)`);
    else if (branch && branch !== "HEAD" && src.startsWith("https://raw.githubusercontent.com/") && !src.startsWith(prefix)) {
      problems.push(`plugins/vscode/README.md: image ${src} doesn't point to ${prefix} (repository or branch "${branch}" differ)`);
    } else if (src.startsWith(prefix) && !fs.existsSync(path.join(root, src.slice(prefix.length)))) {
      problems.push(`plugins/vscode/README.md: image ${src.slice(prefix.length)} doesn't exist in the repository`);
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

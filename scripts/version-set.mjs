// Sets the project version everywhere and opens its CHANGELOG section:
//   task version:set V=0.2.0
// - VERSION (read by the Go core build, the JetBrains build and the VS Code build)
// - plugins/vscode/package.json and package-lock.json
// - "Version" lines in README.md and README.uk.md
// - CHANGELOG.md: what's under [Unreleased] becomes the new version's section
import fs from "node:fs";
import path from "node:path";

const root = path.resolve(path.dirname(new URL(import.meta.url).pathname), "..");
const v = (process.env.V || "").trim();
if (!/^\d+\.\d+\.\d+(-[0-9A-Za-z.]+)?$/.test(v)) {
  console.error("version:set: V must be a semantic version like 1.2.3 or 1.3.0-beta");
  process.exit(1);
}
const file = rel => path.join(root, rel);
const today = new Date().toISOString().slice(0, 10);

fs.writeFileSync(file("VERSION"), v + "\n");

for (const rel of ["plugins/vscode/package.json", "plugins/vscode/package-lock.json"]) {
  if (!fs.existsSync(file(rel))) continue;
  const d = JSON.parse(fs.readFileSync(file(rel), "utf8"));
  d.version = v;
  if (d.packages && d.packages[""]) d.packages[""].version = v;
  fs.writeFileSync(file(rel), JSON.stringify(d, null, 2) + "\n");
}

for (const rel of ["README.md", "README.uk.md", "README.pl.md", "README.de.md", "README.es.md"]) {
  if (!fs.existsSync(file(rel))) continue;
  const s = fs.readFileSync(file(rel), "utf8");
  fs.writeFileSync(file(rel), s.replace(/(\*\*(?:Version|Версія):\*\* )[0-9A-Za-z.-]+/, `$1${v}`));
}

const cl = fs.readFileSync(file("CHANGELOG.md"), "utf8");
if (cl.includes(`## [${v}]`)) {
  console.log(`CHANGELOG.md already has a ${v} section`);
} else {
  const m = cl.match(/## \[Unreleased\][^\n]*\n([\s\S]*?)(?=\n## |\s*$)/);
  if (!m) { console.error("CHANGELOG.md has no ## [Unreleased] section"); process.exit(1); }
  const notes = m[1].trim();
  const updated = cl.replace(m[0], `## [Unreleased]\n\n## [${v}] - ${today}\n\n${notes || "- No notable changes."}\n`);
  fs.writeFileSync(file("CHANGELOG.md"), updated);
}
console.log(`Version is now ${v}. Review CHANGELOG.md, then commit and tag: git tag v${v}`);

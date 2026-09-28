// Prints the CHANGELOG.md section of the current VERSION (for GitHub releases).
import fs from "node:fs";
import path from "node:path";

const root = path.resolve(path.dirname(new URL(import.meta.url).pathname), "..");
const v = fs.readFileSync(path.join(root, "VERSION"), "utf8").trim();
const cl = fs.readFileSync(path.join(root, "CHANGELOG.md"), "utf8");
const m = cl.match(new RegExp(`## \\[${v.replace(/\./g, "\\.")}\\][^\\n]*\\n([\\s\\S]*?)(?=\\n## |\\s*$)`));
if (!m || !m[1].trim()) {
  console.error(`CHANGELOG.md has no notes for ${v}`);
  process.exit(1);
}
process.stdout.write(m[1].trim() + "\n");

// Copies the core binaries built by `task core:dist` into ./bin.
const fs = require("fs");
const path = require("path");
const dist = path.join(__dirname, "..", "..", "..", "core", "dist");
const out = path.join(__dirname, "..", "bin");
const files = fs.existsSync(dist) ? fs.readdirSync(dist).filter(f => f.startsWith("commit-hike-")) : [];
if (files.length === 0) {
  console.error(`No core binaries in ${dist}. Run \`task core:dist\` in the repository root first.`);
  process.exit(1);
}
fs.mkdirSync(out, { recursive: true });
for (const f of files) {
  fs.copyFileSync(path.join(dist, f), path.join(out, f));
  fs.chmodSync(path.join(out, f), 0o755);
}
console.log(`copied ${files.length} core binaries to bin/`);

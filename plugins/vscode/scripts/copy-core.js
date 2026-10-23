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
  // Copy beside, then rename over: writing into a binary that is running (a test,
  // an Extension Development Host) fails with ETXTBSY on Linux, a rename doesn't.
  const tmp = path.join(out, `.${f}.${process.pid}.tmp`);
  fs.copyFileSync(path.join(dist, f), tmp);
  fs.chmodSync(tmp, 0o755);
  fs.renameSync(tmp, path.join(out, f));
}
console.log(`copied ${files.length} core binaries to bin/`);

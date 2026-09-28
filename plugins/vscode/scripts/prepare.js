// Build-time preparation, run by `npm run build`:
//  - copies the shared trail panel (ui/panel/panel.html) into media/
//  - copies the root CHANGELOG.md (shown on the Marketplace "Changelog" tab)
//  - writes media/build.json with the version and the build flavor:
//      COMMIT_HIKE_FLAVOR=dev  -> the trail view may look ahead (for development)
//      anything else / unset  -> prod: only back (what the marketplaces get)
const fs = require("fs");
const path = require("path");
const root = path.join(__dirname, "..", "..", "..");
const media = path.join(__dirname, "..", "media");

fs.copyFileSync(path.join(root, "ui", "panel", "panel.html"), path.join(media, "panel.html"));
fs.copyFileSync(path.join(root, "CHANGELOG.md"), path.join(__dirname, "..", "CHANGELOG.md"));

const version = fs.readFileSync(path.join(root, "VERSION"), "utf8").trim();
const pkg = JSON.parse(fs.readFileSync(path.join(__dirname, "..", "package.json"), "utf8"));
if (pkg.version !== version) {
  console.error(`package.json version ${pkg.version} differs from VERSION ${version}. Run: task version:set V=${version}`);
  process.exit(1);
}
const flavor = process.env.COMMIT_HIKE_FLAVOR === "dev" ? "dev" : "prod";
fs.writeFileSync(path.join(media, "build.json"), JSON.stringify({ version, flavor }) + "\n");
console.log(`prepared media/ (Commit Hike ${version}, ${flavor} build)`);

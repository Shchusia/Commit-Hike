// The trail panel is shared with the JetBrains plugin; copy it in at build time.
const fs = require("fs");
const path = require("path");
const src = path.join(__dirname, "..", "..", "..", "ui", "panel", "panel.html");
const dst = path.join(__dirname, "..", "media", "panel.html");
fs.copyFileSync(src, dst);
console.log("copied", path.relative(process.cwd(), dst));

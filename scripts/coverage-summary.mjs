// Coverage summaries for `task check` and `task coverage` (not named coverage.* : .gitignore hides those).
//   node scripts/coverage-summary.mjs lcov  FILE "VS Code (TS)"   -> .sandbox/cover/summary-*.json
//   node scripts/coverage-summary.mjs kover FILE "JetBrains (Kotlin)"
//   node scripts/coverage-summary.mjs print                        -> the table of every part measured
import fs from "node:fs";
import path from "node:path";

const root = path.resolve(path.dirname(new URL(import.meta.url).pathname), "..");
const dir = process.env.COVER_DIR || path.join(root, ".sandbox", "cover");
const [cmd, file, part] = process.argv.slice(2);
const pct = (hit, all) => (all ? Math.round((1000 * hit) / all) / 10 : 100);
const save = (slug, summary) => {
  fs.mkdirSync(dir, { recursive: true });
  fs.writeFileSync(path.join(dir, `summary-${slug}.json`), JSON.stringify(summary));
};

if (cmd === "lcov") {
  // LH/LF lines, FNH/FNF functions, BRH/BRF branches, summed over the source files
  const sum = { LF: 0, LH: 0, FNF: 0, FNH: 0, BRF: 0, BRH: 0 };
  for (const line of fs.readFileSync(file, "utf8").split("\n")) {
    const [k, v] = line.split(":");
    if (k in sum) sum[k] += Number(v) || 0;
  }
  save("vscode", { part, percent: pct(sum.LH, sum.LF), detail: `lines · ${pct(sum.FNH, sum.FNF)}% of functions · ${pct(sum.BRH, sum.BRF)}% of branches` });
} else if (cmd === "kover") {
  // the report's own totals: <counter type="LINE" missed=".." covered=".."/> directly under <report>
  const xml = fs.readFileSync(file, "utf8");
  const tail = xml.slice(xml.lastIndexOf("</package>") + 1);
  const counter = type => {
    const m = tail.match(new RegExp(`<counter type="${type}" missed="(\\d+)" covered="(\\d+)"`));
    return m ? { missed: +m[1], covered: +m[2] } : { missed: 0, covered: 0 };
  };
  const line = counter("LINE"), method = counter("METHOD");
  save("jetbrains", { part, percent: pct(line.covered, line.missed + line.covered), detail: `lines · ${pct(method.covered, method.missed + method.covered)}% of methods · UI glue (ui, actions) not counted` });
} else if (cmd === "print") {
  const order = ["core", "panel", "vscode", "jetbrains"];
  const rows = order.map(slug => {
    const f = path.join(dir, `summary-${slug}.json`);
    return fs.existsSync(f) ? JSON.parse(fs.readFileSync(f, "utf8")) : null;
  }).filter(Boolean);
  if (!rows.length) {
    console.log("No coverage measured yet: run task coverage.");
  } else {
    const w = Math.max(...rows.map(r => r.part.length));
    console.log("\nCoverage");
    for (const r of rows) console.log(`  ${r.part.padEnd(w)}  ${String(r.percent.toFixed(1)).padStart(5)}%  ${r.detail}`);
    console.log(`Details: ${path.relative(process.cwd(), dir) || dir}/ (core.html, panel-uncovered.txt) and plugins/jetbrains/build/reports/kover/html/`);
  }
} else {
  console.error("usage: coverage-summary.mjs lcov|kover FILE PART | print");
  process.exit(2);
}

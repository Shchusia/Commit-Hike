// After all panel tests: adds up what each test ran (a part of the script
// counts as covered when any test ran it), prints the panel's coverage and
// writes the uncovered functions next to the other coverage reports.
import fs from "node:fs";
import path from "node:path";
import { COVERAGE } from "./panel.mjs";

export default function report() {
  if (!fs.existsSync(COVERAGE)) return;
  const files = fs.readdirSync(COVERAGE).filter(f => f.endsWith(".json"));
  if (!files.length) return;
  let covered = null, known = null;
  const fns = new Map(); // "start-end" -> { name, ran }
  for (const f of files) {
    for (const entry of JSON.parse(fs.readFileSync(path.join(COVERAGE, f), "utf8"))) {
      covered ||= new Uint8Array(entry.length);
      known ||= new Uint8Array(entry.length);
      if (entry.length !== covered.length) continue; // a different panel build; shouldn't happen
      // V8 block coverage: nested ranges override their parents, so paint the
      // largest first. 1 = ran, 2 = didn't.
      const mark = new Uint8Array(entry.length);
      const ranges = entry.functions.flatMap(fn => fn.ranges).sort((a, b) => a.startOffset - b.startOffset || b.endOffset - a.endOffset);
      for (const r of ranges) mark.fill(r.count > 0 ? 1 : 2, r.startOffset, Math.min(r.endOffset, entry.length));
      for (let i = 0; i < mark.length; i++) {
        if (mark[i]) known[i] = 1;
        if (mark[i] === 1) covered[i] = 1;
      }
      for (const fn of entry.functions) {
        const r = fn.ranges[0];
        if (!fn.functionName && r.startOffset === 0) continue; // the script itself
        const key = `${r.startOffset}-${r.endOffset}`;
        const prev = fns.get(key) || { name: fn.functionName || "(anonymous)", ran: false };
        prev.ran ||= r.count > 0;
        fns.set(key, prev);
      }
    }
  }
  if (!covered) return;
  let total = 0, hit = 0;
  for (let i = 0; i < known.length; i++) if (known[i]) { total++; if (covered[i]) hit++; }
  const all = [...fns.values()], ran = all.filter(f => f.ran).length;
  const codePct = (100 * hit / total).toFixed(1), fnPct = (100 * ran / all.length).toFixed(1);
  const named = all.filter(f => !f.ran && f.name !== "(anonymous)").map(f => f.name);

  const out = path.resolve(process.env.COVER_DIR || path.join(COVERAGE, "..", "..", "..", ".sandbox", "cover"));
  fs.mkdirSync(out, { recursive: true });
  fs.writeFileSync(path.join(out, "panel-uncovered.txt"), [...new Set(named)].sort().join("\n") + "\n");
  fs.writeFileSync(path.join(out, "summary-panel.json"),
    JSON.stringify({ part: "Panel (JS)", percent: +codePct, detail: `code · ${fnPct}% of functions` }));
  console.log(`\nPanel coverage: ${codePct}% of the code, ${fnPct}% of functions (${ran}/${all.length}).`);
  console.log(`Functions no test runs: ${path.join(out, "panel-uncovered.txt")}`);
  fs.rmSync(COVERAGE, { recursive: true, force: true });
}

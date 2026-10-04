// The <template id="script-classes"> lists every class the stylesheet styles
// that the static markup doesn't use: editors read it, the browser ignores it.
// It must neither miss a class (the editor would warn again) nor keep one
// nothing uses any more (that would hide dead CSS).
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { expect, test } from "./panel.mjs";

const here = path.dirname(fileURLToPath(import.meta.url));
const html = fs.readFileSync(path.join(here, "..", "panel", "panel.html"), "utf8");
const css = html.slice(html.indexOf("<style"), html.indexOf("</style>")).replace(/\/\*[\s\S]*?\*\//g, "");
const script = html.slice(html.indexOf("<script nonce"), html.lastIndexOf("</script>"));
const markup = html.slice(html.indexOf("<body"), html.indexOf("<script"));
const tpl = markup.slice(markup.indexOf('<template id="script-classes">'), markup.indexOf("</template>"));
const classesIn = s => [...s.matchAll(/class="([^"]*)"/g)].flatMap(m => m[1].split(/\s+/)).filter(Boolean);
const styled = new Set([...css.matchAll(/([^{}]+)\{/g)].flatMap(m => [...m[1].matchAll(/\.([A-Za-z_][\w-]*)/g)].map(c => c[1])));
const listed = new Set(classesIn(tpl));
const staticMarkup = new Set(classesIn(markup.replace(tpl, "")));
// Built from parts in the script, or set by a host rather than the script.
const BUILT = new Set(["l0", "l1", "l2", "l3", "l4", "vscode-light"]);

test("the class list covers every styled class the markup doesn't use", () => {
  expect([...styled].filter(c => !staticMarkup.has(c) && !listed.has(c))).toEqual([]);
});

test("every listed class is styled and used by the script", () => {
  const word = c => new RegExp(`(?<![\\w-])${c.replace(/[-]/g, "\\-")}(?![\\w-])`);
  expect([...listed].filter(c => !styled.has(c))).toEqual([]);
  expect([...listed].filter(c => !BUILT.has(c) && !word(c).test(script))).toEqual([]);
});

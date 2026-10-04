// Every language of the panel has every key of English, with the same {placeholders}.
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { data, expect, openPanel, status, test } from "./panel.mjs";

const here = path.dirname(fileURLToPath(import.meta.url));
const html = fs.readFileSync(path.join(here, "..", "panel", "panel.html"), "utf8");
const script = html.slice(html.indexOf(">", html.indexOf("<script nonce")) + 1, html.indexOf("</script>"));
const start = script.indexOf("{", script.indexOf("const STRINGS = {"));
let depth = 0, end = start;
for (let k = start; k < script.length; k++) {
  if (script[k] === "{") depth++;
  else if (script[k] === "}" && --depth === 0) { end = k; break; }
}
const STRINGS = new Function(`return (${script.slice(start, end + 1)})`)();
const PLURAL = /\.(zero|one|two|few|many|other)$/;
const flat = (o, p = "") => Object.entries(o).flatMap(([k, v]) => (typeof v === "object" ? flat(v, p + k + ".") : [[p + k, v]]));
const vars = s => (s.match(/\{[a-z]+\}/g) || []).sort().join();
const en = Object.fromEntries(flat(STRINGS.en));

for (const lang of Object.keys(STRINGS).filter(l => l !== "en")) {
  test(`${lang}: every string, with the same placeholders`, () => {
    const mine = Object.fromEntries(flat(STRINGS[lang]));
    const missing = Object.keys(en).filter(k => !(k in mine) && !(PLURAL.test(k) && k.replace(PLURAL, ".other") in mine));
    expect(missing).toEqual([]);
    const wrong = Object.keys(mine).filter(k => k in en && vars(mine[k]) !== vars(en[k]));
    expect(wrong).toEqual([]);
  });

  test(`${lang}: all tabs fit a 380 px panel`, async ({ page }) => {
    const st = status("en");
    st.locale = lang;
    await openPanel(page, data(st, { locale: lang }), { width: 380 });
    const bar = await page.locator(".tabs").boundingBox();
    const last = await page.getByRole("tab").last().boundingBox();
    expect(last.x + last.width).toBeLessThanOrEqual(bar.x + bar.width + 0.5);
  });
}

for (const [lang, highest] of [["en", "2,061 m"], ["de", "2.061 m"], ["pl", "2061 m"], ["es", "2061 m"]]) {
  test(`${lang}: whole numbers follow the language's rules`, async ({ page }) => {
    const st = status("en");
    st.locale = lang;
    st.global.max_elevation_m = 2061;
    await openPanel(page, data(st, { locale: lang }));
    await page.getByRole("tab").nth(4).click();
    await expect(page.locator(".stats .stat").filter({ hasText: highest }).first()).toBeVisible();
  });
}

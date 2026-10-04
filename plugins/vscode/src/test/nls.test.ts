import * as assert from "node:assert/strict";
import { catalogs, formatDistanceL, setLanguage } from "../i18n";
import * as fs from "node:fs";
import * as path from "node:path";
import { test } from "node:test";

const root = path.join(__dirname, "..", "..");
const read = (f: string) => JSON.parse(fs.readFileSync(path.join(root, f), "utf8")) as Record<string, unknown>;

void test("every %key% in package.json is translated into every language", () => {
  const keys = [...fs.readFileSync(path.join(root, "package.json"), "utf8").matchAll(/"%([\w.]+)%"/g)].map(m => m[1]);
  assert.ok(keys.length > 10);
  for (const file of ["package.nls.json", "package.nls.uk.json", "package.nls.pl.json", "package.nls.de.json", "package.nls.es.json"]) {
    const nls = read(file);
    assert.deepEqual(keys.filter(k => typeof nls[k] !== "string"), [], `${file} is missing keys`);
  }
  assert.deepEqual(Object.keys(read("package.nls.json")).sort(), Object.keys(read("package.nls.uk.json")).sort());
});

void test("every runtime language has the same {placeholders} as English", () => {
  const vars = (s: string) => (s.match(/\{\d\}/g) ?? []).sort().join();
  for (const [lang, cat] of Object.entries(catalogs)) {
    const wrong = Object.keys(catalogs.en).filter(k => vars(cat[k as keyof typeof cat]) !== vars(catalogs.en[k as keyof typeof catalogs.en]));
    assert.deepEqual(wrong, [], `${lang}: placeholders differ`);
  }
});

void test("distances use a decimal comma in Ukrainian, Polish, German and Spanish", () => {
  for (const [lang, want] of [["uk", "8,4 км"], ["pl", "8,4 km"], ["de", "8,4 km"], ["es", "8,4 km"], ["en", "8.4 km"], ["fr", "8.4 km"]]) {
    setLanguage(lang);
    assert.equal(formatDistanceL(8400), want, lang);
  }
  setLanguage("en");
});

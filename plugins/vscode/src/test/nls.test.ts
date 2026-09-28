import * as assert from "node:assert/strict";
import * as fs from "node:fs";
import * as path from "node:path";
import { test } from "node:test";

const root = path.join(__dirname, "..", "..");
const read = (f: string) => JSON.parse(fs.readFileSync(path.join(root, f), "utf8")) as Record<string, unknown>;

void test("every %key% in package.json is translated into every language", () => {
  const keys = [...fs.readFileSync(path.join(root, "package.json"), "utf8").matchAll(/"%([\w.]+)%"/g)].map(m => m[1]);
  assert.ok(keys.length > 10);
  for (const file of ["package.nls.json", "package.nls.uk.json"]) {
    const nls = read(file);
    assert.deepEqual(keys.filter(k => typeof nls[k] !== "string"), [], `${file} is missing keys`);
  }
  assert.deepEqual(Object.keys(read("package.nls.json")).sort(), Object.keys(read("package.nls.uk.json")).sort());
});

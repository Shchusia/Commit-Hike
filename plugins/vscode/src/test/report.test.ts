import { test } from "node:test";
import * as assert from "node:assert/strict";
import { scrubReport } from "../report";

void test("the developer report keeps no home folder and no e-mail", () => {
  const raw = "open /home/denis/MyProjects/x failed for denis.s+git@mail.example.com and me@x.io; /home/denis again";
  assert.equal(scrubReport(raw, "/home/denis"), "open ~/MyProjects/x failed for <email> and <email>; ~ again");
  assert.equal(scrubReport("C:\\Users\\Den\\a.json", "C:\\Users\\Den"), "~\\a.json");
  assert.equal(scrubReport("nothing personal", ""), "nothing personal");
});

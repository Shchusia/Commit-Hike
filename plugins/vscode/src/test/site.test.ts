// The routes website: addresses, links the panel may open, the panel's search as core arguments,
// and the real core searching a fake site.
import { test } from "node:test";
import * as assert from "node:assert/strict";
import * as fs from "fs";
import * as http from "http";
import * as os from "os";
import * as path from "path";
import { Cli, CliError, bundledBinary, ensureExecutable } from "../cli";
import { isRouteId, isSitePage, siteSearchArgs, siteUrl } from "../site";

void test("the site's address, and COMMIT_HIKE_SITE for tests and mirrors", () => {
  assert.equal(siteUrl({}), "https://commit-hike.dev");
  assert.equal(siteUrl({ COMMIT_HIKE_SITE: "http://127.0.0.1:9000/" }), "http://127.0.0.1:9000");
  assert.equal(siteUrl({ COMMIT_HIKE_SITE: "javascript:alert(1)" }), "https://commit-hike.dev");
});

void test("only route and author pages of the site can be opened", () => {
  const base = "https://commit-hike.dev";
  for (const ok of [`${base}/routes/svydovets-ridge`, `${base}/u/12`, `${base}/`]) assert.ok(isSitePage(ok, base), ok);
  for (const bad of ["https://evil.example/routes/x", `${base}/admin`, `${base}/routes/../admin`, "https://user:pw@commit-hike.dev/routes/x", 42])
    assert.ok(!isSitePage(bad, base), String(bad));
});

void test("the panel's search becomes core arguments; anything odd is dropped", () => {
  assert.deepEqual(siteSearchArgs({ q: " lakes ", kind: "story", length: "m", sort: "rating", page: 3 }),
    ["site", "routes", "--q", "lakes", "--kind", "story", "--length", "m", "--sort", "rating", "--page", "3"]);
  assert.deepEqual(siteSearchArgs({ q: "", kind: "--evil", sort: "drop table", page: -1 }), ["site", "routes"]);
  assert.deepEqual(siteSearchArgs("nonsense"), ["site", "routes"]);
  assert.ok(isRouteId("lake-walk") && !isRouteId("../x") && !isRouteId("A") && !isRouteId(1));
});

void test("the core searches the site and reports installs and errors", async () => {
  const server = http.createServer((req, res) => {
    if (req.url?.startsWith("/api/routes?")) {
      res.setHeader("content-type", "application/json");
      res.end(JSON.stringify({ total: 1, page: 1, pages: 1, routes: [{ id: "lake-walk", title: "Lake Walk", titles: { uk: "Озерна" }, author: "Olena",
        length_m: 5000, stops: 3, version: 1, page: "x", q: req.url }] }));
    } else { res.statusCode = 404; res.end(); }
  });
  await new Promise<void>(r => server.listen(0, "127.0.0.1", r));
  const port = (server.address() as { port: number }).port;
  process.env.COMMIT_HIKE_HOME = fs.mkdtempSync(path.join(os.tmpdir(), "ct-site-"));
  process.env.COMMIT_HIKE_SITE = `http://127.0.0.1:${port}`;
  try {
    const bin = bundledBinary(path.resolve(__dirname, "..", ".."));
    ensureExecutable(bin);
    const cli = new Cli(bin);
    cli.lang = "uk";
    const page = await cli.siteRoutes(siteSearchArgs({ q: "lake" })) as { routes: { id: string; title: string; installed?: string }[] };
    assert.equal(page.routes[0].id, "lake-walk");
    assert.equal(page.routes[0].title, "Озерна");                              // in the IDE's language
    await assert.rejects(cli.siteInstall("nowhere", false), (e: CliError) => e.code === "unknown_route");
    process.env.COMMIT_HIKE_SITE = "http://127.0.0.1:1";
    await assert.rejects(cli.siteRoutes(["site", "routes"]), (e: CliError) => e.code === "site_unreachable");
  } finally {
    delete process.env.COMMIT_HIKE_SITE;
    server.close();
  }
});

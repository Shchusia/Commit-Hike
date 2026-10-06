// Routes from the site: the panel asks the host to search and install; answers come back via window.commitHike.site.
import { data, expect, openPanel, status, test } from "./panel.mjs";

const card = (id, extra = {}) => ({
  id, title: "Title " + id, author: "Olena", length_m: 34000, stops: 9, ascent_m: 1200, real: true, gps: true,
  languages: ["en", "uk"], tags: ["carpathians", "lakes"], downloads: 33, rating: 4.6, ratings: 5, version: 2,
  page: "https://routes.example/routes/" + id, ...extra,
});
const catalogue = (routes, extra = {}) => ({ kind: "page", page: { site: "https://routes.example", total: routes.length, page: 1, pages: 1, routes, ...extra } });
const answer = (page, m) => page.evaluate(x => window.commitHike.site(x), m);
const cardOf = (page, id) => page.locator(".site-card", { hasText: "Title " + id });

async function openSite(page) {
  await page.getByRole("button", { name: "Menu" }).click();
  await page.getByRole("menuitem", { name: /Routes from the site/ }).click();
}

test("search, install, walk: everything goes through the host", async ({ page }) => {
  const p = await openPanel(page, data(status("en")));
  await openSite(page);
  expect(await p.sent()).toContainEqual({ command: "siteSearch", query: { q: "", kind: "", length: "", sort: "popular", page: 1 } });
  await expect(page.getByText("Looking…")).toBeVisible();
  await answer(page, catalogue([card("new-one"), card("mine", { installed: "local" }), card("old-one", { installed: "site", installed_version: 1, update: true }),
    card("got-it", { installed: "site", installed_version: 2 }), card("demo-trail", { installed: "builtin" })]));
  await expect(page.getByText("5 routes")).toBeVisible();
  await expect(cardOf(page, "new-one").getByRole("button", { name: "Install" })).toBeVisible();
  await expect(cardOf(page, "old-one").getByRole("button", { name: "Update to version 2" })).toBeVisible();
  await expect(cardOf(page, "got-it").getByText("✓ Installed")).toBeVisible();
  await expect(cardOf(page, "demo-trail").getByText("Built in")).toBeVisible();
  await expect(cardOf(page, "new-one")).toContainText("by Olena");
  await expect(cardOf(page, "new-one")).toContainText("#carpathians #lakes");
  await expect(page.getByText(/Searches the public catalogue at routes\.example/)).toBeVisible();

  await cardOf(page, "new-one").getByRole("button", { name: "Install" }).click();
  await expect(cardOf(page, "new-one").getByRole("button", { name: "Installing…" })).toBeDisabled();
  expect(await p.sent()).toContainEqual({ command: "siteInstall", id: "new-one", replaceLocal: false });
  await answer(page, { kind: "installed", id: "new-one", version: 2 });
  await cardOf(page, "new-one").getByRole("button", { name: "Walk it now" }).click();
  expect(await p.sent()).toContainEqual({ command: "startRoute", id: "new-one" });

  page.once("dialog", d => d.accept());                                      // replacing your own route asks first
  await cardOf(page, "mine").getByRole("button", { name: "Replace my route" }).click();
  expect(await p.sent()).toContainEqual({ command: "siteInstall", id: "mine", replaceLocal: true });
  await answer(page, { kind: "installError", id: "mine", error: { code: "invalid_route", message: "locales/en.json is missing: name" } });
  await expect(cardOf(page, "mine")).toContainText("locales/en.json is missing: name");

  await cardOf(page, "got-it").getByRole("button", { name: /On the site/ }).click();
  expect(await p.sent()).toContainEqual({ command: "openUrl", url: "https://routes.example/routes/got-it" });
  expect(p.errors).toEqual([]);
});

test("filters, search and pages ask the host again", async ({ page }) => {
  const p = await openPanel(page, data(status("en")));
  await openSite(page);
  await answer(page, catalogue([card("a")], { total: 30, pages: 2 }));
  await page.getByRole("combobox", { name: "Kind" }).selectOption("story");
  expect((await p.sent()).at(-1)).toEqual({ command: "siteSearch", query: { q: "", kind: "story", length: "", sort: "popular", page: 1 } });
  await answer(page, catalogue([card("a")], { total: 30, pages: 2 }));
  await page.getByRole("searchbox").fill("lakes");
  await page.getByRole("searchbox").press("Enter");
  expect((await p.sent()).at(-1).query).toMatchObject({ q: "lakes", kind: "story", page: 1 });
  await answer(page, catalogue([card("a")], { total: 30, pages: 2 }));
  await expect(page.getByText("Page 1 of 2")).toBeVisible();
  await page.getByRole("button", { name: "Next →" }).click();
  expect((await p.sent()).at(-1).query).toMatchObject({ q: "lakes", page: 2 });
});

test("offline: says so and retries", async ({ page }) => {
  const p = await openPanel(page, data(status("uk")));
  await page.getByRole("button", { name: "Меню" }).click();
  await page.getByRole("menuitem", { name: /Маршрути з сайту/ }).click();
  await answer(page, { kind: "error", error: { code: "site_unreachable", message: "dial tcp: no route" } });
  await expect(page.getByText(/Не вдається зв'язатися із сайтом маршрутів/)).toBeVisible();
  await page.getByRole("button", { name: "Спробувати ще" }).click().catch(() => page.getByRole("button", { name: /ще раз|Retry/ }).click());
  expect((await p.sent()).filter(m => m.command === "siteSearch").length).toBe(2);
  await page.getByRole("button", { name: /Назад/ }).first().click();
  await expect(page.getByRole("tab").first()).toBeVisible();                // back on the trail
});

test("the host can open it, e.g. from a command", async ({ page }) => {
  const st = status("en");
  const p = await openPanel(page, data(st));
  await p.update(data(st, { open_view: "site", open_token: 7 }));
  expect(await p.sent()).toContainEqual({ command: "siteSearch", query: { q: "", kind: "", length: "", sort: "popular", page: 1 } });
  await expect(page.getByRole("heading", { name: "Routes from the site" })).toBeVisible();
});

test("the site's editor and home from the menu; closing the menu repaints the page", async ({ page }) => {
  const p = await openPanel(page, data(status("en")));
  await page.getByRole("button", { name: "Menu" }).click();
  await expect(page.getByRole("menuitem", { name: /route template/i })).toHaveCount(0);
  await page.getByRole("menuitem", { name: "Create a route on the site…" }).click();
  expect(await p.sent()).toContainEqual({ command: "openSite", path: "/create" });
  await page.getByRole("button", { name: "Menu" }).click();
  await page.getByRole("menuitem", { name: /commit-hike\.dev/ }).click();
  expect(await p.sent()).toContainEqual({ command: "openSite", path: "/" });
  // the one-frame repaint after closing ends with the page fully opaque again
  await expect.poll(() => page.evaluate(() => document.body.style.opacity)).toBe("");
});

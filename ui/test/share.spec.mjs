// Sharing: the menu, the post text, the URL shapes the hosts allow, and the
// postcard put into the clipboard (the host's, when the browser's is closed).
import { data, expect, openPanel, pngSize, status, test } from "./panel.mjs";

async function openMenu(page) {
  await page.getByRole("button", { name: /Share/ }).first().click();
  await expect(page.getByRole("menu", { name: "Share where you are" })).toBeVisible();
}

test("share on a network: the postcard to the clipboard, then the post with the text", async ({ page }) => {
  const st = status("en");
  const p = await openPanel(page, data(st));
  await openMenu(page);
  await page.locator('[data-net="x"]').click();
  await expect.poll(async () => (await p.sent()).map(m => m.command)).toEqual(["copyImage", "openUrl"]);
  const [copy, open] = await p.sent();
  expect(pngSize(copy.data)).toMatchObject({ png: true, width: 1200, height: 630 });
  const url = new URL(open.url);
  expect(url.origin + url.pathname).toBe("https://twitter.com/intent/tweet");
  const j = st.project || st.global;
  expect(url.searchParams.get("text")).toContain(j.route.name);
  expect(url.searchParams.get("text")).toContain("#CommitHike");
  expect(url.searchParams.get("url")).toBe("https://github.com/Shchusia/Commit-Hike");
  await expect(page.getByRole("menu")).toHaveCount(0); // closed after the choice
  expect(p.errors).toEqual([]);
});

test("every network opens a known sharing URL", async ({ page }) => {
  const p = await openPanel(page, data(status("en")));
  const shapes = {
    bluesky: "https://bsky.app/intent/compose", threads: "https://www.threads.net/intent/post",
    linkedin: "https://www.linkedin.com/sharing/share-offsite/", facebook: "https://www.facebook.com/sharer/sharer.php",
    telegram: "https://t.me/share/url", reddit: "https://www.reddit.com/submit",
  };
  for (const [net, shape] of Object.entries(shapes)) {
    await page.evaluate(() => { window.__msgs = []; });
    await openMenu(page);
    await page.locator(`[data-net="${net}"]`).click();
    await expect.poll(async () => (await p.sent()).find(m => m.command === "openUrl")?.url || "").toContain(shape);
  }
  expect(p.errors).toEqual([]);
});

test("Mastodon goes to the server you name, and only a real host name", async ({ page }) => {
  const p = await openPanel(page, data(status("uk")));
  await page.getByRole("button", { name: /Поділитися/ }).first().click();
  const server = page.getByRole("textbox", { name: "Твій сервер Mastodon" });
  await server.fill("javascript:alert(1)");
  await page.locator('[data-net="mastodon"]').click();
  expect((await p.sent()).filter(m => m.command === "openUrl")).toEqual([]);
  await server.fill("https://Social.Example.org/@me");
  await page.locator('[data-net="mastodon"]').click();
  await expect.poll(async () => (await p.sent()).find(m => m.command === "openUrl")?.url || "").toMatch(/^https:\/\/social\.example\.org\/share\?text=/);
  const url = new URL((await p.sent()).find(m => m.command === "openUrl").url);
  expect(url.searchParams.get("text")).toMatch(/моїми комітами/);
  expect(p.errors).toEqual([]);
});

test("save and copy the text from the same menu", async ({ page }) => {
  const p = await openPanel(page, data(status("en")));
  await openMenu(page);
  await page.getByRole("menuitem", { name: /Copy the text/ }).click();
  await expect(page.getByRole("status").filter({ hasText: "The text is in your clipboard." })).toBeVisible();
  await openMenu(page);
  await page.getByRole("menuitem", { name: /Save the postcard/ }).click();
  await expect.poll(async () => (await p.sent()).find(m => m.command === "savePostcard")?.name || "").toMatch(/^commit-hike-.+-day-\d+\.png$/);
  await openMenu(page);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("menu")).toHaveCount(0);
  expect(p.errors).toEqual([]);
});

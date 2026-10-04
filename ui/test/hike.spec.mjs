import { data, expect, openPanel, pngSize, status, test } from "./panel.mjs";

// The released (prod) build never shows what lies ahead; a dev build may.
for (const dev of [false, true]) {
  test(`looking ahead ${dev ? "works in a dev build" : "is blocked in a prod build"}`, async ({ page }) => {
    const st = status("en");
    st.global.distance_m = 6000; // mid-route, so there is something ahead
    await openPanel(page, data(st, { dev }));
    const scene = page.locator(".scene");
    await scene.focus();
    for (let i = 0; i < 4; i++) await page.keyboard.press("Shift+ArrowRight");
    await page.waitForTimeout(700);
    if (dev) await expect(page.locator(".scene .recenter")).toBeVisible();
    else await expect(page.locator(".scene .recenter")).toBeHidden();
  });
}

test("the postcard is a 1200×630 PNG handed to the host", async ({ page }) => {
  const p = await openPanel(page, data(status("en")));
  await page.locator(".hike-actions .share").click();
  await page.getByRole("menuitem", { name: /Save the postcard/ }).click();
  await expect.poll(async () => (await p.sent()).some(m => m.command === "savePostcard")).toBe(true);
  const m = (await p.sent()).find(x => x.command === "savePostcard");
  expect(m.name).toMatch(/^commit-hike-chornohora-ridge-day-\d+\.png$/);
  expect(pngSize(m.data)).toEqual({ png: true, width: 1200, height: 630 });
  expect(await page.locator("body > .scene").count()).toBe(0); // the off-screen scene is gone
  expect(p.errors).toEqual([]);
});

import { data, expect, openPanel, status, test } from "./panel.mjs";

test.beforeEach(async ({ page }) => {
  await openPanel(page, data(status("en")));
  await page.getByRole("tab", { name: "Map" }).click();
});

test("zooming changes the scale bar, not the size of symbols", async ({ page }) => {
  const label = page.locator(".mapscale span");
  const before = await label.textContent();
  const sym = page.locator(".map-svg .sym").first();
  const w0 = (await sym.boundingBox()).width;
  for (let i = 0; i < 3; i++) await page.getByRole("button", { name: "Zoom in" }).click();
  await expect(label).not.toHaveText(before);
  expect(Math.abs((await sym.boundingBox()).width - w0)).toBeLessThan(1.5);
});

test("labels stay inside the map", async ({ page }) => {
  const box = await page.locator(".mapbox").boundingBox();
  for (const l of await page.locator(".map-svg .label").all()) {
    if (!(await l.isVisible())) continue;
    const b = await l.boundingBox();
    expect(b.x).toBeGreaterThanOrEqual(box.x - 1);
    expect(b.x + b.width).toBeLessThanOrEqual(box.x + box.width + 1);
  }
});

test("Chornohora is drawn from its GPS track: Zaroslyak north-west of Pip Ivan", async ({ page }) => {
  const syms = page.locator(".map-svg .sym");
  const first = await syms.first().boundingBox(), last = await syms.last().boundingBox();
  expect(first.x).toBeLessThan(last.x);
  expect(first.y).toBeLessThan(last.y);
});

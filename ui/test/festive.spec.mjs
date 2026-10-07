import { data, expect, openPanel, status, test } from "./panel.mjs";

// Holidays are computed from the date for any year: no release is needed for
// next year's Easter or lunar new year.
test("the calendar finds the holidays of any year", async ({ page }) => {
  await openPanel(page, data(status("en")));
  const on = day => page.evaluate(d => window.commitHike.festivals(d), day);
  const cases = {
    "2026-12-25": "christmas", "2027-01-07": "christmas", "2031-01-01": "christmas",
    "2026-04-05": "easter", "2026-04-12": "easter", "2031-04-13": "easter", "2035-04-29": "easter",
    "2027-02-06": "lunar", "2030-02-03": "lunar", "2025-10-21": "diwali", "2027-10-29": "diwali", "2024-03-25": "holi",
    "2026-12-06": "hanukkah", "2027-12-26": "hanukkah", "2026-03-01": "ramadan", "2026-10-31": "halloween",
    "2026-02-14": "valentine", "2026-03-17": "patrick", "2026-06-24": "kupala", "2026-08-24": "ukraine",
  };
  for (const [day, id] of Object.entries(cases)) expect(await on(day), day).toContain(id);
  expect(await on("2026-07-15")).toEqual([]);
  expect(await on("2026-09-10")).toEqual([]);
});

// Every celebration dresses the scene, day and night, without errors.
const DAYS = ["2026-12-25T12:00:00Z", "2026-12-31T23:30:00Z", "2026-04-12T10:00:00Z", "2026-10-31T21:00:00Z", "2027-02-06T20:00:00Z",
  "2027-10-29T20:00:00Z", "2026-02-14T12:00:00Z", "2026-03-17T12:00:00Z", "2026-06-23T22:00:00Z", "2026-08-24T21:00:00Z",
  "2024-03-25T12:00:00Z", "2026-12-06T19:00:00Z", "2026-03-01T22:00:00Z", "2026-03-21T21:00:00Z"];
for (const when of DAYS) {
  test(`a holiday scene on ${when}`, async ({ page }) => {
    const st = status("en");
    for (const type of ["forest", "tropical", "desert", "city"]) {
      st.global.route.biomes = [{ at_m: 0, type }];
      const p = await openPanel(page, data(st));
      await page.clock.setFixedTime(new Date(when));
      await p.update(data(st));
      await page.waitForTimeout(120);
      expect(p.errors, `${type} ${when}`).toEqual([]);
    }
    await expect(page.locator(".scene canvas.fx")).toHaveCount(1);
  });
}

test("with holidays turned off, a holiday looks like any other day", async ({ page }) => {
  const st = status("en");
  st.settings = { ...st.settings, festive: "off", ambient: "off" };
  const shot = async () => page.locator(".scene canvas").first().screenshot();
  const p = await openPanel(page, data(st));
  await page.clock.setFixedTime(new Date("2026-12-25T12:00:00Z")); await p.update(data(st)); await page.waitForTimeout(100);
  const off = await shot();
  st.settings.festive = "on"; await p.update(data(st)); await page.waitForTimeout(100);
  const on = await shot();
  expect(on.equals(off)).toBe(false);
  expect(p.errors).toEqual([]);
});

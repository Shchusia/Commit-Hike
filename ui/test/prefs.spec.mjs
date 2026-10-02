import { data, expect, openPanel, status, test } from "./panel.mjs";

const classes = page => page.evaluate(() => document.documentElement.className.split(" ").filter(Boolean).sort());

test("motion and contrast follow the settings", async ({ page }) => {
  const st = status("en");
  const p = await openPanel(page, data(st));
  expect(await classes(page)).toEqual([]);
  st.settings = { reduce_motion: "on", high_contrast: "on", notifications: "all" };
  await p.update(data(st));
  expect(await classes(page)).toEqual(["hc", "reduce-motion"]);
  const transition = await page.getByRole("tab").first().evaluate(e => getComputedStyle(e).transitionDuration);
  expect(parseFloat(transition)).toBeLessThan(0.01);
});

test("auto follows the system, and off overrides it", async ({ page }) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  const st = status("en");
  const p = await openPanel(page, data(st));
  expect(await classes(page)).toEqual(["reduce-motion"]);
  st.settings = { reduce_motion: "off", high_contrast: "auto", notifications: "all" };
  await p.update(data(st));
  expect(await classes(page)).toEqual([]);
});

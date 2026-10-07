import { data, expect, openPanel, status, test } from "./panel.mjs";

// Biomes added in 1.1.0: each one draws (day and night, in every season)
// without errors, its own way, and has a name on the map legend.
const NEW = ["jungle", "savanna", "canyon", "glacier", "heath", "bamboo", "sakura", "tropical", "wasteland", "enchanted", "city", "ruins",
  "birch", "lavender", "vineyard", "rice", "mangrove", "saltflat", "reef", "seaice", "taiga", "underwater", "moon", "mars"];
const ALL = ["forest", "grove", "meadow", "fields", "steppe", "desert", "rock", "snow", "tundra", "water", "swamp", "coast", "volcanic", "village", ...NEW];
const KINDS = ["church", "temple", "monastery", "windmill", "waterfall", "shrine", "statue", "gate", "mine", "lair", "battlefield", "wreck", "oasis", "inn"];

async function sceneHash(page) {
  return page.evaluate(() => {
    const c = document.querySelector(".scene canvas");
    const px = c.getContext("2d").getImageData(0, 0, c.width, c.height).data;
    let h = 0;
    for (let i = 0; i < px.length; i += 97) h = (h * 31 + px[i]) >>> 0;
    return h;
  });
}

for (const when of ["2026-07-15T13:00:00Z", "2026-01-10T23:30:00Z", "2026-04-10T09:00:00Z"]) {
  test(`every new biome draws at ${when}`, async ({ page }) => {
    const seen = new Set();
    for (const type of NEW) {
      const st = status("en");
      st.global.route.biomes = [{ at_m: 0, type }];
      const p = await openPanel(page, data(st));
      await page.clock.setFixedTime(new Date(when));
      await p.update(data(st));
      await page.waitForTimeout(150);
      seen.add(await sceneHash(page));
      expect(p.errors, type).toEqual([]);
    }
    expect(seen.size).toBe(NEW.length); // no two look the same
  });
}

test("the map legend names the new biomes", async ({ page }) => {
  const st = status("uk");
  st.global.route.biomes = NEW.map((type, i) => ({ at_m: i * 500, type }));
  const p = await openPanel(page, data(st));
  await p.tab(1);
  const legend = page.locator(".legend");
  for (const name of ["Джунглі", "Каньйон", "Льодовик", "Зачарований ліс", "Місто", "Руїни"]) await expect(legend).toContainText(name);
  expect(p.errors).toEqual([]);
});

// A long walk through one biome passes set pieces, thickets and clearings and
// far-off silhouettes; none of them may break the scene, by day or by night.
for (const when of ["2026-07-15T13:00:00Z", "2026-12-20T23:30:00Z"]) {
  test(`a long walk through every biome at ${when}`, async ({ page }) => {
    test.setTimeout(120000);
    const st = status("en");
    const L = st.global.route.length_m;
    st.global.route.biomes = [{ at_m: 0, type: "meadow" }];
    const p = await openPanel(page, data(st));
    await page.clock.setFixedTime(new Date(when));
    for (const type of ALL) {
      st.global.route.biomes = [{ at_m: 0, type }];
      for (const k of [0.15, 0.35, 0.55, 0.8]) {
        st.global.distance_m = L * k;
        await p.update(data(st));
      }
      await page.waitForTimeout(60);
      expect(p.errors, type).toEqual([]);
    }
  });
}

test("the new kinds of stops have map symbols and scene landmarks", async ({ page }) => {
  const st = status("en");
  const L = st.global.route.length_m, wps = st.global.route.waypoints;
  KINDS.forEach((kind, i) => { if (wps[i + 1]) wps[i + 1].kind = kind; else wps.push({ id: "k" + i, kind, at_m: L * (i + 1) / (KINDS.length + 2), name: kind, text: kind }); });
  const p = await openPanel(page, data(st));
  for (let i = 1; i < Math.min(wps.length, KINDS.length + 1); i++) {
    st.global.distance_m = Math.max(0, wps[i].at_m - 30);
    await p.update(data(st));
  }
  await p.tab(1);
  await expect(page.locator(".map-svg .sym").first()).toBeVisible();
  expect(p.errors).toEqual([]);
});

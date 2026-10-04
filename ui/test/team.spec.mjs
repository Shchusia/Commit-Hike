// The Team tab: the team's week and the goal walked together.
import { data, expect, openPanel, status, test } from "./panel.mjs";

const ROUTES = [
  { id: "chornohora-ridge", name: "Chornohora Ridge", length_m: 24000 },
  { id: "camino-frances", name: "Camino Francés", length_m: 772000 },
];
const WEEK = {
  from: "2026-09-28", to: "2026-10-04", total_m: 25300, commits: 14, active_days: 3, prev_total_m: 20000,
  best_day: "2026-09-28", best_day_m: 12000, days_elapsed: 3,
  members: [
    { id: "aaa", name: "Anna Koval", distance_m: 15300, commits: 9, active_days: 3 },
    { id: "me1", name: "Me", me: true, distance_m: 10000, commits: 5, active_days: 2 },
  ],
};
const GOAL = {
  route_id: "camino-frances", route_name: "Camino Francés", length_m: 772000, since: 1790553600,
  distance_m: 120000, percent: 15.5, next_waypoint: { id: "burgos", name: "Burgos", at_m: 130000 }, to_next_m: 10000,
  pace_m: 9000, days_left: 72,
  members: [{ id: "aaa", name: "Anna Koval", distance_m: 70000 }, { id: "me1", name: "Me", me: true, distance_m: 50000 }],
};

function teamData(team, lang = "en") {
  const st = status(lang);
  st.team = true;
  return data(st, { team: { scope: "global", route_id: "chornohora-ridge", length_m: 24000, members: [], routes: ROUTES, week: WEEK, ...team } });
}

test("the week: total, trend, days, best day and everyone's part", async ({ page }) => {
  const p = await openPanel(page, teamData({}));
  await p.tab(3);
  const week = page.locator(".team-week");
  await expect(week).toContainText("This week · Sep 28 – Oct 4");
  await expect(week).toContainText("25.3 km");
  await expect(week).toContainText("↑ 27% on last week");
  await expect(week).toContainText("3 of 3 days with commits");
  await expect(week).toContainText("Best day: Monday, 12.0 km");
  await expect(week.locator(".week-rows .nm")).toHaveText(["Anna Koval", "Me (you)"]);
  expect(p.errors).toEqual([]);
});

test("a week without commits says so", async ({ page }) => {
  const p = await openPanel(page, teamData({ week: { ...WEEK, total_m: 0, commits: 0, active_days: 0, members: [], best_day: undefined } }, "uk"));
  await p.tab(3);
  await expect(page.locator(".team-week")).toContainText("Цього тижня комітів ще не було.");
  await expect(page.locator(".team-week")).toContainText("Минулого тижня: 20,0 км");
  expect(p.errors).toEqual([]);
});

test("no goal yet: pick a route and start it", async ({ page }) => {
  const p = await openPanel(page, teamData({}));
  await p.tab(3);
  const goal = page.locator(".team-goal");
  await expect(goal).toContainText("Walk a route together");
  await goal.getByRole("combobox", { name: "Route" }).selectOption("camino-frances");
  await goal.getByRole("button", { name: "Start" }).click();
  expect((await p.sent()).filter(m => m.command === "setTeamGoal")).toEqual([{ command: "setTeamGoal", route: "camino-frances" }]);
  expect(p.errors).toEqual([]);
});

test("a goal: progress together, who walked what, pace; change or stop it", async ({ page }) => {
  const p = await openPanel(page, teamData({ goal: GOAL }, "de"));
  await p.tab(3);
  const goal = page.locator(".team-goal");
  await expect(goal).toContainText("Camino Francés — 120 km von 772 km gemeinsam (15,5%)");
  await expect(goal).toContainText("Nächster Halt: Burgos, noch 10,0 km");
  await expect(goal).toContainText("~72 Tage bis zum Ziel");
  await expect(goal.locator(".goal-bar i")).toHaveCount(2);
  await expect(goal.locator(".goal-legend span")).toHaveText(["Anna Koval 70,0 km", "du 50,0 km"]);
  await goal.getByRole("button", { name: "Andere Route" }).click();
  await goal.getByRole("combobox").selectOption("chornohora-ridge");
  await goal.getByRole("button", { name: "Starten" }).click();
  await goal.getByRole("button", { name: "Ziel beenden" }).click();
  expect((await p.sent()).filter(m => m.command === "setTeamGoal").map(m => m.route)).toEqual(["chornohora-ridge", ""]);
  expect(p.errors).toEqual([]);
});

test("a finished goal", async ({ page }) => {
  const p = await openPanel(page, teamData({ goal: { ...GOAL, distance_m: 772000, percent: 100, finished: true, days_left: 0 } }, "es"));
  await p.tab(3);
  await expect(page.locator(".team-goal")).toContainText("¡Recorrida juntos! ✓");
  expect(p.errors).toEqual([]);
});

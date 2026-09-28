// When to ask for a rating. Pure logic, no VS Code APIs, so it's unit-tested.
//
// Marketplaces don't tell an extension whether someone left a review, so
// "rated" means the user pressed "Rate it": after that we never ask again.
// We ask only people who really use Commit Hike (a week since install, commits
// on at least 3 days), only right after a commit moved them along the trail,
// at most 3 times, and never again after "Don't ask again".

export interface RatingState {
  firstSeen: number;          // ms since epoch
  activeDays: string[];       // YYYY-MM-DD with progress, most recent last
  status: "pending" | "rated" | "never";
  snoozedUntil: number;       // ms since epoch
  asks: number;               // how many times we asked
}

export const DAY_MS = 86_400_000;
export const MIN_AGE_DAYS = 7;
export const MIN_ACTIVE_DAYS = 3;
export const MAX_ASKS = 3;
export const SNOOZE_DAYS = 7;

export function initial(now: number): RatingState {
  return { firstSeen: now, activeDays: [], status: "pending", snoozedUntil: 0, asks: 0 };
}

/** Normalizes stored state (older or damaged values fall back to a fresh start). */
export function load(raw: unknown, now: number): RatingState {
  const s = raw as Partial<RatingState> | undefined;
  if (!s || typeof s.firstSeen !== "number") return initial(now);
  return {
    firstSeen: s.firstSeen,
    activeDays: Array.isArray(s.activeDays) ? s.activeDays.filter(d => typeof d === "string").slice(-30) : [],
    status: s.status === "rated" || s.status === "never" ? s.status : "pending",
    snoozedUntil: typeof s.snoozedUntil === "number" ? s.snoozedUntil : 0,
    asks: typeof s.asks === "number" ? s.asks : 0,
  };
}

/** Records a commit that moved the user along the trail today. */
export function recordProgress(s: RatingState, now: number): RatingState {
  const day = new Date(now).toISOString().slice(0, 10);
  if (s.activeDays[s.activeDays.length - 1] === day) return s;
  return { ...s, activeDays: [...s.activeDays, day].slice(-30) };
}

export function shouldAsk(s: RatingState, now: number): boolean {
  return s.status === "pending"
    && now - s.firstSeen >= MIN_AGE_DAYS * DAY_MS
    && s.activeDays.length >= MIN_ACTIVE_DAYS
    && now >= s.snoozedUntil
    && s.asks < MAX_ASKS;
}

/** Called when the prompt is shown: closing it without an answer counts as "later". */
export function asked(s: RatingState, now: number): RatingState {
  return { ...s, asks: s.asks + 1, snoozedUntil: now + SNOOZE_DAYS * DAY_MS };
}

export function answered(s: RatingState, answer: "rate" | "later" | "never"): RatingState {
  if (answer === "rate") return { ...s, status: "rated" };
  if (answer === "never") return { ...s, status: "never" };
  return s; // "later": asked() already snoozed it
}

// Where reviews go. Microsoft's VS Code reads the Visual Studio Marketplace;
// VSCodium, Cursor, Windsurf and friends read Open VSX.
export const VS_MARKETPLACE_LIVE = false; // flip to true once Commit Hike is on the Visual Studio Marketplace
export const OPEN_VSX_REVIEWS = "https://open-vsx.org/extension/shchusia/commit-hike/reviews";
export const VS_MARKETPLACE_REVIEWS = "https://marketplace.visualstudio.com/items?itemName=shchusia.commit-hike&ssr=false#review-details";

export function reviewTarget(appName: string): { url: string; store: string } {
  return /^Visual Studio Code/i.test(appName) && VS_MARKETPLACE_LIVE
    ? { url: VS_MARKETPLACE_REVIEWS, store: "Visual Studio Marketplace" }
    : { url: OPEN_VSX_REVIEWS, store: "Open VSX" };
}

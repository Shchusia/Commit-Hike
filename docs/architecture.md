# Architecture

## Principles

1. **One engine, thin plugins.** All counting, storage, route logic, achievements, teammates and translations of route content live in the Go core. Plugins only detect commits, call the core and render its answer. A new IDE costs a thin adapter, not a rewrite.
2. **Data, not code, for content.** Routes, elevation profiles, facts, objects, stories, achievements and translations are JSON files and pictures. Adding a route never touches Go code.
3. **Read-only towards repositories, local-only towards the world.** The core only runs `git log`, `git rev-parse` and `git merge-base`. No network, no hooks, no files in projects.
4. **A versioned contract.** Plugins and core talk through a JSON envelope with an `api` version (see protocol.md).
5. **Core returns data and codes, plugins own UI text.** Error codes, not sentences; route texts come translated from route packs.

## Layers (core/internal)

```
cmd/commit-hike      main: wires the data dir and calls cli.Run
cli               args -> use case -> one JSON envelope (no logic)
app               use cases: Init, Scan, Verify, Scoring, Status, Team, SetJourney, Locale, routes, Render
  ├─ routes       route packs: loading, validation, elevation profile, assets, translated texts
  ├─ achievements rule engine (distance, waypoint, finish, commits, streak, day_distance, altitude, climb)
  ├─ render       trail geometry (scene JSON) and SVG
  ├─ i18n         locale chains (uk-UA -> uk -> en) and catalogs
  ├─ score        commit size -> meters, the walking-speed limit (and the older per-commit rules)
  ├─ filter       which files count
  ├─ gitlog       read-only git access (log with .mailmap identities, HEAD, ancestry)
  └─ store        local files, locking, atomic writes, HMAC ids
protocol          JSON types shared with plugins (a leaf package)
content           built-in route packs, embedded into the binary
```

Dependencies point downward only. `app` is the only package that knows about all others; `cli` knows only `app` and `protocol`.

## Counting commits correctly

- **Git runs without the lock.** A scan reads config and state, runs `git log`, and only then takes the lock, re-reads state and merges. A big repository never blocks other IDEs. The lock file is refreshed while held, so a live lock never looks stale; a crashed owner's lock expires after 30 s.
- **Rewritten history is recounted.** Commits are keyed by HMAC(author email | author time), which survives amend, rebase and cherry-pick. Every scan also re-checks the last 7 days (by committer date, with a day of slack) and drops records git no longer has. If the plugin reports a previous HEAD that is not an ancestor of the current one, the whole project is recounted.
- **Clones don't fight.** Each record remembers which work tree counted it (HMAC of the path); only that clone may drop it.
- **The project follows a new root.** A project's id is an HMAC of its root commit. The state remembers which id each work tree had; when it changes (the first commit was amended or everything was squashed), records, achievements, the project journey and settings move to the new id. A different repository cloned into a known path is told apart by sharing no commits with the old one and by the old one being known elsewhere.
- **The user's calendar.** "Today", streaks and the daily cap of the older rules use the local time zone. Commits dated more than a day in the future are ignored.

## Pace and difficulty

A commit's size is its significant changed lines (added + deleted, after the
filters). Since 1.2 the state stores them per commit (`l`) next to the base
points of the older per-commit rules (`m`). Meters are computed when read.

**Volume rules (1.2 on).** A commit is worth 45·log2(1 + lines/10) base points,
at most 300 (reached at about 1,000 lines): a one-line commit is 6.2, 60 lines
126.3. Base points × pace (13) × the difficulty of the time the commit was made
(easy ×1.25, medium ×1, hard ×0.8) give meters. Then the walking-speed limit:
all the user's commits in the tracked projects go, in time order, through one
walker whose reserve of trail refills at 5 km an hour up to 10 km (both scaled
by the difficulty); a commit gets at most what the reserve holds. A typical day
of about 6 commits of ~60 lines is 9.9 km on medium and never meets the limit,
roughly half of a hiker's day, so real-length routes take months (Frodo's
2,863 km take ~290 typical days on medium). One commit gives at most 3.9 km
(4.9 km on easy): no route can be finished with a couple of commits. A burst of
commits is cut to the reserve, and splitting work into tiny commits gains
little (60 one-line commits are worth about 3 commits of 60 lines). Commits
spread over a working day count in full however late they come: there is no
daily cap.

**Per-commit rules (before 1.2).** 10 + 20·log2(1 + lines) base points, at
most 200, × pace × difficulty; per day, meters count in full up to 15 km, at
35% up to 30 km and at 5% beyond (scaled by the level).

**Which rules apply.** `scoring_from` in `config.json` says from when the volume
rules count. New installs set it to 0: their whole history counts by volume. A
config written before 1.2 gets it set to the moment of its first write after
the upgrade (and `scoring_upgraded_at` to the same), so distance already walked
stays as it was. `scoring --history on` moves it back to where the real pace
starts (`pace_from`), recounting the history by volume; `--history off` moves
it back to `scoring_upgraded_at`. Records stored before 1.2 have no lines: they
are recovered from their base points (exactly up to ~723 lines, 723 beyond)
until a scan or `scoring --repo`/`verify` re-reads the project and stores them.

A difficulty change applies from its moment on (`config.json` keeps the
history of changes), so switching never rewrites distance already walked. For
the same reason, a config written before the pace existed gets `pace_from` set
on its first write: older commits keep 1 base point = 1 m. New installs use the
pace for their whole history.

## Teammates

`team` reads the project's history (after `.mailmap`) and places every author on the same route with the same rules as the user. Nothing about other people is stored; the result is computed on request and plugins cache it for a few minutes or until the next scan. It's off by default and turned on per project.

## The panel

`ui/panel/panel.html` is one self-contained file used by both IDEs:

- **Hike:** in release builds the camera looks back but not ahead (development builds may look ahead). Layered canvases (sky by the user's clock with day-seeded weather; three distant ranges whose shapes blend by biome; the ground from a monotone-cubic elevation profile; per-biome plants picked by weight so biomes blend; tunnels with a rock vault and torchlight; a cold, dim chill near encounters; a fast foreground) and a DOM layer for route objects, which fade by the walker's distance. Below: the elevation profile strip.
- **Minimap:** a corner overview of the whole trail over the same geometry as the map, updated as the walker moves; a click opens the map.
- **Map:** terrain generated from the route itself (a height field from the profile and distance to the trail, smoothed, hill-shaded, with marching-squares contours and shorelines), or the pack's `map_image` in its own proportions, with an SVG overlay (trail, stops, found objects, fact pins, teammates), zoom and pan.
- **Places, Team, Stats** tabs and a menu with language, routes, the hiker icon and teammates.

HTML objects are sanitized in a `<template>`, rendered in a closed shadow root, and the CSP allows only the page's own nonce-carrying scripts.

## Extension points

| You want to… | Do this |
|---|---|
| add a route | new folder in `core/content/routes` or in the user's `routes` data dir |
| add a language | `locales/<lang>.json` in each route, `STRINGS` in the panel, `src/i18n.ts` and `package.nls.<lang>.json` in VS Code, `I18n.kt` in JetBrains (menu items included) |
| add a biome | `BiomeTypes` in `routes`, plus colours, a distant shape and plants in the panel's `BIOMES` / `drawDeco` |
| add an achievement type | a case in `achievements.Rule.Met` and `validate` |
| support a new IDE | a plugin that calls the CLI and hosts `ui/panel` |
| draw elsewhere (terminal, README badge) | `commit-hike render --format svg` or `--format scene` |
| run as a service later | put a server in front of `app.Service`; no logic changes |

## Data on disk

`~/.config/commit-hike` (Linux), `~/Library/Application Support/commit-hike` (macOS), `%AppData%\commit-hike` (Windows), overridable with `COMMIT_HIKE_HOME`:

- `config.json`: mode, emails, locale, journeys, projects with teammates turned on, difficulty changes, `pace_from`, `scoring_from`, `scoring_upgraded_at`
- `state.json`: counted commits (project id, time, meters, which clone), work trees seen (HMAC of path → project id) and unlocked achievements
- `key`: random key for HMAC ids
- `avatar.png`: optional custom hiker
- `routes/`: optional user route packs

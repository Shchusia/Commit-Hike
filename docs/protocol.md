# Core protocol (api 1)

Plugins run `commit-hike <command> [flags]` and read one JSON document from stdout. Warnings go to stderr and never break stdout.

```json
{ "api": 1, "ok": true, "data": { } }
{ "api": 1, "ok": false, "error": { "code": "not_initialized", "message": "…" } }
```

Error codes: `not_initialized`, `invalid_argument`, `unknown_route`, `not_a_repo`, `busy`, `invalid_route`, `route_exists`, `route_in_use`, `invalid_image`, `internal`. Show your own translated text per code; `message` is for logs.

Every command accepts `--lang` (the IDE's language, e.g. `uk`). A language fixed with `locale --set` wins.

| Command | Returns |
|---|---|
| `init [--difficulty easy/medium/hard]` | config; a difficulty chosen here applies to the whole history |
| `difficulty [--set easy/medium/hard]` | `{level, levels, typical_day_m: {easy, medium, hard}}`; a change applies to commits from now on |
| `scan --repo [--prev-head SHA]` | status + `new_commits`, `updated_commits`, `removed_commits`, `rewritten`, `added_m`, `events` |
| `status [--repo]` | status: `global` and `project` journeys, `today_m`, `total_m`, `locale`, `team`, `difficulty`, `typical_day_m` (a typical day of commits at that level); for an untracked project `tracked: false` and `reason_code` |
| `team --repo` | everyone who commits to the project on the same route (see below) |
| `project enable/disable --repo` | `{enabled}` |
| `project team-on/team-off --repo` | `{team}` |
| `locale [--set auto/en/uk]` | `{locale, effective, available}`; `locale` is `""` when following the IDE |
| `avatar get` | `{custom, data_url}`: the hiker icon, `data_url` only when custom |
| `avatar set --path ICON.png` | the new icon; PNG, 8–512 px per side, up to 256 KiB |
| `avatar reset` | back to the default hiker |
| `routes` | list of routes |
| `route assets --id` | `{id, images, html}`: the route's pictures as data URLs and HTML snippets as text, keyed by asset path |
| `route import --path [--replace]` | the imported route |
| `route remove --id` | `{removed}` (also forgets that route's achievements) |
| `route template --id --path` | `{path}` of the new pack |
| `journey --scope --route [--repo] [--from-history]` | status |
| `verify --repo` | `{added, updated, removed}` |
| `render [--scope] [--format svg/scene]` | `{svg}` or `{scene}` |

## Scanning

Call `scan` when HEAD moves and on start-up. Pass `--prev-head` with the HEAD
you saw at the previous scan (keep it across restarts). If it is no longer an
ancestor of HEAD (amend, rebase, reset, squash), the core recounts the whole
project and drops commits git no longer has; `rewritten` is then true. Without
`--prev-head`, a scan still reconciles the last 7 days. `verify` recounts
everything on demand.

## Journeys and routes

A journey carries `route`, `distance_m`, `percent`, `finished`, `last_waypoint`,
`next_waypoint`, `to_next_m`, `story`, `achievements`, `commits`,
`streak_days`, `daily` (last 14 local days), `day`, and, when the route has an
elevation profile, `elevation_m`, `ascent_m`, `max_elevation_m` and
`to_next_climb_m` (climb left until the next stop); `underground` is true
while the walker is in a tunnel.

A route carries its texts and `waypoints` (`kind`, `elevation_m`), `biomes`,
`profile` (waypoint heights merged with extra points, sorted), `ascent_m`,
`min_elevation_m`, `max_elevation_m`, `facts` (`{id, at_m, text}`), `objects`
(`{id, at_m, asset, height_px, lift_px, offset_m, fade_m, layer, caption}`,
defaults filled in), `path`, `map_image`, `underground` (`[{from_m, to_m}]`)
and `dangers` (`[{id, at_m, text}]`). Objects and the map image name an
asset; fetch the files once per route with `route assets`.

Event types in `scan`: `waypoint`, `story`, `achievement`, `fact`, `danger`, `finished`.

## Team

`team` returns `{scope, route_id, length_m, members, hidden}`: the project's
journey route (else the global one) and up to 40 people, furthest first. A
member is `{id, name, me, distance_m, percent, finished, commits, today_m,
last_commit_at, elevation_m}`. `id` is an HMAC for colours, never an email.
Identities are merged through `.mailmap`, the user's own addresses become one
`me`, bots are skipped, and the same rules apply as for the user (filters,
per-commit and daily caps, the journey's start date). Nothing is stored: the
result is computed from git history on each call, so cache it in the plugin.

## Panel

Plugins host `ui/panel/panel.html`, replacing `{{CSP}}` and `{{NONCE}}` (use a
nonce-based `script-src`; `style-src 'unsafe-inline'` and `img-src data:` are
needed). They post:

```
{type: "update", state: "ok" | "not_initialized" | "error", error, repo, locale, status,
 avatar, avatar_custom, assets: {routeId: RouteAssets}, team, team_error, locale_setting, dev}
```

`dev` is true in development builds (VS Code launched with F5 or in tests,
JetBrains `runIde`, or `COMMIT_HIKE_DEV=1`): the trail view may then look
ahead of the walker. Release builds only let people look back along the way
they have walked.

```
```

The panel sends `{command}`: `ready`, `setup`, `refresh`, `enableProject`,
`setAvatar`, `resetAvatar`, `chooseRoute` (`scope`), `setLocale` (`locale`),
`setTeam` (`on`), `setDifficulty` (`level`), `requestTeam`, `importRoute`,
`createRouteTemplate`, `verify`.

Compatibility: new fields may appear at any time, so ignore unknown fields. Renaming or removing a field bumps `api`.

## Days off, history and backups

```
commit-hike rest-days [--set sat,sun|none]     -> {"days": [0, 6]}    # 0 = Sunday … 6 = Saturday
commit-hike backup export --path FILE.json     -> BackupResult
commit-hike backup import --path FILE.json [--replace] -> BackupResult
```

`status` also returns:

- `history`: every day with counted commits, all time, oldest first:
  `[{"date": "2026-09-28", "m": 1290.5, "commits": 3}, …]`. The panel draws the
  activity calendar and the year in review from it.
- `rest_days`: the weekdays off. A day off without commits neither breaks nor
  extends a streak; with commits it counts as usual.
- `daily[].commits`: commits on each of the last 14 days.

A backup is one JSON file (`"format": "commit-hike-backup"`, version 1) with
the settings, the counted commits, achievements, the hiker icon, user routes
and the local key. The key is needed because commits and projects are stored
under ids keyed with it: without it, the same commits would count again on
the new computer. That makes the file private; it's written with 0600
permissions. Importing over existing progress fails with `data_exists` unless
`--replace` is given; a file that isn't a valid backup fails with
`invalid_backup`.

## Panel preferences and postcards

The host may add to the data it sends to the panel:

- `reduce_motion`: `true`/`false` overrides the system's reduced-motion setting
  (left out: follow the system). On, every animation and transition is off,
  including animated HTML scenes in route packs.
- `high_contrast`: `true`/`false` overrides the automatic choice (left out:
  on with a VS Code high-contrast theme, `prefers-contrast: more` or forced colors).

The panel sends `{"command": "savePostcard", "name": "commit-hike-<route>-day-<n>.png",
"data": "data:image/png;base64,…"}` when the user saves a postcard. Hosts accept
only PNG data URLs under 20 MB, keep only the file name part of `name`, and
ask where to save.

## Settings

```
commit-hike settings [--reduce-motion auto|on|off] [--high-contrast auto|on|off] [--notifications all|milestones|off]
  -> {"reduce_motion": "auto", "high_contrast": "on", "notifications": "milestones"}
```

Every status carries them as `settings`. The panel follows `reduce_motion` and
`high_contrast` (`auto` = the system or IDE theme); hosts follow `notifications`:
`milestones` drops the per-commit distance and road encounters but keeps stops,
achievements and the finish, `off` shows no pop-ups at all.

The settings page lives in the panel. It sends `setLocale`, `setDifficulty`,
`setRestDays {days: [0..6]}`, `setSettings {reduce_motion?, high_contrast?, notifications?}`,
`setAvatar`, `resetAvatar`, `exportProgress` and `importProgress`. A host opens
it by adding `open_view: "settings"` and a new `open_token` to the data it
sends; the panel opens a page once per token.

## Diagnostics and badges

```
commit-hike diagnostics                -> versions, OS, counts and settings for a bug report
commit-hike badge [--repo PATH] [--lang uk] -> {"svg": "<svg…>", "file_name": "commit-hike-badge.svg", "markdown": "…"}
```

`diagnostics` never includes e-mail addresses, paths, or project and repository
names (a test checks it). Hosts add their own versions and recent errors, with
the home folder and any e-mail address taken out, and copy it all for a GitHub
issue. The panel sends `{"command": "panelError", "message": …}` for its own
errors (at most 20 per session), `copyDiagnostics` for the report and
`saveBadge` for the badge, which shows the journey the panel shows.

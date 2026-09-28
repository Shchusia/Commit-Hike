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

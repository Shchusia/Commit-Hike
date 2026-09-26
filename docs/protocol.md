# Core protocol (api 1)

Plugins run `commit-hike <command> [flags]` and read one JSON document from stdout. Warnings go to stderr and never break stdout.

```json
{ "api": 1, "ok": true, "data": { } }
{ "api": 1, "ok": false, "error": { "code": "not_initialized", "message": "…" } }
```

Error codes: `not_initialized`, `invalid_argument`, `unknown_route`, `not_a_repo`, `busy`, `internal`. Show your own translated text per code; `message` is for logs.

Every command accepts `--lang` (IDE language, e.g. `uk`). A locale fixed in config (`init --locale`) wins.

| Command | Returns |
|---|---|
| `init` | config |
| `scan --repo` | status + `new_commits`, `added_m`, `events` |
| `status [--repo]` | status: `global` and `project` journeys, `today_m`, `total_m`, `locale` |
| `routes` | list of routes |
| `journey --scope --route [--repo] [--from-history]` | status |
| `project enable/disable --repo` | `{enabled}` |
| `verify --repo` | `{added, updated, removed}` |
| `render [--scope] [--format svg/scene]` | `{svg}` or `{scene}` |

Event types in `scan`: `waypoint`, `story`, `achievement`, `finished`.

Compatibility: new fields may appear at any time, so ignore unknown fields. Renaming or removing a field bumps `api`.

# Architecture

## Principles

1. **One engine, thin plugins.** All counting, storage, route logic, achievements and translations of route content live in the Go core. Plugins only detect commits, call the core and render its answer. A new IDE costs a thin adapter, not a rewrite.
2. **Data, not code, for content.** Routes, stories, achievements and translations are JSON files. Adding a route never touches Go code.
3. **Read-only towards repositories, local-only towards the world.** The core only runs `git log`/`git rev-parse`. No network, no hooks, no files in projects.
4. **A versioned contract.** Plugins and core talk through a JSON envelope with an `api` version (see protocol.md).
5. **Core returns data and codes, plugins own UI text.** Error codes, not sentences; route texts come translated from route packs.

## Layers (core/internal)

```
cmd/commit-hike      main: wires the data dir and calls cli.Run
cli               args -> use case -> one JSON envelope (no logic)
app               use cases: Init, Scan, Status, SetJourney, Verify, Render
  ├─ routes       route packs: loading, validation, translated texts
  ├─ achievements rule engine (distance, waypoint, finish, commits, streak, day_distance)
  ├─ render       trail geometry (scene JSON) and SVG
  ├─ i18n         locale chains (uk-UA -> uk -> en) and catalogs
  ├─ score        commit size -> meters, daily soft cap
  ├─ filter       which files count
  ├─ gitlog       read-only git access
  └─ store        local files, locking, atomic writes, HMAC ids
protocol          JSON types shared with plugins (a leaf package)
content           built-in route packs, embedded into the binary
```

Dependencies point downward only. `app` is the only package that knows about all others; `cli` knows only `app` and `protocol`.

## Extension points

| You want to… | Do this |
|---|---|
| add a route | new folder in `core/content/routes` or in the user's `routes` data dir |
| add a language | `locales/<lang>.json` in each route + plugin UI strings |
| add an achievement type | a case in `achievements.Rule.Met` and `validate` |
| support a new IDE | a plugin that calls the CLI and hosts `ui/panel` |
| draw elsewhere (terminal, README badge) | `commit-hike render --format svg` or `--format scene` |
| run as a service later | put a server in front of `app.Service`; no logic changes |

## Data on disk

`~/.config/commit-hike` (Linux), `~/Library/Application Support/commit-hike` (macOS), `%AppData%\commit-hike` (Windows), overridable with `COMMIT_HIKE_HOME`:

- `config.json`: mode, emails, locale, journeys
- `state.json`: counted commits (project id, time, meters) and unlocked achievements
- `key`: random key for HMAC ids
- `routes/`: optional user route packs

# Routes

A route is a folder: one `route.json` plus one translation file per language.
No code changes are needed to add one.

```
my-trail/
  route.json
  locales/en.json      required (or whatever default_locale says)
  locales/uk.json      optional, any number of languages
  README.md            optional
```

## route.json

```json
{
  "id": "my-trail",
  "version": 1,
  "default_locale": "en",
  "length_m": 5000,
  "waypoints": [
    { "id": "start", "at_m": 0, "kind": "hut", "elevation_m": 1290 },
    { "id": "top", "at_m": 2500, "kind": "peak", "elevation_m": 2061 },
    { "id": "lake", "at_m": 5000, "kind": "lake" }
  ],
  "biomes": [
    { "at_m": 0, "type": "forest" },
    { "at_m": 1800, "type": "meadow" },
    { "at_m": 2200, "type": "rock" }
  ],
  "story": [{ "id": "treeline", "at_m": 1500 }],
  "achievements": [
    { "id": "summit", "rule": { "type": "waypoint", "waypoint": "top" } },
    { "id": "secret", "rule": { "type": "commits", "count": 100 }, "hidden": true }
  ]
}
```

| Field | Meaning |
|---|---|
| `id` | lowercase letters, digits and dashes; must match the folder name |
| `length_m` | route length in meters of commits |
| `waypoints[].kind` | map symbol: `start`, `finish`, `peak`, `pass`, `lake`, `river`, `bridge`, `hut`, `village`, `landmark` |
| `waypoints[].elevation_m` | optional, shown on the map and in the places list |
| `biomes` | terrain from `at_m` until the next entry: `forest`, `meadow`, `rock`, `snow`, `water`, `village` |
| `story` | narration shown once the walker passes `at_m` |
| `achievements[].rule.type` | `distance` (`min_m`), `waypoint` (`waypoint`), `finish`, `commits` (`count`), `streak` (`days`), `day_distance` (`min_m`) |
| `path` | optional hand-drawn trail shape, points in 0..1 (the core's SVG renderer uses it) |

Unknown fields are rejected, so a typo fails loudly instead of being ignored.

## locales/&lt;lang&gt;.json

```json
{
  "name": "My Trail",
  "description": "What this route is about.",
  "waypoints": { "top": { "name": "The Top", "text": "Shown when you arrive." } },
  "story": { "treeline": "The trees thin out." },
  "achievements": { "summit": { "name": "Summit", "description": "Reach the top." } }
}
```

Every waypoint, story beat and achievement needs its texts in the default locale. Other languages fall back to it key by key.

## Built-in routes (in this repository)

```bash
task route:new ID=my-trail     # template in core/content/routes/my-trail
task routes:check              # structure + every translation has exactly the same keys
```

## User routes (without rebuilding anything)

In the IDE: **Tools → Commit Hike → Create a Route Template…**, edit the files, then **Import a Route…** (a folder or a `.zip`). The route chooser has the same two links. Imported routes are copied into the `routes` folder next to your Commit Hike data. A broken pack is rejected with a message saying what's wrong; a route with a built-in route's id can't be imported.

From the command line: `commit-hike route template --id my-trail --path DIR`, `commit-hike route import --path DIR_OR_ZIP [--replace]`, `commit-hike route remove --id my-trail`. To check a pack without touching your real data: `task route:try SRC=path`.

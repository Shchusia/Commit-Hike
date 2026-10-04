# Routes

A route is a folder: one `route.json`, one translation file per language and,
optionally, pictures. No code changes are needed to add one.

```
my-trail/
  route.json
  locales/en.json      required (or whatever default_locale says)
  locales/uk.json      optional, any number of languages
  assets/              optional: pictures and static HTML for objects and the map
    signpost.svg
    campfire.png
    fireflies.html
    map.svg
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
    { "id": "start", "at_m": 0, "kind": "start", "elevation_m": 420 },
    { "id": "top", "at_m": 2500, "kind": "peak", "elevation_m": 900 },
    { "id": "lake", "at_m": 5000, "kind": "lake", "elevation_m": 610 }
  ],
  "profile": [
    { "at_m": 1200, "elevation_m": 560 },
    { "at_m": 3800, "elevation_m": 700 }
  ],
  "biomes": [
    { "at_m": 0, "type": "meadow" },
    { "at_m": 1200, "type": "forest" },
    { "at_m": 2200, "type": "rock" },
    { "at_m": 3200, "type": "grove" }
  ],
  "story": [{ "id": "treeline", "at_m": 1500 }],
  "facts": [{ "id": "old-road", "at_m": 800 }],
  "objects": [
    { "id": "sign", "at_m": 300, "asset": "assets/signpost.svg", "height_px": 70, "fade_m": 300 },
    { "id": "lights", "at_m": 2600, "asset": "assets/fireflies.html", "height_px": 100, "lift_px": 20 },
    { "id": "far-tower", "at_m": 4000, "asset": "assets/tower.png", "layer": "far", "fade_m": 1500 }
  ],
  "achievements": [
    { "id": "summit", "rule": { "type": "waypoint", "waypoint": "top" } },
    { "id": "high", "rule": { "type": "altitude", "altitude_m": 850 } },
    { "id": "climber", "rule": { "type": "climb", "min_m": 400 } },
    { "id": "secret", "rule": { "type": "commits", "count": 100 }, "hidden": true }
  ]
}
```

| Field | Meaning |
|---|---|
| `id` | lowercase letters, digits and single dashes (`my-trail`); must match the folder name |
| `length_m` | route length in meters of commits |
| `waypoints[].kind` | map symbol: `start`, `finish`, `peak`, `pass`, `lake`, `river`, `bridge`, `hut`, `village`, `landmark`, `castle`, `tower`, `ruin`, `cave`, `spring`, `viewpoint`, `camp`, `volcano`, `lighthouse`, `harbor` |
| `waypoints[].elevation_m` | height of the stop; also part of the elevation profile |
| `profile` | extra heights between waypoints. Together with waypoint heights this is the elevation profile: the ground in the side view, the profile strip, the terrain of the generated map, the altitude and climb shown to the user |
| `biomes` | terrain from `at_m` until the next entry: `forest` (conifers), `grove` (deciduous woods), `meadow`, `fields`, `steppe`, `desert`, `rock`, `snow`, `tundra`, `water`, `swamp`, `coast` (sea on the horizon), `volcanic`, `village`. Neighbouring biomes blend smoothly: plants thin out, colours and distant land shapes morph |
| `story` | narration shown once the walker passes `at_m` |
| `facts` | short real-world (or in-world) facts; text in `facts.<id>`. They unlock when passed and are collected on the Places tab and as pins on the map |
| `objects` | pictures or static HTML standing at `at_m` (see below) |
| `underground` | stretches walked underground, `[{"from_m": 1284257, "to_m": 1351849}]`: mines, tunnels, caves. The side view shows a mountain face before the mouth, a rock vault and torchlight inside; the map draws a tunnel |
| `dangers` | encounters at `at_m` (a pursuer on the road, an ambush, a storm), text in `dangers.<id>`. The scene turns cold and dim around them; once passed they are marked on the map, the profile and the Places tab |
| `map_image` | optional picture for the map and the minimap instead of the generated terrain, e.g. `assets/map.svg`. Any proportions work: the map takes the picture's shape. Needs `path` |
| `path` | optional trail shape on the map, points `[x, y]` in 0..1 (x to the right, y down). Required with `map_image`, so the trail lies on your drawing |
| `achievements[].rule.type` | `distance` (`min_m`), `waypoint` (`waypoint`), `finish`, `commits` (`count`), `streak` (`days`), `day_distance` (`min_m`), `altitude` (`altitude_m`: stood this high), `climb` (`min_m`: total ascent) |

Unknown fields are rejected, so a typo fails loudly instead of being ignored.

### Objects

| Field | Default | Meaning |
|---|---|---|
| `asset` | | file under `assets/`: `svg`, `png`, `jpg`, `webp`, `gif` or `html` |
| `height_px` | 90 | drawn height in the side view (0–400) |
| `lift_px` | 0 | raise above the ground: birds, signs on a pole, northern lights |
| `offset_m` | 0 | shift the picture along the trail without moving where it fades |
| `fade_m` | 350 | distance from the walker at which the object has faded out completely. It is fully visible within 30% of that, fades in as the walker approaches and out once they've passed |
| `layer` | `trail` | `trail` stands on the path; `far` stands on the distant hills, smaller and hazier, moving slower |

An optional caption goes in `objects.<id>` in the locales; it's shown under the
object when the walker is next to it, and on the Places tab.

**HTML objects are static.** The panel removes scripts, event handlers, forms,
frames and every URL that isn't a `data:image/…`, and shows the rest inside an
isolated shadow root; the panel's Content Security Policy blocks scripts and
network access anyway. CSS (including animations) works, so you can make
glowing lights, waves or an aurora. Give the root element a fixed size; it is
scaled to `height_px`. Respect `prefers-reduced-motion`.

Limits: a pack may contain up to 200 files, 16 MiB in total, 4 MiB per asset.

On long routes the side view shows about 190 px per kilometre, capped at
70,000 px for the whole route (about 24 px per km on a 2,900 km route), so use
larger `fade_m` there (tens of kilometres) or objects flash by in a few pixels.

## locales/&lt;lang&gt;.json

```json
{
  "name": "My Trail",
  "description": "What this route is about.",
  "waypoints": { "top": { "name": "The Top", "text": "Shown when you arrive." } },
  "story": { "treeline": "The trees thin out." },
  "facts": { "old-road": "This path follows a salt road used for 300 years." },
  "dangers": { "wolves": "Wolves howl around the camp at night." },
  "objects": { "sign": "Signpost" },
  "achievements": { "summit": { "name": "Summit", "description": "Reach the top." } }
}
```

Every waypoint, story beat, fact, danger and achievement needs its texts in the default
locale. Object captions are optional. Other languages fall back to the default
key by key.

## Built-in routes (in this repository)

| Route | Length | What it is |
|---|---|---|
| `chornohora-ridge` | 24 km | Ukraine's highest ridge, Zaroslyak – Hoverla – Pip Ivan, with real facts |
| `demo-trail` | 20 km | a short practice route |
| `tongariro-crossing` | 19.4 km | New Zealand's volcanic day hike, with real facts |
| `molfar-path` | 36 km | an original tale on Hutsul folklore; shows PNG and HTML objects |
| `seven-lighthouses` | 70 km | an original fantasy journey from desert to ice; shows almost every biome and a hand-drawn `map_image` |
| `frodo-journey` | 2,863 km | Frodo's road in The Lord of the Rings at real scale: 1,779 miles with mileposts from the Éowyn Challenge (after Karen Wynn Fonstad's atlas), the Black Riders and other encounters at their mileposts, Moria and Shelob's lair underground; the map follows the real road: the Old Forest, south along the Misty Mountains, down the Anduin, the Black Gate, Ithilien, Cirith Ungol and north through the Morgai |
| `bilbo-journey` | ≈2,977 km | Bilbo's road in The Hobbit, there and back again; an estimate from the maps, as no milepost table exists; the map shows the way there (High Pass, Beorn, the Elf-path through Mirkwood, the Forest River, Lake-town) and the different way back, round the north of Mirkwood |
| `alchemist-road` | 4,370 km | Santiago's road in Paulo Coelho's The Alchemist through real places, measured along real coordinates, on real terrain (Natural Earth, public domain) |

Routes use real-world lengths (see "Pace" in architecture.md for how commits
become meters). Heights on the book routes are invented. The book routes retell events in their own words and quote
nothing. Their maps are generated from the route, not copied from the books'
maps (the two Tolkien routes in `extras` carry a simple schematic map of their
own, drawn for this project, so the trail follows the real geography) (the Alchemist map shows real geography). Names like Rivendell, Mordor or
Hobbit are trademarks of Middle-earth Enterprises: if you publish the plugin,
consider shipping those two routes as a separate pack. Each route is one folder
and can be removed before a release.

```bash
task route:new ID=my-trail     # template in core/content/routes/my-trail
task routes:check              # structure + every translation has exactly the same keys
```

## Series

`core/content/series.json` lists routes walked one after another. When a
route of a series is finished, the plugins offer the next one (the status
carries it as `next_route`).

```json
{
  "series": [
    { "id": "carpathians", "routes": ["chornohora-ridge", "svydovets-ridge", "gorgany-popadia-ring"],
      "name": { "en": "The Carpathians", "uk": "Карпати" } }
  ]
}
```

A series needs an id, an English name and at least two built-in routes; a
route can be in one series only. The core checks this when it starts, and
`task routes:check` runs the same check.

## User routes (without rebuilding anything)

In the IDE: **Create a Route Template…** (Tools → Commit Hike in JetBrains IDEs,
the Command Palette or the panel's ⋯ menu in VS Code), edit the files, then
**Import a Route or Map…** (a folder or a `.zip`). The template already contains
an elevation profile, a fact, an SVG object and a climb achievement to copy
from. Imported routes are copied into the `routes` folder next to your Commit
Hike data. A broken pack is rejected with a message saying what's wrong; a
route with a built-in route's id can't be imported.

From the command line: `commit-hike route template --id my-trail --path DIR`,
`commit-hike route import --path DIR_OR_ZIP [--replace]`,
`commit-hike route remove --id my-trail`. To check a pack without touching your
real data: `task route:try SRC=path`.

## Real routes: GPS track

A route through real places can carry its trail as GPS points. The map then
draws it at true scale (north up, with a correct scale bar), instead of a drawn
or generated shape:

```json
{
  "id": "my-ridge",
  "length_m": 24000,
  "track": [[48.164, 24.538], [48.1606, 24.5003], [48.1511, 24.5119], [48.0478, 24.6278]],
  "waypoints": [
    { "id": "start", "at_m": 0, "kind": "hut", "lat": 48.164, "lon": 24.538 },
    { "id": "summit", "at_m": 4000, "kind": "peak", "lat": 48.1606, "lon": 24.5003 },
    { "id": "end", "at_m": 24000, "kind": "peak" }
  ]
}
```

- `track`: `[latitude, longitude]` points in walking order, from a GPX file or
  a map. A point every few hundred meters is plenty; the map smooths between them.
- `lat`/`lon` on a stop pins it exactly to that place. Between pinned stops the
  hiker moves along the track in proportion to the route meters, so `length_m`
  and `at_m` can stay the walking distances (switchbacks make them longer than
  the straight line between points).
- A route has either a `track` or a drawn `path`, not both. Stops may have
  coordinates only on routes with a track.
- Use coordinates you are allowed to use: your own GPS recording, OpenStreetMap
  (ODbL: credit it in the route description) or public sources.

## Drawn maps: pinning stops

A route with its own picture (`map_image`) and a drawn `path` can pin stops to
the picture, like `lat`/`lon` on real tracks. `x` and `y` are fractions of the
picture's width and height (0..1, from the top left):

```json
"path": [[0.65, 0.9], [0.74, 0.86], [0.46, 0.8]],
"waypoints": [
  { "id": "gate", "at_m": 0, "kind": "start", "x": 0.65, "y": 0.9 },
  { "id": "light", "at_m": 60000, "kind": "lighthouse", "x": 0.74, "y": 0.86 }
]
```

Put the path's points on the stops (and a few in between if the trail must go
around something): the map then places every stop exactly where it's drawn,
and the walker moves along the path between them.

## Round trips

`"loop": true` marks a route that ends where it started ("there and back
again"). Without a track or path of its own, the map then draws the trail as a
round trip instead of a one-way line.

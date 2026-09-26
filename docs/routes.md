# Adding a route

```
core/content/routes/my-route/
  route.json
  locales/en.json
  locales/uk.json
```

`route.json`: id (must match the folder), `length_m`, `waypoints` (`id`, `at_m`), optional `story` beats, `achievements` with rules, optional `path` (points in 0..1 for a hand-drawn shape).

Rule types: `distance` (`min_m`), `waypoint` (`waypoint`), `finish`, `commits` (`count`), `streak` (`days`), `day_distance` (`min_m`). Set `"hidden": true` to keep an achievement secret until earned.

Each `locales/<lang>.json` has `name`, `description`, `waypoints.<id>.name/text`, `story.<id>`, `achievements.<id>.name/description`.

Run `task routes:check`: it checks the structure and that every translation has exactly the same keys as English.

Users can also drop route folders into the `routes` directory next to their Commit Hike data; a broken user route is skipped with a warning, and it can't replace a built-in one.

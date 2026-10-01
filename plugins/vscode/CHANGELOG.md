# Changelog

All notable changes to Commit Hike are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/).

Write new entries under **[Unreleased]** as you go. `task version:set V=x.y.z`
turns them into the section of the new version; the JetBrains Marketplace
"What's New" and the VS Code Marketplace changelog tab are built from it.

## [Unreleased]

## [0.3.2] - 2026-10-01

### Fixed
- Team: your own row shows the same distance and commits as your trail. It used to recount them from git, which could give much more: older commits at today's pace, `.mailmap` aliases and a daily limit per repository.

## [0.3.1] - 2026-10-01

### Fixed
- JetBrains: no more use of API the platform plans to remove (`BrowserUtil.browse(File)`) or keeps internal (`PluginManagerCore.getPlugin`)


## [0.3.0] - 2026-10-01

### Added
- Your year on the trail, in the Stats tab: an activity calendar for each year with the distance, commits, active days, best day and longest streak. Switch between years with ‹ ›; hover a day to see its distance and commits.
- Days off: pick weekdays off (Tools → Commit Hike → Days Off…, or the *Days Off…* command in VS Code). A day off without commits doesn't break the streak; one with commits still counts. The Stats tab shows them next to the streak.
- Export and import your progress (settings, counted commits, achievements, hiker icon and your routes) as one file, to move to another computer or keep a backup. Commits are never counted twice after a restore, and importing over existing progress asks first.
- The last-14-days chart also shows how many commits each day had.
- Postcards: the ✉ Postcard button under the hike scene saves a 1200×630 picture of where you are — the real scene with your hiker, the last stop, the route, the day, the distance and your altitude — ready to share.
- Less motion: when the system asks for reduced motion, every animation in the panel is off, including animated scenes in route packs.
- More contrast: with a high-contrast theme in VS Code, or when the system asks for more contrast, the panel uses solid text, visible outlines and stronger focus marks.
- A settings page in the panel, the same in every IDE: language, difficulty, days off, notifications, less motion, more contrast, your hiker and backups. Open it from ⋯ in the panel or with Tools → Commit Hike → Settings… (*Commit Hike: Settings…* in VS Code).
- Notifications: all, only milestones (stops, achievements, the finish) or off. Less motion and more contrast can follow the system or be on or off. These choices are shared by every IDE on the computer and travel with backups.

### Changed
- Tabs take less room, so all five fit in a narrower panel.
- Stats: hover, tap or focus a day in the last-14-days chart to see its date and exact distance. Works for days without commits too, and with the keyboard (Tab, ← →). Before, the values hid in a browser tooltip that JetBrains IDEs never show.

## [0.2.3] - 2026-10-01

### Fixed
- JetBrains IDEs 2026.2 (GoLand, IntelliJ IDEA, PhpStorm, WebStorm, Rider…): the trail window failed to open with `NoClassDefFoundError: JBCefApp`. Since 2026.2 the embedded browser is a separate bundled plugin, and Commit Hike now declares it; older IDEs are unaffected. If "Web Browser (JCEF)" is disabled, the window explains that instead of failing.

## [0.2.2] - 2026-09-29

### Fixed
- `task release:check` refused to release from any branch but `master`: it wanted the README screenshots to point to the current branch, which is deleted after the merge. Now they must point to the default branch (`BRANCH` in `release.env`, `master` by default), and if the checkout knows `origin/master`, each picture must already be there.
- Extras: Frodo's and Bilbo's journeys no longer walk a made-up winding trail on the map. Both carry a schematic map of Middle-earth drawn for this project and a path along the real road, with every stop pinned to its place. Bilbo comes home a different way than he went: round the north of Mirkwood to Beorn's, then over the High Pass. Re-import them with `task routes:install-extras`.

## [0.2.1] - 2026-09-29

### Fixed
- Screenshots on the Open VSX and Visual Studio Marketplace pages: they pointed to a branch that doesn't exist.
- `task release:check` now makes sure every picture on the marketplace page points to this repository and branch and exists.
- JetBrains: the map drew a made-up winding path instead of the real GPS track of Chornohora Ridge and Tongariro. The trail panel now gets the core's answer untouched, so new route data can't get lost on the way again.
- A JetBrains plugin test failed between 00:00 and 06:00 UTC because it expected the journey to start today.
- The Seven Lighthouses: every lighthouse stands on the coast of its map again, the pass sits between the peaks and the oasis among the dunes.

### Added
- Stops can be pinned to a drawn map with `x`/`y`, like `lat`/`lon` on real tracks.
- `"loop": true` for round-trip routes: the map draws them coming back to the start.

## [0.2.0] - 2026-09-28

### Added
- A request to rate Commit Hike, shown only to people who really use it: a week after install, after commits on at least 3 days, right after a commit that moved you along. It opens the right catalog for your editor: JetBrains Marketplace in JetBrains IDEs, Open VSX in VS Code, VSCodium, Cursor and Windsurf. "Rate it" or "Don't ask again" end it for good; "Later" asks again in a week, at most three times.
- `task routes:install-extras` imports the personal routes from `extras/routes` into your own Commit Hike data, so they work with the plugin installed from a marketplace too.

## [0.1.1] - 2026-09-28

### Fixed
- JetBrains: dialogs no longer use `SimpleListCellRenderer.create`, which the platform has scheduled for removal.

## [0.1.0] - 2026-09-28

### Added
- Commit Hike for JetBrains IDEs 2024.3+ and VS Code 1.85+: every commit moves you along a hiking trail.
- Hike view: a side-on scene with parallax mountains, a ground line that follows the route's elevation profile, weather and a sky that follows your clock. You can look back along the way you've walked; what lies ahead stays hidden.
- Map view: a topographic map of the whole trail with zoom and pan. Routes with a GPS track are drawn at real scale; the scale bar, symbols and labels behave like on a paper map at every zoom.
- Places and stats views: stops passed and ahead, daily distance for the last 14 days, streaks, climbing and achievements.
- Built-in routes: Chornohora Ridge and the Tongariro Alpine Crossing on their real tracks, Santiago's road from The Alchemist, The Molfar's Path, The Seven Lighthouses and a demo trail.
- Your own routes: create one from a template, then import it as a folder or a `.zip`. Route packs can bring pictures, small HTML scenes and their own map.
- Walk together: everyone who commits to a project on the same trail, with a leaderboard. Names come from git history and are never stored.
- Difficulty levels, tunnels, encounters, facts to discover and a custom hiker icon.
- English and Ukrainian, including route texts, notifications, dialogs and menus.
- Privacy by design: nothing is written to your repositories and nothing leaves your computer.

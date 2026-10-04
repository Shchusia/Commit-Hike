# Changelog

All notable changes to Commit Hike are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/).

Write new entries under **[Unreleased]** as you go. `task version:set V=x.y.z`
turns them into the section of the new version; the JetBrains Marketplace
"What's New" and the VS Code Marketplace changelog tab are built from it.

## [Unreleased]

## [0.7.0] - 2026-10-04

### Added
- Share where you are: the ↗ Share menu under the hike and on the year postcard posts to X, Bluesky, Mastodon (your own server), Threads, LinkedIn, Facebook, Telegram or Reddit with a ready text in your language. The networks' sharing pages don't take pictures, so the postcard goes to your clipboard first, ready to paste into the post. The menu also saves the postcard, copies it or copies the text. The plugins open only these networks' sharing pages.
- Commit Hike in the terminal: the core installs with one command (`curl … /scripts/install.sh | sh`, or `install.ps1` on Windows) from GitHub Releases, checked against the release's checksums. `commit-hike prompt` prints one line such as "🥾 16,0 км · Озеро Несамовите" for starship, bash, zsh or fish, and counts new commits by itself, so the shell is enough without an IDE. See docs/terminal.md.
- A Neovim plugin (plugins/neovim): the same line in your status line (lualine or plain), updated within a second of any commit, and `:CommitHike` to see where you are, scan, choose a trail, the difficulty or the language. In five languages, with `:checkhealth commit-hike` and `:help commit-hike`.
- Zed: its extension API can't show anything in the status bar or a panel yet, so there is no Zed extension; the starship line works in Zed's terminal and a task shows it on demand (docs/zed.md).
- The core's binaries for Linux, macOS and Windows (x64 and Arm) are published to GitHub Releases for every version tag.
- A team goal: pick a route in the Team tab and the whole team walks it together, everyone's commits since the start adding up, with each person's part, the next stop, the team's pace and the days left. Only the route and the start are saved; the distance is counted from git history, so teammates who pick the same goal see the same numbers. `commit-hike project goal --route ID` / `goal-off` from the command line.
- The team's week in the Team tab: the distance since Monday and how it compares with last week, commits, days with commits, the best day and everyone's part.
- The README in Polish, German and Spanish too, and new screenshots in all five languages, including the Share menu. `task screenshots` makes them from the real panel, so they can be remade after any change.
- The marketplace pages (VS Code / Open VSX and JetBrains) describe all ten routes, sharing, the terminal and the five languages, and link to the README in each language.

### Changed
- The ✉ Postcard buttons became the ↗ Share menu; saving the postcard is its first item.
- The GitHub badge writes the distance like the panel: in metres below 1 km ("640 m", not "0.6 km").

### Fixed
- The GitHub badge used a decimal point in Polish, German and Spanish ("34.8 km"); it uses the comma now, like the panel.
- `task` failed on computers without PyCharm (CI machines included): looking for PyCharm broke every task.
- Editor warnings in the panel (unresolved host colours, fonts without a generic family, unused code) and in the READMEs.

## [0.6.0] - 2026-10-04

### Added
- Polish, German and Spanish: the trail panel, both plugins (menus, dialogs, notifications, the status bar), the command titles in VS Code, and the texts of all ten built-in routes and both series: stops, stories, facts and achievements. Pick a language in Settings → Language, or follow the IDE.
- Distances use a decimal comma in Ukrainian, Polish, German and Spanish, and whole numbers follow each language's rules: 1,925 m · 1 925 м · 1.925 m · 1925 m.

### Fixed
- The four routes added in 0.5.0 (Svydovets, Gorgany, Tour du Mont Blanc, Camino) had their achievement texts mixed up: English descriptions in Ukrainian and Ukrainian names in English. A new check keeps every text in its own script.
- Altitudes in German read like decimals ("1,925 m"): numbers used English grouping in every language but Ukrainian.

## [0.5.0] - 2026-10-04

### Added
- Seasons follow the calendar, flipped south of the equator: snow covers the ground and the grass in winter (not deserts or beaches), meadows and fields turn golden in autumn, blossom comes in spring.
- A camp on days without commits: a tent and a fire next to your hiker, warmer at night. On a day off it's a rest, not a stop.
- Four new routes on real geography: the Svydovets ridge (34 km) and the Popadia ring in the Gorgany (41 km) in the Ukrainian Carpathians, the Tour du Mont Blanc through France, Italy and Switzerland (170 km), and the Camino Francés to Santiago de Compostela (772 km). Each has its own stops, story, facts and achievements, in English and Ukrainian.
- Route series: finish a route and the plugin offers the next one, in the panel and in the notification. The Carpathians: Chornohora → Svydovets → Gorgany. The long ways: Tour du Mont Blanc → Camino Francés.
- The hiker's passport (Places → Passport): a dated stamp for every stop you reach, on every trail, including the ones you've left for another. Trails you switched away from before this version aren't in it.

### Changed
- JetBrains plugin: the logic behind the settings, backups, routes, the hiker, the language, the team and the notifications moved out of the IDE glue into tested code; the plugin's coverage counts that logic, not the Swing dialogs and menu actions.

### Fixed
- JetBrains: a message from the panel with a field of the wrong type was dropped silently instead of throwing in the browser callback.
- The ascent lines between stops on the Places tab picked up the settings page's button layout.

## [0.4.0] - 2026-10-02

### Added
- Browser tests for the trail panel (Playwright), run by `task check` and in CI: tabs, map and zoom, the charts, the year, settings, postcards, motion and contrast. `task check` ends with a coverage table for the core, the panel, the VS Code extension and the JetBrains plugin.
- "Copy a report for the developer" in the settings: versions, your IDE and recent errors (the panel's too), without your name, e-mail, projects or code, ready to paste into a GitHub issue.
- A badge for your GitHub profile: Stats → *Badge for your GitHub profile…* saves an SVG like "🥾 Commit Hike | 342 km · Chornohora Ridge" and copies the Markdown to show it.
- A year postcard: the year's totals and the whole activity calendar on one 1200×630 picture, from the year on the Stats tab.

### Changed
- The panel tests use a Chrome or Chromium already on the computer when there is one, and download Playwright's at most once, never waiting for it forever.
- The IntelliJ Platform Gradle plugin is 2.11.0 (from 2.5.0), the newest that runs on Gradle 8; 2.12 and later need Gradle 9.

### Fixed
- "No commits in the last two weeks yet" showed only when there was no history at all; it now shows after two quiet weeks.
- The map no longer logs an error when you switch tabs quickly.

## [0.3.2] - 2026-10-01

### Fixed
- Team: your own row shows the same distance and commits as your trail. It used to recount them from git, which could give much more: older commits at today's pace, `.mailmap` aliases and a daily limit per repository.

## [0.3.1] - 2026-10-01

### Fixed
- JetBrains: no more use of API the platform plans to remove (`BrowserUtil.browse(File)`) or keeps internal (`PluginManagerCore.getPlugin`).

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

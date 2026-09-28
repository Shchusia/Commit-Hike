# Changelog

All notable changes to Commit Hike are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/).

Write new entries under **[Unreleased]** as you go. `task version:set V=x.y.z`
turns them into the section of the new version; the JetBrains Marketplace
"What's New" and the VS Code Marketplace changelog tab are built from it.

## [Unreleased]

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


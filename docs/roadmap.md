# Roadmap

Each stage is one release. Done items are ticked; see CHANGELOG.md for what shipped.

## 0.4.0 — foundation and quick wins
- [x] Browser tests for the panel (Playwright), in `task check`
- [x] "Copy a report for the developer" (version, IDE, OS, recent errors; nothing personal)
- [x] A badge for your GitHub profile (SVG)
- [x] A year-in-review postcard
- [x] Update the IntelliJ Platform Gradle plugin (2.11.0, the newest for Gradle 8)
- [ ] Move to Gradle 9 (with the Kotlin, ktlint and Kover Gradle plugins), then the IntelliJ Platform Gradle plugin 2.12+
- [ ] Decide on the minimum IDE version (2024.3 vs 2025.1) from the Marketplace download stats

## 0.5.0 — atmosphere, routes and series (0.5.0 and 0.6.0 in one release)
- [x] JetBrains plugin coverage: logic out of the IDE glue (64% → ~70%)
- [x] Seasons by the calendar (snow in winter, autumn colours), respecting each route's biomes
- [x] A camp on days without commits, a rest on days off
- [x] The hiker's passport: a dated stamp for every stop passed, across all routes
- [x] New GPS routes: Svydovets, Gorgany, Tour du Mont Blanc, Camino de Santiago
- [x] Route series: when you finish, carry on with the next route ("The Carpathians", "The long ways")

## 0.6.0 — languages
- [x] Polish, German and Spanish: the panel, both plugins and the core
- [x] Polish, German and Spanish: the texts of all ten built-in routes and both series
- [ ] Proofreading by native speakers

## 0.7.0 — sharing, beyond IDEs, team
- [x] Share on social networks (X, Bluesky, Mastodon, Threads, LinkedIn, Facebook, Telegram, Reddit)
- [x] Core binaries on GitHub Releases, a one-command install
- [x] A starship prompt module (any shell prompt) and a Neovim plugin
- [x] Look into Zed's extension API: no status bar or panels yet, so the terminal for now (docs/zed.md)
- [x] READMEs in five languages, new screenshots for both plugins and the marketplace pages
- [x] A team goal walked together, and the team's week

## 1.0.0 — routes from the site
- [x] [commit-hike.dev](https://commit-hike.dev): share, find, rate and translate routes; sign-in with Google and GitHub; moderation
- [x] A route editor on the site: draw on a map or a picture, load a GPX track, place stops and texts, check and publish
- [x] Find routes from the site in every plugin and the terminal: install in one click, update to the author's new versions

## Later
- [ ] Publish a route from the plugin to the site (a key for the plugin on the site)
- [ ] Notifications for authors: ratings, comments, translations waiting for approval

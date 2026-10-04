# Commit Hike

[![JetBrains Plugin](https://img.shields.io/jetbrains/plugin/v/34595?label=JetBrains&logo=jetbrains)](https://plugins.jetbrains.com/plugin/34595)
[![JetBrains Downloads](https://img.shields.io/jetbrains/plugin/d/34595?label=downloads)](https://plugins.jetbrains.com/plugin/34595)
[![Open VSX](https://img.shields.io/open-vsx/v/shchusia/commit-hike?label=Open%20VSX&logo=eclipseide)](https://open-vsx.org/extension/shchusia/commit-hike)
[![Open VSX Downloads](https://img.shields.io/open-vsx/dt/shchusia/commit-hike?label=downloads)](https://open-vsx.org/extension/shchusia/commit-hike)
[![GitHub stars](https://img.shields.io/github/stars/Shchusia/Commit-Hike?style=flat&logo=github)](https://github.com/Shchusia/Commit-Hike)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

*[English](README.md) · [Українська](docs/README.uk.md) · [Polski](docs/README.pl.md) · [Deutsch](docs/README.de.md) · [Español](docs/README.es.md) · [What's new](CHANGELOG.md)*

Every commit you make moves you along a hiking trail, right inside your IDE.
A day of steady work takes you about ten kilometers along a real mountain
ridge, a desert crossing or a fairy tale, with a scene that changes as you go,
a map, stories, facts and achievements.

<p>
  <img src="https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/docs/screenshots/en-hike.png" width="49%" alt="Hike view: a side-on scene of the trail">
  <img src="https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/docs/screenshots/en-map.png" width="49%" alt="Map view: the real GPS track of the Chornohora ridge">
</p>
<p>
  <img src="https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/docs/screenshots/en-places.png" width="49%" alt="Places view: every stop and its story">
  <img src="https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/docs/screenshots/en-stats.png" width="49%" alt="Stats view: streaks, daily distance and achievements">
</p>

## Features

- **Real, literary and original routes.** Walk the Chornohora ridge over Hoverla and New Zealand's Tongariro Alpine Crossing on their real GPS tracks, Santiago's journey from The Alchemist across real Spain, Morocco, the Sahara and Egypt on real terrain, a folk tale through the Carpathians or a fantasy journey past seven lighthouses. Each route has its own stops, story, facts and achievements.
- **Five languages.** English, Ukrainian, Polish, German and Spanish, following your IDE or chosen in the settings.
- **Series.** Finish a route and walk on to the next: the Carpathians (Chornohora → Svydovets → Gorgany) and the long ways (Tour du Mont Blanc → Camino Francés to Santiago).
- **Seasons, camps and a passport.** Snow in winter, golden meadows in autumn, a camp by the fire on days without commits, and a passport with a dated stamp for every stop you've reached.
- **Climb for real.** Routes have elevation profiles: the ground rises and falls, your hiker leans into the slope, and you see your altitude, the grade, how much you've climbed and the climb left to the next stop. Some achievements are about height.
- **A living scene.** The sky follows your clock and the weather changes day by day. The land changes with the route: conifer forests, deciduous woods that turn gold in autumn, meadows, fields, steppe, desert, rock, snow, tundra, lakes, swamps, the seaside, volcanoes and villages, blending smoothly into each other.
- **Things to find.** Route packs place pictures (SVG, PNG, JPG, WebP, GIF) or small animated HTML scenes along the trail. They fade in as you approach and out once you've passed. Facts about the places unlock as you walk.
- **A minimap and a real map.** The hike view has a minimap of the whole trail. The map tab is a topographic map generated from the route itself: contour lines, hill shading, forests, lakes and the sea, with the stops you've passed, facts you've found and your teammates. Real routes are drawn from their GPS track at true scale. Zoom and pan like a paper map: the scale bar follows the zoom, symbols and labels keep their size, and more labels appear as you zoom in. A route pack can bring its own hand-drawn map instead.
- **Walk together.** Turn on teammates for a project and everyone who commits to it appears on the same trail, with a leaderboard, the team's week and a team goal: a route you all walk together, everyone's commits adding up. Names come from git history and are never stored.
- **Real distances, your difficulty.** Routes have their real lengths: Santiago's road is about 4,370 km. A typical day of commits takes you about 10 km on medium (12.5 on easy, 8 on hard), about half a hiker's day, so a long route is a months-long goal. A difficulty change applies from then on, and you see how many typical days are left.
- **Tunnels and encounters.** Walk through mines and caves by torchlight, and feel the world go cold and dark where danger crossed the road.
- **No spoilers.** You can look back along the way you've walked, but what lies ahead stays hidden until you get there.
- **Your year on the trail.** An activity calendar for every year you've walked, with the distance, commits, active days, best day and longest streak. Hover any day, or any bar of the last two weeks, to see its distance and commits.
- **Days off.** Choose weekdays off (say, Saturday and Sunday): a day off without commits doesn't break your streak, and one with commits still counts.
- **A badge for your profile.** Save an SVG badge with your trail and distance for your GitHub profile README, and a postcard of your year.
- **Share it.** A postcard of where you are — the real scene, your hiker, the route, day and distance — saved, copied or posted to X, Bluesky, Mastodon, Threads, LinkedIn, Facebook, Telegram or Reddit with the text ready.
- **In the terminal too.** One line in your shell prompt (starship, bash, zsh, fish) and in Neovim's status line, from the same core: `🥾 16.0 km · Lake Nesamovyte`.
- **One settings page.** Language, difficulty, days off, notifications, motion, contrast, your hiker and backups in one place, the same in every IDE.
- **Comfortable for everyone.** Reduced motion and high-contrast themes are respected, down to animated scenes in route packs.
- **Take it with you.** Export your progress, settings, hiker and routes to one file and import it on another computer: nothing is counted twice.
- **Fair distance.** Meters grow with the size of a commit, but slowly: tiny commits count, huge ones can't be farmed. Lock files, generated code, whitespace-only changes and other people's commits don't count. Squashed, rebased or amended history is recounted, never counted twice.
- **Your own routes and maps.** Create a route from a template, add pictures and a map, then import it as a folder or a `.zip`, right from the IDE. See [docs/routes.md](docs/routes.md).
- **Your own hiker.** Replace the default figure with any PNG with a transparent background. The default, [`ui/panel/hiker-default.png`](ui/panel/hiker-default.png), shows the expected format: facing right, feet at the bottom edge.
- **Private by design.** Nothing is written to your repositories and nothing leaves your computer. No source code, commit messages, file names or repository names are stored.

## Supported IDEs

| IDE | Where to get it |
|---|---|
| JetBrains IDEs 2024.3+: PyCharm, IntelliJ IDEA, GoLand, WebStorm, PhpStorm, RubyMine, CLion, Rider… | [JetBrains Marketplace](https://plugins.jetbrains.com/plugin/34595) |
| VSCodium, Cursor, Windsurf and other editors that use Open VSX | [Open VSX](https://open-vsx.org/extension/shchusia/commit-hike) |
| VS Code 1.85+ | coming to the Visual Studio Marketplace; until then, install the `.vsix` from Open VSX |
| Neovim 0.10+, and any shell prompt: starship, bash, zsh, fish | [commit-hike.nvim](https://github.com/Shchusia/commit-hike.nvim), [docs/terminal.md](docs/terminal.md) |
| Zed | through its terminal, see [docs/terminal.md](docs/terminal.md#zed) |

## Install

**JetBrains IDEs.** Open **Settings → Plugins → Marketplace**, search for
*Commit Hike* and click **Install**. Or open the
[plugin page](https://plugins.jetbrains.com/plugin/34595) and click
**Install to IDE**.

**VSCodium, Cursor, Windsurf.** Open the Extensions view, search for
*Commit Hike* and click **Install**. Or use the
[Open VSX page](https://open-vsx.org/extension/shchusia/commit-hike).

**VS Code.** Until Commit Hike is on the Visual Studio Marketplace, download
the `.vsix` from the [Open VSX page](https://open-vsx.org/extension/shchusia/commit-hike)
(**Download**), then in VS Code open the Extensions view, **⋯ → Install from VSIX…**
and pick the file.

**Terminal and Neovim.** Install the core with one command, then add it to your
prompt or your status line, see [docs/terminal.md](docs/terminal.md):

```sh
curl -fsSL https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/scripts/install.sh | sh
```

In Neovim 0.10+, with lazy.nvim, then add `require("commit-hike").statusline` to your status line:

```lua
{ "Shchusia/commit-hike.nvim", version = "*", opts = {} }
```

**From source.** See [Development](#development): `task jetbrains:install` or
`task vscode:install`.

## Using it

Open the **Commit Hike** tool window (JetBrains) or the Commit Hike view in the
activity bar (VS Code). The first time, a short setup asks which projects count
and where your journey starts. Then just commit.

| Tab | What you see |
|---|---|
| **Hike** | The side-on scene with your hiker, today's distance, the day of the journey and the story of the place. Scroll back along the way you've walked with the mouse wheel, by dragging or with ← →. |
| **Map** | The whole trail on a topographic map. Zoom with the wheel or +/−, drag to move, ⌖ jumps to you. |
| **Places** | Every stop: passed ones with their stories, the next ones with the distance left. |
| **Team** | Everyone committing to this project on the same trail, when teammates are on. |
| **Stats** | Today, streaks, the last 14 days, climbing, achievements and the version you run. |

Everything else is in **Tools → Commit Hike** (JetBrains) or the command
palette, under *Commit Hike* (VS Code): choosing a trail, importing your own
route, the hiker icon, difficulty, language, counting a project or not, and
recounting a project from its git history.

**↗ Share** under the hike (and on the year postcard in Stats) saves or copies a
postcard of where you are, or opens a post on X, Bluesky, Mastodon, Threads,
LinkedIn, Facebook, Telegram or Reddit with the text ready; the postcard is in
your clipboard to paste into it.

<p><img src="https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/docs/screenshots/en-share.png" width="49%" alt="The Share menu: save, copy or post the postcard"></p>

Your progress is stored in `~/.config/commit-hike` (Linux), `~/Library/Application
Support/commit-hike` (macOS) or `%AppData%\commit-hike` (Windows). `task data:where`
shows it.

## Repository layout

```
core/                Go engine: counting, storage, routes, achievements, i18n, rendering
  content/routes/    built-in route packs (route.json + locales/*.json + assets/)
  tools/uncovered/   prints what the tests don't cover
ui/panel/            the trail view shared by all IDE plugins
plugins/vscode/      VS Code extension (TypeScript)
plugins/jetbrains/   JetBrains plugin (Kotlin)
extras/routes/       route packs that are never shipped (personal use only)
scripts/             release helpers: configure, version, checks, notes
docs/                architecture, protocol, routes, publishing
```

## Development

You need Go, Node.js 22, JDK 21 and [Task](https://taskfile.dev)
(`sudo snap install task --classic`).

```bash
task setup               # check tools, install linters, download dependencies
task                     # list every command
task check               # everything CI runs: linters, vulnerabilities, tests, coverage gate
task test                # all tests: core, routes, VS Code, JetBrains
task lint                # golangci-lint, ESLint + tsc, ktlint
task fmt                 # format Go and Kotlin

task core:demo REPO=~/projects/my-app    # try the core on a real repository (sandbox data)
task core:run -- status --lang uk         # call any core command
task route:new ID=my-trail                # start a new built-in route
task route:try SRC=path/to/route          # check a route pack in the sandbox
task routes:install-extras                # add the personal routes from extras/ to your own plugin

task deps:outdated       # what can be updated
task deps:update         # update dependencies and re-run all checks
```

### Development and production builds

| | prod (default) | dev |
|---|---|---|
| Who gets it | the marketplaces, `task jetbrains:install` | you, while developing |
| Hike view | looks only back from where you are: no spoilers | can look ahead along the whole trail |
| How to build | `task jetbrains:install`, `task vscode:install` | `task dev:jetbrains`, `task dev:vscode`, or `FLAVOR=dev` on any build task |

`task jetbrains:run` / `task jetbrains:run-pycharm` (a separate test IDE) and
F5 in VS Code always behave like dev builds. The Stats tab shows the version
and says when a build is a development build.

### Tests and coverage

```bash
task coverage            # all three languages
task core:cover          # Go: per package, every uncovered line with its function, HTML report; fails below 80%
task core:cover MIN=85   # a different threshold
task ui:test             # browser tests for the panel (Playwright; uses your Chrome if Playwright's isn't there)
task vscode:cover        # TypeScript, with uncovered lines (trust branch and function %; imports count as lines)
task jetbrains:cover     # Kotlin via Kover, per class, plus an HTML report
```

Reports land in `.sandbox/cover/` and `plugins/jetbrains/build/reports/kover/`.

## Releasing

One-time setup, then one command per release. The full walkthrough, including
accounts, tokens and signing, is in [docs/publishing.md](docs/publishing.md).

```bash
task release:configure OWNER=your-github-name NAME="Your Name" EMAIL=you@example.com   # once
task version:set V=0.8.0     # notes under [Unreleased] in CHANGELOG.md become 0.8.0
task release:check           # ready? also lists new screenshots that still have to be pushed
task release:build           # exactly what the marketplaces get, after every check
git commit -am "Release 0.8.0" && git tag v0.8.0 && git push origin master --follow-tags
task ovsx:publish            # the marketplaces last: their pages show the images from master
```

Pushing the tag publishes the core to GitHub Releases and the Neovim plugin to
[commit-hike.nvim](https://github.com/Shchusia/commit-hike.nvim) (CI). The
signed JetBrains zip is uploaded by hand.

## Adding a route or a translation

A route is a folder with a `route.json`, one file per language and optional
pictures. No code changes are needed; see [docs/routes.md](docs/routes.md).
Real routes can carry their GPS track, so the map shows them at true scale.
Tests fail if any translation is incomplete, so a missing string never reaches
users.

Routes based on books, films or games need the rights holder's permission.
The built-in fantasy routes are original works; routes based on other people's
worlds stay in `extras/` and are never published.

## How it works

IDE plugins are thin: they notice commits and draw the result. All logic lives
in one small Go binary that plugins call and that answers in versioned JSON.
Details: [docs/architecture.md](docs/architecture.md),
[docs/protocol.md](docs/protocol.md).

## Support the project

Commit Hike is free and open source. If you enjoy the walk:

- ⭐ star the repository: it helps other developers find it;
- rate it on [JetBrains Marketplace](https://plugins.jetbrains.com/plugin/34595) or [Open VSX](https://open-vsx.org/extension/shchusia/commit-hike/reviews);
- report bugs and share ideas in [issues](https://github.com/Shchusia/Commit-Hike/issues);
- make a route and share it: see [docs/routes.md](docs/routes.md).

## License

[MIT](LICENSE)

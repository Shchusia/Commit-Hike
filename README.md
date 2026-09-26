# Commit Hike

*[Українською](README.uk.md)*

Every commit you make moves you along a hiking trail, right inside your IDE.

## Features

- **Real routes.** Walk the Chornohora ridge from Zaroslyak over Hoverla to Pip Ivan, or a short demo trail. Each route has its own waypoints, story and achievements.
- **Fair distance.** Meters grow with the size of a commit, but slowly: tiny commits count, huge ones can't be farmed. Lock files, generated code, whitespace-only changes and other people's commits don't count.
- **Your language.** Route texts are translated; the core picks the IDE's language and falls back to English.
- **Private by design.** Nothing is written to your repositories and nothing leaves your computer. No source code, commit messages, file names or repository names are stored.

## Supported IDEs

| IDE | Status |
|---|---|
| JetBrains IDEs 2024.3+ (PyCharm, IntelliJ IDEA, GoLand, WebStorm…) | in development |
| VS Code 1.85+ | in development |

## Repository layout

```
core/            Go engine: counting, storage, routes, achievements, i18n, rendering
  content/routes/  built-in route packs (route.json + locales/*.json)
ui/panel/        trail view shared by all IDE plugins
plugins/vscode/  VS Code extension (TypeScript)
plugins/jetbrains/  JetBrains plugin (Kotlin)
docs/            architecture, protocol, how to add a route
```

## Development

You need Go, Node.js 22, JDK 21 and [Task](https://taskfile.dev) (`sudo snap install task --classic`).

```bash
task setup               # check tools, install linters, download dependencies
task                     # list every command
task test                # all tests: core, routes, VS Code, JetBrains
task lint                # golangci-lint, ESLint + tsc, ktlint
task check               # everything CI runs; use before pushing

task core:demo REPO=~/projects/my-app    # try the core on a real repository (sandbox data)
task core:run -- status --lang uk         # call any core command

task jetbrains:install   # build and install the plugin into your PyCharm, then restart it
task jetbrains:run-pycharm   # or run PyCharm with the plugin in a separate profile
task vscode:install      # build and install the VS Code extension

task deps:outdated       # what can be updated
task deps:update         # update dependencies and re-run all checks
```

## Adding a route or a translation

A route is a folder with a `route.json` and one file per language. No code changes are needed; see [docs/routes.md](docs/routes.md). Tests fail if any translation is incomplete, so a missing string never reaches users.

## How it works

IDE plugins are thin: they notice commits and draw the result. All logic lives in one small Go binary that plugins call and that answers in versioned JSON. Details: [docs/architecture.md](docs/architecture.md), [docs/protocol.md](docs/protocol.md).

## License

MIT

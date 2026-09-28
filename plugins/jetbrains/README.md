# Commit Hike for JetBrains IDEs

Works in PyCharm, IntelliJ IDEA, GoLand, WebStorm and other JetBrains IDEs 2024.3+.
Only the bundled Git plugin is required (enabled by default).

Run everything from the repository root with [Task](https://taskfile.dev):

```bash
task jetbrains:test          # tests against the real core
task jetbrains:lint          # ktlint (style from ../../.editorconfig)
task jetbrains:build         # plugin zip in build/distributions/
task jetbrains:install       # install into your newest PyCharm, then restart it
task jetbrains:run-pycharm   # run your PyCharm with the plugin in a separate profile
task jetbrains:run           # run a fresh IntelliJ IDEA Community with the plugin
```

The first build downloads Gradle and IntelliJ IDEA Community 2024.3.5 (about 1 GB) to compile against.

Manual install: Settings → Plugins → ⚙ → Install Plugin from Disk → pick the zip.

## Layout

- `core/`: talks to the core binary (protocol v1). No IntelliJ APIs, so it's unit-tested with plain JUnit.
- `CommitHikeApp.kt`: app-level service, extracts the bundled core binary, serializes calls.
- `ProjectTrek.kt`: per-project service, detects commits through Git4Idea, runs scans, notifications.
- `ui/`: status bar widget, tool window (hosts `ui/panel/panel.html` in JCEF), dialogs.
- `actions/`: Tools → Commit Hike menu.

## Troubleshooting

Help → Show Log in Files, search `idea.log` for `Commit Hike`.
`COMMIT_HIKE_BINARY=/path/to/commit-hike` makes the plugin use your own core build.

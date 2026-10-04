# Commit Hike for Neovim

Your commits walk a hiking trail, in your status line:

```
 main  init.lua  🥾 16,0 км · Озеро Несамовите   utf-8  lua
```

A thin wrapper around `commit-hike`, the same core as the JetBrains and VS
Code plugins (they share your progress). Install the core and the plugin as
described in [docs/terminal.md](../../docs/terminal.md#neovim); `:help
commit-hike` has the rest.

Tests (against the real core, in a headless Neovim): `task neovim:test`.

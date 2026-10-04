# Zed: what its extension API allows (October 2026)

Decision: **no Zed extension for now.** People in Zed get Commit Hike through
the terminal (starship) and a task, see [terminal.md](terminal.md#zed).
Revisit when extensions can add status bar items.

## What extensions can do

Zed extensions are Rust compiled to WebAssembly, using the
[`zed_extension_api`](https://docs.rs/zed_extension_api) crate. They can add:

- languages (grammars, language servers), themes and icon themes, snippets;
- debug adapters;
- slash commands for the AI assistant, and MCP context servers;
- processes they run (with permission) and HTTP downloads, e.g. to install a
  language server.

Everything is "headless": there is no API for status bar items, panels,
notifications or custom views. Zed's own extension FAQ says UI customization
is planned for the future, and an
[RFC for a visual extension API](https://github.com/zed-industries/zed/discussions/53403)
(April 2026) proposes status bar items as its first phase. It isn't accepted
or shipped yet.

## What a Commit Hike extension could do today, and why we don't

| Option | What the user would get | Verdict |
|---|---|---|
| Slash command `/hike` in the assistant | the progress line pasted into an AI chat | odd place for it; nothing visible while coding |
| MCP context server | the AI could read your progress | no value for the walker |
| A fake "language server" that reports progress as diagnostics or hovers | warnings in the code that aren't warnings | misuse of the editor; confusing |
| Status bar item | the same line as in VS Code and JetBrains | **not possible yet** |

Meanwhile, Zed's terminal runs your shell, so the starship module shows the
line there, and a Zed task (`tasks.json`) shows it on demand from the command
palette. Both need nothing from us beyond the core.

## When to come back

When the extension API gets status bar items (phase 1 of the RFC above): the
extension would run `commit-hike prompt --scan` on save and on focus, like the
Neovim plugin, and show its line. A panel with the trail scene would need
rendering (GPUI), not a web view, so the shared HTML panel couldn't be reused.

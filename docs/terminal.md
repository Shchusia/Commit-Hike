# Commit Hike outside the IDE: shell prompt, Neovim, Zed

The IDE plugins bundle the core. Everywhere else you install it once:
`commit-hike` is a single program with no dependencies.

## Install the core

macOS and Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/scripts/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/scripts/install.ps1 | iex
```

Both download the archive for your system from
[GitHub Releases](https://github.com/Shchusia/Commit-Hike/releases), check it
against the release's `checksums.txt` and put `commit-hike` into
`~/.local/bin` (Windows: `%LOCALAPPDATA%\Programs\commit-hike`, added to your
PATH). A specific version: `... | sh -s -- --version 0.7.0`, or
`$env:COMMIT_HIKE_VERSION = "0.7.0"` before the PowerShell line.

Prefer to do it by hand? Download `commit-hike_<os>_<arch>.tar.gz` (`.zip` on
Windows) from the release page, check it with `sha256sum -c checksums.txt
--ignore-missing` and put `commit-hike` anywhere on your PATH. From source:
`cd core && go build ./cmd/commit-hike`.

## Set it up

If you already use Commit Hike in an IDE, there is nothing to do: the
terminal and the IDE share your progress (the same data folder).

Otherwise, once:

```sh
commit-hike init --email "$(git config --global user.email)"
```

It counts your existing history too (`--from-history=false` to start from
today); pick a trail with `commit-hike journey --scope global --route
chornohora-ridge` (`commit-hike routes` lists them) and a difficulty with
`commit-hike difficulty --set easy|medium|hard`.

## The line: `commit-hike prompt`

```console
$ commit-hike prompt --scan
🥾 16,0 км · Озеро Несамовите
```

- The journey the IDE panel would show: this project's own trail, else your
  main one. Outside a repository it shows your main journey.
- `--scan` counts new commits first, but only when HEAD moved since the last
  prompt: about 20 ms when nothing changed, 30 ms after a commit. That's
  what makes the shell enough without an IDE.
- The language comes from Commit Hike's setting, else `LC_ALL`, `LC_MESSAGES`
  or `LANG`; `--lang uk` to force one. `--icon ""` drops the 🥾.
- It prints plain text (no JSON), and never fails a prompt: when something is
  wrong it prints nothing and exits 0.

## starship

Add to `~/.config/starship.toml`:

```toml
[custom.commit_hike]
command = "commit-hike prompt --scan"
require_repo = true      # only inside git repositories; remove to show it everywhere
shell = ["sh"]
format = "[$output]($style) "
style = "bold yellow"
```

and put `${custom.commit_hike}` into your `format` if you have a custom one
(with the default format, custom modules show up by themselves).

## Any other prompt

bash (`~/.bashrc`):

```bash
PS1='$(commit-hike prompt --scan) '"$PS1"
```

zsh (`~/.zshrc`), on the right:

```zsh
setopt prompt_subst
RPROMPT='$(commit-hike prompt --scan)'
```

fish (`~/.config/fish/functions/fish_right_prompt.fish`):

```fish
function fish_right_prompt
    commit-hike prompt --scan
end
```

## Neovim

The plugin needs Neovim 0.10+ and the core above. It's published with every
release as [`Shchusia/commit-hike.nvim`](https://github.com/Shchusia/commit-hike.nvim),
a copy of `plugins/neovim`. With [lazy.nvim](https://github.com/folke/lazy.nvim):

```lua
{ "Shchusia/commit-hike.nvim", version = "*", opts = {} }
```

`version = "*"` keeps you on released versions. vim-plug:
`Plug 'Shchusia/commit-hike.nvim', { 'tag': '*' }` and
`lua require("commit-hike").setup()` after `plug#end()`.

Straight from this repository, e.g. to try `master`:

```lua
{
  "Shchusia/Commit-Hike",
  config = function(plugin)
    vim.opt.rtp:append(plugin.dir .. "/plugins/neovim")
    require("commit-hike").setup()
  end,
}
```

Show it in your status line, e.g. with lualine:

```lua
require("lualine").setup({
  sections = { lualine_x = { require("commit-hike").statusline, "encoding", "filetype" } },
})
```

or without plugins: `set statusline+=%{v:lua.require'commit-hike'.statusline()}`.

`:CommitHike` shows where you are, `:CommitHike scan`, `:CommitHike route`,
`:CommitHike difficulty`, `:CommitHike lang` and `:CommitHike init` do what
they say; `:help commit-hike` has the details and `:checkhealth commit-hike`
checks the setup. Commits made anywhere (the terminal, fugitive, lazygit)
reach the status line within a second.

## Zed

Zed's extension API doesn't let extensions show anything in the status bar
or in a panel yet (see [zed.md](zed.md)), so there is no Zed extension. Two
things work today:

- Zed's terminal uses your shell, so the starship line above shows up there.
- A task to check progress from the command palette, in `~/.config/zed/tasks.json`:

  ```json
  [
    {
      "label": "Commit Hike: where am I",
      "command": "commit-hike prompt --scan",
      "reveal": "always",
      "hide": "on_success",
      "use_new_terminal": false
    }
  ]
  ```

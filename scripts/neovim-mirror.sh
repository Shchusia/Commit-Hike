#!/bin/sh
# Publishes plugins/neovim as a repository of its own, e.g. Shchusia/commit-hike.nvim:
# one commit per release on its main branch, plus the version tag.
#   scripts/neovim-mirror.sh URL TAG
# URL: where to push (https://github.com/OWNER/commit-hike.nvim.git, or with a token in CI).
# The commit holds exactly the plugins/neovim tree of HEAD, nothing else (no history
# rewriting: git subtree split gets merges wrong), on top of the mirror's last good commit.
set -eu
url="$1"; tag="$2"

tree=$(git rev-parse "HEAD:plugins/neovim")
for need in lua plugin doc; do
  git ls-tree --name-only "$tree" | grep -qx "$need" || {
    echo "plugins/neovim at HEAD has no $need/: is everything committed?" >&2; exit 1; }
done

parent=""
if git fetch --quiet "$url" main 2>/dev/null; then
  if git ls-tree --name-only FETCH_HEAD | grep -qx lua; then
    parent=$(git rev-parse FETCH_HEAD)
  else
    echo "The mirror's main isn't a Neovim plugin (lua/ missing): replacing it."
  fi
fi

if [ -n "$parent" ] && [ "$(git rev-parse "$parent^{tree}")" = "$tree" ]; then
  commit="$parent" # nothing changed in the plugin: just tag the same commit
else
  msg="commit-hike.nvim $tag"
  from="From plugins/neovim of Commit-Hike at $(git rev-parse --short HEAD)."
  if [ -n "$parent" ]; then
    commit=$(git commit-tree "$tree" -p "$parent" -m "$msg" -m "$from")
  else
    commit=$(git commit-tree "$tree" -m "$msg" -m "$from")
  fi
fi

git push --force "$url" "$commit:refs/heads/main" "$commit:refs/tags/$tag"
echo "Published $tag: $(git ls-tree --name-only "$commit" | tr '\n' ' ')"

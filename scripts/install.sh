#!/bin/sh
# Installs commit-hike, the Commit Hike core, from GitHub Releases:
#
#   curl -fsSL https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/scripts/install.sh | sh
#
# It downloads the archive for this OS and CPU, checks it against the
# release's checksums.txt and puts `commit-hike` into ~/.local/bin.
#
# Options (as flags, or environment variables for `curl ... | sh`):
#   --version 0.7.0   COMMIT_HIKE_VERSION      a release (default: the latest)
#   --to DIR          COMMIT_HIKE_INSTALL_DIR  where to put it (default: ~/.local/bin)
#   --from DIR        COMMIT_HIKE_FROM         local archives instead of GitHub (e.g. dist/ from task release:snapshot)
#
# Windows: use scripts/install.ps1.
set -eu

REPO="Shchusia/Commit-Hike"
version="${COMMIT_HIKE_VERSION:-}"
dest="${COMMIT_HIKE_INSTALL_DIR:-$HOME/.local/bin}"
from="${COMMIT_HIKE_FROM:-}"

while [ $# -gt 0 ]; do
  case "$1" in
    --version) version="${2:?--version needs a value}"; shift 2 ;;
    --to) dest="${2:?--to needs a folder}"; shift 2 ;;
    --from) from="${2:?--from needs a folder}"; shift 2 ;;
    -h | --help) sed -n '2,17p' "$0" 2>/dev/null || true; exit 0 ;;
    *) echo "commit-hike install: unknown option $1" >&2; exit 2 ;;
  esac
done
version="${version#v}"

say() { printf '%s\n' "$*"; }
fail() { printf 'commit-hike install: %s\n' "$*" >&2; exit 1; }

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  MINGW* | MSYS* | CYGWIN*) fail "on Windows run scripts/install.ps1 in PowerShell instead" ;;
  *) fail "no build for $(uname -s); build from source: cd core && go build ./cmd/commit-hike" ;;
esac
case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *) fail "no build for the $(uname -m) CPU; build from source: cd core && go build ./cmd/commit-hike" ;;
esac
archive="commit-hike_${os}_${arch}.tar.gz"

tmp="$(mktemp -d 2>/dev/null || mktemp -d -t commit-hike)"
trap 'rm -rf "$tmp"' EXIT INT TERM

fetch() { # url file
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL --retry 2 -o "$2" "$1"
  elif command -v wget >/dev/null 2>&1; then
    wget -q -O "$2" "$1"
  else
    fail "needs curl or wget"
  fi
}

if [ -n "$from" ]; then
  [ -f "$from/$archive" ] || fail "$from/$archive not found"
  cp "$from/$archive" "$from/checksums.txt" "$tmp/"
  source_name="$from"
else
  if [ -n "$version" ]; then
    base="https://github.com/$REPO/releases/download/v$version"
  else
    base="https://github.com/$REPO/releases/latest/download"
  fi
  say "Downloading $archive${version:+ $version}..."
  fetch "$base/$archive" "$tmp/$archive" || fail "couldn't download $base/$archive"
  fetch "$base/checksums.txt" "$tmp/checksums.txt" || fail "couldn't download the checksums"
  source_name="GitHub Releases"
fi

# The archive must match the release's checksum: a broken or altered download is never installed.
want="$(awk -v f="$archive" '$2 == f { print $1 }' "$tmp/checksums.txt")"
[ -n "$want" ] || fail "checksums.txt has no entry for $archive"
if command -v sha256sum >/dev/null 2>&1; then
  got="$(sha256sum "$tmp/$archive" | awk '{ print $1 }')"
elif command -v shasum >/dev/null 2>&1; then
  got="$(shasum -a 256 "$tmp/$archive" | awk '{ print $1 }')"
else
  fail "needs sha256sum or shasum to check the download"
fi
[ "$got" = "$want" ] || fail "checksum mismatch for $archive: the download is broken, try again"

tar -xzf "$tmp/$archive" -C "$tmp" commit-hike
mkdir -p "$dest"
# Replace atomically, so a running prompt never sees a half-written file.
cp "$tmp/commit-hike" "$dest/.commit-hike.new"
chmod 755 "$dest/.commit-hike.new"
mv -f "$dest/.commit-hike.new" "$dest/commit-hike"

installed="$("$dest/commit-hike" version 2>/dev/null | awk -F'"' '/"version"/ { print $4 }')"
say "Installed commit-hike ${installed:-?} from $source_name into $dest"
case ":$PATH:" in
  *":$dest:"*) ;;
  *) say "Add it to your PATH, e.g. in ~/.profile:  export PATH=\"$dest:\$PATH\"" ;;
esac
if ! "$dest/commit-hike" config >/dev/null 2>&1; then
  say ""
  say "First time? Set it up (your commit e-mail; counts your history too):"
  say "  commit-hike init --email \"\$(git config --global user.email)\""
fi
say "Shell prompt and Neovim: https://github.com/$REPO/blob/master/docs/terminal.md"

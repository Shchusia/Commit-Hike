# Installs commit-hike, the Commit Hike core, from GitHub Releases on Windows:
#
#   irm https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/scripts/install.ps1 | iex
#
# It downloads the zip for this CPU, checks it against the release's
# checksums.txt, puts commit-hike.exe into %LOCALAPPDATA%\Programs\commit-hike
# and adds that folder to your user PATH.
#
# Options (environment variables, so they work with `| iex` too):
#   $env:COMMIT_HIKE_VERSION = "0.7.0"         a release (default: the latest)
#   $env:COMMIT_HIKE_INSTALL_DIR = "C:\tools"  where to put it
#   $env:COMMIT_HIKE_FROM = "dist"             local archives instead of GitHub (task release:snapshot)
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue" # Invoke-WebRequest is much faster without the progress bar

$Repo = "Shchusia/Commit-Hike"
$Version = "$env:COMMIT_HIKE_VERSION".TrimStart("v")
$From = "$env:COMMIT_HIKE_FROM"
$Dest = if ($env:COMMIT_HIKE_INSTALL_DIR) { $env:COMMIT_HIKE_INSTALL_DIR }
        elseif ($env:LOCALAPPDATA) { Join-Path $env:LOCALAPPDATA "Programs\commit-hike" }
        else { Join-Path $HOME ".local/bin" }

function Fail([string]$Message) {
  Write-Error "commit-hike install: $Message" -ErrorAction Continue
  exit 1
}

$Arch = switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture) {
  "X64" { "amd64" }
  "Arm64" { "arm64" }
  default { Fail "no build for this CPU ($_); build from source: cd core; go build ./cmd/commit-hike" }
}
$Os = if ($IsLinux) { "linux" } elseif ($IsMacOS) { "darwin" } else { "windows" }
$Ext = if ($Os -eq "windows") { "zip" } else { "tar.gz" }
$Exe = if ($Os -eq "windows") { "commit-hike.exe" } else { "commit-hike" }
$Archive = "commit-hike_${Os}_${Arch}.$Ext"

$Tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("commit-hike-" + [System.Guid]::NewGuid())
New-Item -ItemType Directory -Path $Tmp | Out-Null
try {
  if ($From) {
    if (-not (Test-Path (Join-Path $From $Archive))) { Fail "$(Join-Path $From $Archive) not found" }
    Copy-Item (Join-Path $From $Archive), (Join-Path $From "checksums.txt") $Tmp
    $Source = $From
  } else {
    $Base = if ($Version) { "https://github.com/$Repo/releases/download/v$Version" }
            else { "https://github.com/$Repo/releases/latest/download" }
    Write-Host "Downloading $Archive $Version..."
    try {
      Invoke-WebRequest -UseBasicParsing -Uri "$Base/$Archive" -OutFile (Join-Path $Tmp $Archive)
      Invoke-WebRequest -UseBasicParsing -Uri "$Base/checksums.txt" -OutFile (Join-Path $Tmp "checksums.txt")
    } catch { Fail "couldn't download $Base/$Archive ($($_.Exception.Message))" }
    $Source = "GitHub Releases"
  }

  # The archive must match the release's checksum: a broken or altered download is never installed.
  $Want = (Get-Content (Join-Path $Tmp "checksums.txt") | Where-Object { ($_ -split "\s+")[1] -eq $Archive } |
           ForEach-Object { ($_ -split "\s+")[0] }) | Select-Object -First 1
  if (-not $Want) { Fail "checksums.txt has no entry for $Archive" }
  $Got = (Get-FileHash -Algorithm SHA256 (Join-Path $Tmp $Archive)).Hash.ToLowerInvariant()
  if ($Got -ne $Want.ToLowerInvariant()) { Fail "checksum mismatch for ${Archive}: the download is broken, try again" }

  $Out = Join-Path $Tmp "x"
  if ($Ext -eq "zip") { Expand-Archive -Path (Join-Path $Tmp $Archive) -DestinationPath $Out }
  else { New-Item -ItemType Directory -Path $Out | Out-Null; tar -xzf (Join-Path $Tmp $Archive) -C $Out }
  New-Item -ItemType Directory -Force -Path $Dest | Out-Null
  Copy-Item -Force (Join-Path $Out $Exe) (Join-Path $Dest $Exe)
  if ($Os -ne "windows") { chmod 755 (Join-Path $Dest $Exe) }
} finally {
  Remove-Item -Recurse -Force $Tmp -ErrorAction SilentlyContinue
}

$Installed = try { (& (Join-Path $Dest $Exe) version | ConvertFrom-Json).data.version } catch { "?" }
Write-Host "Installed commit-hike $Installed from $Source into $Dest"

if ($Os -eq "windows") {
  $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
  if (-not (($UserPath -split ";") -contains $Dest)) {
    [Environment]::SetEnvironmentVariable("Path", (($UserPath.TrimEnd(";"), $Dest) -join ";").TrimStart(";"), "User")
    Write-Host "Added $Dest to your PATH: open a new terminal to use it."
  }
}
$Config = & (Join-Path $Dest $Exe) config 2>$null | Out-String
if (-not ($Config -match '"ok":\s*true')) {
  Write-Host ""
  Write-Host "First time? Set it up (your commit e-mail; counts your history too):"
  Write-Host '  commit-hike init --email "$(git config --global user.email)"'
}
Write-Host "Shell prompt and Neovim: https://github.com/$Repo/blob/master/docs/terminal.md"

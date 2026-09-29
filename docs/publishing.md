# Publishing Commit Hike

Commit Hike goes to three catalogs:

| Catalog | For | Package |
|---|---|---|
| JetBrains Marketplace | PyCharm, IntelliJ IDEA and the other JetBrains IDEs | `plugins/jetbrains/build/distributions/commit-hike-jetbrains-<version>.zip` |
| Visual Studio Marketplace | VS Code | `plugins/vscode/commit-hike-<version>.vsix` |
| Open VSX | VSCodium, Cursor, Windsurf and other editors without Microsoft's store | the same `.vsix` |

Every published build is a **prod** build: the hike view only looks back from
where the user is. `task release:check` refuses anything else.

## 1. Before the first release

1. **Everything passes:** `task check`, then `task jetbrains:verify` (and
   `task jetbrains:verify-all` once, which downloads several IDEs).
2. **Try it on macOS and Windows.** The core is cross-compiled for both but has
   to be run once on each: install the zip or `.vsix` on a friend's machine.
3. **Content rights.** Every built-in route must be yours or properly licensed,
   and every picture in `assets/` needs a known source and license. Routes set
   in other people's worlds live in `extras/` and are never shipped;
   `task release:check` blocks Tolkien's world among built-ins.
4. **Choose who publishes** (the plugin id can never change after the first upload):

   ```bash
   task release:configure OWNER=your-github-name NAME="Your Name" EMAIL=you@example.com
   # optional: PLUGIN_ID=io.github.you.commithike PUBLISHER=your-vscode-publisher REPO=commit-hike
   ```

   This sets the JetBrains plugin id and vendor, the VS Code publisher and links,
   and the Go module path, and writes `release.env` (public information only).
5. **Publish the repository** on GitHub: the plugin ships native binaries, and
   open source is what lets moderators and users trust them.
6. **Screenshots.** Add a few to the README and to both marketplace pages. Images in `plugins/vscode/README.md` are absolute links to the default branch (`https://raw.githubusercontent.com/OWNER/REPO/master/...`, or `BRANCH` from `release.env`), never to a feature branch: the marketplace page keeps them after that branch is deleted. `task release:check` checks this, and that every picture is already on `origin/master`.

## 2. Accounts and tokens

Tokens never go into files. Locally, export them in your shell; in CI, add them
as secrets of the `marketplaces` environment (Settings → Environments, with
yourself as a required reviewer).

| Secret | Where to get it |
|---|---|
| `JETBRAINS_TOKEN` | plugins.jetbrains.com → your profile → **My Tokens** |
| `CERTIFICATE_CHAIN`, `PRIVATE_KEY`, `PRIVATE_KEY_PASSWORD` | your plugin-signing key, see below |
| `VSCE_PAT` | an Azure DevOps personal access token with **Marketplace → Manage**, scoped to your organization |
| `OVSX_PAT` | open-vsx.org → Settings → Access Tokens (needs an Eclipse account and the publisher agreement) |

**VS Code Marketplace publisher.** Create it at marketplace.visualstudio.com/manage
with the same id as `publisher` in `plugins/vscode/package.json`.
Azure DevOps retires *global* personal access tokens on 1 December 2026: create
an organization-scoped token, or publish with Microsoft Entra ID
(`npx vsce publish --azure-credential`) instead of `VSCE_PAT`.

**Open VSX namespace.** Once: `npx ovsx create-namespace <publisher> -p "$OVSX_PAT"`.

**Plugin signing (JetBrains).** Without a signature the IDE warns users when they
install the plugin. Create a key and a self-signed certificate once:

```bash
openssl genpkey -aes-256-cbc -algorithm RSA -out private_encrypted.pem -pkeyopt rsa_keygen_bits:4096
openssl rsa -in private_encrypted.pem -out private.pem          # asks for the password you chose
openssl req -key private.pem -new -x509 -days 3650 -out chain.crt
export CERTIFICATE_CHAIN="$(cat chain.crt)" PRIVATE_KEY="$(cat private.pem)" PRIVATE_KEY_PASSWORD='your password'
```

Keep `private.pem` and the password outside the repository (a password manager).

## 3. The very first upload (by hand)

- **JetBrains Marketplace:** the first version can't be published by a tool.
  Run `task release:build`, then on plugins.jetbrains.com choose **Upload plugin**,
  pick the zip, the MIT license and the repository URL. Moderators review it
  before it becomes public.
- **Visual Studio Marketplace:** the simplest first step is the website too:
  drag the `.vsix` from `task release:build` onto your publisher page.
- **Open VSX:** `task ovsx:publish` works from the start.

## 4. Every release after that

```bash
# 1. write what changed under [Unreleased] in CHANGELOG.md
task version:set V=0.2.0          # VERSION, package.json, READMEs, CHANGELOG section
task release:check                # ready?
task release:build                # prod packages, after every check and verification
git commit -am "Release 0.2.0" && git tag v0.2.0 && git push --follow-tags
```

Then upload by hand: the signed JetBrains zip (`./gradlew signPlugin` in
`plugins/jetbrains`, see above) on plugins.jetbrains.com → **Upload Update**,
the `.vsix` with `task ovsx:publish` and on the Visual Studio Marketplace
publisher page. A GitHub release with `gh release create` and the notes from
`node scripts/release-notes.mjs` is optional.

To publish from CI later, add a workflow that runs `task release:build` and the
`*:publish` tasks on version tags, with the secrets from section 2.

**Pre-releases.** A version like `0.3.0-beta` goes to the JetBrains `beta`
channel (users opt in to it) and is marked as a pre-release on GitHub.

## 5. What each task checks

| Task | Checks |
|---|---|
| `release:check` | semantic version; `package.json` matches `VERSION`; `CHANGELOG.md` has notes for it; the publisher is configured and the placeholder id is gone; a prod build; no built-in route in Tolkien's world |
| `release:build` | all of the above, plus `task check` (linters, vulnerabilities, tests, coverage gate) and `jetbrains:verify` |
| CI on every push | linters, core tests with the coverage gate, route packs, both plugins' tests and packages |

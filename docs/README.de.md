# Commit Hike

[![JetBrains Plugin](https://img.shields.io/jetbrains/plugin/v/34595?label=JetBrains&logo=jetbrains)](https://plugins.jetbrains.com/plugin/34595)
[![JetBrains Downloads](https://img.shields.io/jetbrains/plugin/d/34595?label=downloads)](https://plugins.jetbrains.com/plugin/34595)
[![Open VSX](https://img.shields.io/open-vsx/v/shchusia/commit-hike?label=Open%20VSX&logo=eclipseide)](https://open-vsx.org/extension/shchusia/commit-hike)
[![Open VSX Downloads](https://img.shields.io/open-vsx/dt/shchusia/commit-hike?label=downloads)](https://open-vsx.org/extension/shchusia/commit-hike)
[![GitHub stars](https://img.shields.io/github/stars/Shchusia/Commit-Hike?style=flat&logo=github)](https://github.com/Shchusia/Commit-Hike)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue)](../LICENSE)

*[English](../README.md) · [Українська](README.uk.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Español](README.es.md) · [Neuigkeiten](../CHANGELOG.md)*

Jeder Commit bringt dich auf einem Wanderweg voran, direkt in deiner IDE. Ein
Tag stetiger Arbeit führt dich etwa zehn Kilometer über einen echten
Bergkamm, durch eine Wüste oder ein Märchen — mit einer Szene, die sich
unterwegs verändert, einer Karte, Geschichten, Fakten und Erfolgen.

<p>
  <img src="https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/docs/screenshots/de-hike.png" width="49%" alt="Wanderung: der Weg in der Seitenansicht">
  <img src="https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/docs/screenshots/de-map.png" width="49%" alt="Karte: der echte GPS-Track des Tschornohora-Kamms">
</p>
<p>
  <img src="https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/docs/screenshots/de-places.png" width="49%" alt="Orte: jeder Halt und seine Geschichte">
  <img src="https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/docs/screenshots/de-stats.png" width="49%" alt="Statistik: Serien, Tagesstrecke und Erfolge">
</p>

## Funktionen

- **Echte, literarische und eigene Routen.** Der Tschornohora-Kamm über die Howerla und das Tongariro Alpine Crossing in Neuseeland auf ihren echten GPS-Tracks, Santiagos Weg aus „Der Alchimist“ durch das echte Spanien, Marokko, die Sahara und Ägypten, ein Märchen in den Karpaten oder eine Fantasy-Reise an sieben Leuchttürmen vorbei. Jede Route hat eigene Halte, eine Geschichte, Fakten und Erfolge.
- **Fünf Sprachen.** Englisch, Ukrainisch, Polnisch, Deutsch und Spanisch — wie deine IDE oder in den Einstellungen gewählt.
- **Serien.** Beende eine Route und wandere weiter: die Karpaten (Tschornohora → Swydiwez → Gorgany) und die langen Wege (Tour du Mont Blanc → Camino Francés nach Santiago).
- **Echte Höhenmeter.** Routen haben Höhenprofile: Das Gelände steigt und fällt, und du siehst Höhe, Steigung, geschaffte Höhenmeter und den Anstieg bis zum nächsten Halt.
- **Eine lebendige Szene.** Der Himmel folgt deiner Uhr, das Wetter wechselt von Tag zu Tag, und die Landschaft — Wälder, Wiesen, Steppe, Wüste, Fels, Schnee, Seen, Meer, Vulkane — geht fließend ineinander über. Im Winter Schnee, im Herbst goldene Wiesen, an Tagen ohne Commits ein Lager am Feuer.
- **Karte und Minikarte.** Eine topografische Karte des ganzen Wegs mit Höhenlinien, Schummerung, Wäldern und Seen; echte Routen nach ihrem GPS-Track in echtem Maßstab.
- **Echte Entfernungen, deine Schwierigkeit.** Ein typischer Tag mit Commits bringt dich auf Mittel etwa 10 km weit (12,5 auf Leicht, 8 auf Schwer), eine lange Route ist also ein Ziel für Monate.
- **Keine Spoiler.** Den zurückgelegten Weg kannst du dir ansehen, was vor dir liegt, siehst du erst, wenn du dort bist.
- **Dein Jahr auf dem Weg und ein Pass.** Ein Aktivitätskalender, der beste Tag, die längste Serie, und im Pass ein Stempel mit Datum für jeden erreichten Halt.
- **Teilen.** Eine Postkarte von deinem Standort: speichern, kopieren oder mit fertigem Text auf X, Bluesky, Mastodon, Threads, LinkedIn, Facebook, Telegram oder Reddit posten.
- **Auch im Terminal.** Eine Zeile in deinem Shell-Prompt (starship, bash, zsh, fish) und in der Statuszeile von Neovim, vom selben Kern: `🥾 16,0 km · See Nessamowyte`.
- **Gemeinsam wandern.** Alle, die in ein Projekt committen, auf demselben Weg, mit Rangliste, Wochenbilanz und Teamziel: einer Route, die ihr gemeinsam geht, die Commits aller zählen zusammen.
- **Faire Strecke.** Lock-Dateien, generierter Code, reine Leerzeichen-Änderungen und fremde Commits zählen nicht; nach Squash, Rebase oder Amend wird neu gezählt, nie doppelt.
- **Privat von Grund auf.** Nichts wird in deine Repositories geschrieben, nichts verlässt deinen Computer. Kein Code, keine Commit-Nachrichten, keine Datei- oder Repository-Namen werden gespeichert.

## Unterstützte IDEs

| IDE | Bezugsquelle |
|---|---|
| JetBrains-IDEs 2024.3+: PyCharm, IntelliJ IDEA, GoLand, WebStorm, PhpStorm, RubyMine, CLion, Rider… | [JetBrains Marketplace](https://plugins.jetbrains.com/plugin/34595) |
| VSCodium, Cursor, Windsurf und andere Editoren mit Open VSX | [Open VSX](https://open-vsx.org/extension/shchusia/commit-hike) |
| VS Code 1.85+ | bald im Visual Studio Marketplace; bis dahin das `.vsix` von Open VSX installieren |
| Neovim 0.10+ und jeder Shell-Prompt: starship, bash, zsh, fish | [docs/terminal.md](terminal.md) |
| Zed | über sein Terminal, siehe [docs/terminal.md](terminal.md#zed) |

## Installation

**JetBrains-IDEs.** Öffne **Settings → Plugins → Marketplace**, suche nach
*Commit Hike* und klicke auf **Install**.

**VSCodium, Cursor, Windsurf.** Öffne die Erweiterungsansicht, suche nach
*Commit Hike* und klicke auf **Install**.

**VS Code.** Solange Commit Hike nicht im Visual Studio Marketplace ist, lade
das `.vsix` von der [Open-VSX-Seite](https://open-vsx.org/extension/shchusia/commit-hike)
herunter (**Download**) und wähle in VS Code in der Erweiterungsansicht
**⋯ → Install from VSIX…**.

**Terminal und Neovim.** Installiere den Kern mit einem Befehl und füge ihn
deinem Prompt oder deiner Statuszeile hinzu, siehe [docs/terminal.md](terminal.md):

```sh
curl -fsSL https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/scripts/install.sh | sh
```

## Verwendung

Öffne das Werkzeugfenster **Commit Hike** (JetBrains) oder die Commit-Hike-
Ansicht in der Aktivitätsleiste (VS Code). Beim ersten Mal fragt eine kurze
Einrichtung, welche Projekte zählen und wo deine Reise beginnt. Danach einfach
committen.

Reiter: **Wanderung** (die Szene, die heutige Strecke, die Geschichte des
Orts), **Karte**, **Orte**, **Team** und **Statistik**. Alles andere — Route
wählen, eigene Routen, Wanderer-Symbol, Schwierigkeit, Sprache — findest du
unter **Tools → Commit Hike** (JetBrains) oder in der Befehlspalette unter
*Commit Hike* (VS Code).

**↗ Teilen** unter der Wanderung (und auf der Jahrespostkarte in der Statistik)
speichert oder kopiert eine Postkarte oder öffnet einen Beitrag mit fertigem
Text; die Postkarte liegt schon in der Zwischenablage zum Einfügen.

<p><img src="https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/docs/screenshots/de-share.png" width="49%" alt="Das Menü Teilen: Postkarte speichern, kopieren oder posten"></p>

## Für Entwickler

Bauen, Tests, Releases, Architektur und eigene Routen stehen im
[englischen README](../README.md#development) und in [docs/](docs/).
Übersetzungen von Routen und Oberfläche sind willkommen: siehe
[docs/routes.md](routes.md).

## Das Projekt unterstützen

Commit Hike ist kostenlos und quelloffen. Wenn dir die Wanderung gefällt:

- ⭐ gib dem Repository einen Stern — so finden es andere;
- bewerte es im [JetBrains Marketplace](https://plugins.jetbrains.com/plugin/34595) oder auf [Open VSX](https://open-vsx.org/extension/shchusia/commit-hike/reviews);
- melde Fehler und Ideen in den [Issues](https://github.com/Shchusia/Commit-Hike/issues);
- erstelle eine Route und teile sie: siehe [docs/routes.md](routes.md).

## Lizenz

[MIT](../LICENSE)

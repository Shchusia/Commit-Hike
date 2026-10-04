# Commit Hike

[![JetBrains Plugin](https://img.shields.io/jetbrains/plugin/v/34595?label=JetBrains&logo=jetbrains)](https://plugins.jetbrains.com/plugin/34595)
[![JetBrains Downloads](https://img.shields.io/jetbrains/plugin/d/34595?label=downloads)](https://plugins.jetbrains.com/plugin/34595)
[![Open VSX](https://img.shields.io/open-vsx/v/shchusia/commit-hike?label=Open%20VSX&logo=eclipseide)](https://open-vsx.org/extension/shchusia/commit-hike)
[![Open VSX Downloads](https://img.shields.io/open-vsx/dt/shchusia/commit-hike?label=downloads)](https://open-vsx.org/extension/shchusia/commit-hike)
[![GitHub stars](https://img.shields.io/github/stars/Shchusia/Commit-Hike?style=flat&logo=github)](https://github.com/Shchusia/Commit-Hike)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue)](../LICENSE)

*[English](../README.md) · [Українська](README.uk.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Español](README.es.md) · [Co nowego](../CHANGELOG.md)*

Każdy twój commit prowadzi cię szlakiem turystycznym, wprost w IDE. Dzień
rzetelnej pracy to około dziesięciu kilometrów prawdziwym grzbietem górskim,
przez pustynię albo przez baśń — ze sceną, która zmienia się po drodze, mapą,
historiami, ciekawostkami i osiągnięciami.

<p>
  <img src="https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/docs/screenshots/pl-hike.png" width="49%" alt="Wędrówka: widok szlaku z boku">
  <img src="https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/docs/screenshots/pl-map.png" width="49%" alt="Mapa: prawdziwy ślad GPS grzbietu Czarnohory">
</p>
<p>
  <img src="https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/docs/screenshots/pl-places.png" width="49%" alt="Miejsca: każdy przystanek i jego historia">
  <img src="https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/docs/screenshots/pl-stats.png" width="49%" alt="Statystyki: serie, dzienny dystans i osiągnięcia">
</p>

## Możliwości

- **Prawdziwe, literackie i autorskie trasy.** Czarnohora przez Howerlę i nowozelandzkie Tongariro Alpine Crossing na prawdziwych śladach GPS, droga Santiago z „Alchemika” przez prawdziwą Hiszpanię, Maroko, Saharę i Egipt, baśń w Karpatach albo fantastyczna wędrówka wzdłuż siedmiu latarni. Każda trasa ma własne przystanki, historię, ciekawostki i osiągnięcia.
- **Pięć języków.** Angielski, ukraiński, polski, niemiecki i hiszpański — jak w IDE albo wybrany w ustawieniach.
- **Serie.** Skończ trasę i idź dalej: Karpaty (Czarnohora → Świdowiec → Gorgany) i długie drogi (Tour du Mont Blanc → Camino Francés do Santiago).
- **Prawdziwe wspinanie.** Trasy mają profile wysokości: teren wznosi się i opada, a ty widzisz wysokość, nachylenie, pokonane przewyższenie i podejście do następnego przystanku.
- **Żywa scena.** Niebo idzie za twoim zegarem, pogoda zmienia się z dnia na dzień, a krajobraz — lasy, łąki, step, pustynia, skały, śnieg, jeziora, morze, wulkany — płynnie przechodzi jeden w drugi. Zimą śnieg, jesienią złote łąki, w dni bez commitów obóz przy ognisku.
- **Mapa i minimapa.** Mapa topograficzna całego szlaku: poziomice, cieniowanie, lasy i jeziora; prawdziwe trasy według śladu GPS w prawdziwej skali.
- **Prawdziwe odległości, twój poziom trudności.** Typowy dzień commitów to około 10 km na średnim poziomie (12,5 na łatwym, 8 na trudnym), więc długa trasa to cel na miesiące.
- **Bez spoilerów.** Przebytą drogę możesz obejrzeć, a to, co przed tobą, zobaczysz dopiero na miejscu.
- **Rok na szlaku i paszport.** Kalendarz aktywności, najlepszy dzień, najdłuższa seria, a w paszporcie stempel z datą za każdy przystanek.
- **Udostępnij.** Pocztówka z miejscem, w którym jesteś: zapisz ją, skopiuj albo opublikuj w X, Bluesky, Mastodonie, Threads, LinkedInie, na Facebooku, w Telegramie lub na Reddicie z gotowym tekstem.
- **Także w terminalu.** Jedna linijka w znaku zachęty (starship, bash, zsh, fish) i w pasku stanu Neovima, z tego samego rdzenia: `🥾 16,0 km · Jezioro Niesamowite`.
- **Wędrujcie razem.** Wszyscy, którzy commitują do projektu, na jednym szlaku, z tabelą wyników, podsumowaniem tygodnia i celem zespołu: trasą, którą przechodzicie wspólnie, sumując commity wszystkich.
- **Uczciwy dystans.** Pliki lock, kod generowany, zmiany samych spacji i cudze commity się nie liczą; po squash, rebase czy amend historia jest przeliczana, nigdy liczona dwa razy.
- **Prywatność przede wszystkim.** Nic nie trafia do twoich repozytoriów i nic nie opuszcza twojego komputera. Nie zapisujemy kodu, opisów commitów, nazw plików ani repozytoriów.

## Obsługiwane IDE

| IDE | Gdzie pobrać |
|---|---|
| IDE JetBrains 2024.3+: PyCharm, IntelliJ IDEA, GoLand, WebStorm, PhpStorm, RubyMine, CLion, Rider… | [JetBrains Marketplace](https://plugins.jetbrains.com/plugin/34595) |
| VSCodium, Cursor, Windsurf i inne edytory korzystające z Open VSX | [Open VSX](https://open-vsx.org/extension/shchusia/commit-hike) |
| VS Code 1.85+ | wkrótce w Visual Studio Marketplace; do tego czasu zainstaluj `.vsix` z Open VSX |
| Neovim 0.10+ i dowolny znak zachęty: starship, bash, zsh, fish | [commit-hike.nvim](https://github.com/Shchusia/commit-hike.nvim), [docs/terminal.md](terminal.md) |
| Zed | przez jego terminal, zob. [docs/terminal.md](terminal.md#zed) |

## Instalacja

**IDE JetBrains.** Otwórz **Settings → Plugins → Marketplace**, wyszukaj
*Commit Hike* i kliknij **Install**.

**VSCodium, Cursor, Windsurf.** Otwórz widok rozszerzeń, wyszukaj *Commit Hike*
i kliknij **Install**.

**VS Code.** Dopóki Commit Hike nie ma w Visual Studio Marketplace, pobierz
`.vsix` ze [strony Open VSX](https://open-vsx.org/extension/shchusia/commit-hike)
(**Download**), a potem w VS Code: widok rozszerzeń, **⋯ → Install from VSIX…**.

**Terminal i Neovim.** Zainstaluj rdzeń jednym poleceniem, a potem dodaj go do
znaku zachęty lub paska stanu, zob. [docs/terminal.md](terminal.md):

```sh
curl -fsSL https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/scripts/install.sh | sh
```

W Neovimie 0.10+ z lazy.nvim, a potem dodaj `require("commit-hike").statusline` do paska stanu:

```lua
{ "Shchusia/commit-hike.nvim", version = "*", opts = {} }
```

## Jak używać

Otwórz okno **Commit Hike** (JetBrains) albo widok Commit Hike na pasku
aktywności (VS Code). Za pierwszym razem krótka konfiguracja zapyta, które
projekty się liczą i skąd zaczyna się wędrówka. Potem po prostu commituj.

Zakładki: **Wędrówka** (scena, dzisiejszy dystans, historia miejsca),
**Mapa**, **Miejsca**, **Zespół** i **Statystyki**. Wszystko inne — wybór
trasy, własne trasy, ikona wędrowca, poziom trudności, język — znajdziesz w
**Tools → Commit Hike** (JetBrains) albo w palecie poleceń pod *Commit Hike*
(VS Code).

**↗ Udostępnij** pod wędrówką (i na pocztówce roku w statystykach) zapisuje lub
kopiuje pocztówkę albo otwiera post z gotowym tekstem; pocztówka jest już w
schowku, wystarczy ją wkleić.

<p><img src="https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/docs/screenshots/pl-share.png" width="49%" alt="Menu Udostępnij: zapisz, skopiuj albo opublikuj pocztówkę"></p>

## Dla programistów

Budowanie, testy, wydania, architektura i dodawanie tras są opisane w
[angielskim README](../README.md#development) oraz w [docs/](./). Tłumaczenia
tras i interfejsu są mile widziane: zob. [docs/routes.md](routes.md).
Wydania, w tym wtyczki do Neovima, opisuje [docs/publishing.md](publishing.md).

## Wesprzyj projekt

Commit Hike jest darmowy i otwarty. Jeśli lubisz tę wędrówkę:

- ⭐ daj gwiazdkę repozytorium — pomaga to innym ją znaleźć;
- oceń wtyczkę na [JetBrains Marketplace](https://plugins.jetbrains.com/plugin/34595) lub [Open VSX](https://open-vsx.org/extension/shchusia/commit-hike/reviews);
- zgłaszaj błędy i pomysły w [issues](https://github.com/Shchusia/Commit-Hike/issues);
- stwórz trasę i podziel się nią: zob. [docs/routes.md](routes.md).

## Licencja

[MIT](../LICENSE)

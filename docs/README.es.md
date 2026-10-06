# Commit Hike

[![JetBrains Plugin](https://img.shields.io/jetbrains/plugin/v/34595?label=JetBrains&logo=jetbrains)](https://plugins.jetbrains.com/plugin/34595)
[![JetBrains Downloads](https://img.shields.io/jetbrains/plugin/d/34595?label=downloads)](https://plugins.jetbrains.com/plugin/34595)
[![Open VSX](https://img.shields.io/open-vsx/v/shchusia/commit-hike?label=Open%20VSX&logo=eclipseide)](https://open-vsx.org/extension/shchusia/commit-hike)
[![Open VSX Downloads](https://img.shields.io/open-vsx/dt/shchusia/commit-hike?label=downloads)](https://open-vsx.org/extension/shchusia/commit-hike)
[![GitHub stars](https://img.shields.io/github/stars/Shchusia/Commit-Hike?style=flat&logo=github)](https://github.com/Shchusia/Commit-Hike)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue)](../LICENSE)

*[English](../README.md) · [Українська](README.uk.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Español](README.es.md) · [Novedades](../CHANGELOG.md)*

Cada commit que haces te lleva por un sendero, dentro de tu IDE. Un día de
trabajo constante te lleva unos diez kilómetros por una cresta de montaña
real, un desierto o un cuento, con una escena que cambia por el camino, un
mapa, historias, datos curiosos y logros.

<p>
  <img src="https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/docs/screenshots/es-hike.png" width="49%" alt="Travesía: el sendero en vista lateral">
  <img src="https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/docs/screenshots/es-map.png" width="49%" alt="Mapa: el track GPS real de la cresta de Chornohora">
</p>
<p>
  <img src="https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/docs/screenshots/es-places.png" width="49%" alt="Lugares: cada parada y su historia">
  <img src="https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/docs/screenshots/es-stats.png" width="49%" alt="Estadísticas: rachas, distancia diaria y logros">
</p>

## Funciones

- **Rutas reales, literarias y propias.** La cresta de Chornohora por el Hoverla y el Tongariro Alpine Crossing de Nueva Zelanda sobre sus tracks GPS reales, el camino de Santiago de «El Alquimista» por la España, Marruecos, el Sáhara y Egipto reales, un cuento por los Cárpatos o un viaje de fantasía junto a siete faros. Cada ruta tiene sus paradas, su historia, datos curiosos y logros.
- **Rutas del sitio web.** Busca rutas de otros caminantes en [commit-hike.dev](https://commit-hike.dev) desde el IDE: búsqueda, filtros, portadas, valoraciones; instálalas con un clic y actualízalas cuando el autor publique una nueva versión. En Neovim: `:CommitHike find`.
- **Cinco idiomas.** Inglés, ucraniano, polaco, alemán y español, como tu IDE o elegido en los ajustes.
- **Series.** Termina una ruta y sigue a la siguiente: los Cárpatos (Chornohora → Svydovets → Gorgany) y los caminos largos (Tour du Mont Blanc → Camino Francés a Santiago).
- **Desnivel de verdad.** Las rutas tienen perfil de altitud: el terreno sube y baja, y ves la altitud, la pendiente, el desnivel acumulado y la subida hasta la próxima parada.
- **Una escena viva.** El cielo sigue tu reloj, el tiempo cambia cada día y el paisaje —bosques, praderas, estepa, desierto, roca, nieve, lagos, mar, volcanes— se funde de uno a otro. Nieve en invierno, praderas doradas en otoño y un campamento junto al fuego los días sin commits.
- **Mapa y minimapa.** Un mapa topográfico de todo el sendero con curvas de nivel, sombreado, bosques y lagos; las rutas reales siguen su track GPS a escala real.
- **Distancias reales, tu dificultad.** Un día típico de commits te lleva unos 10 km en dificultad media (12,5 en fácil, 8 en difícil), así que una ruta larga es una meta de meses.
- **Sin spoilers.** Puedes mirar atrás el camino recorrido; lo que viene lo verás al llegar.
- **Tu año en el sendero y un pasaporte.** Un calendario de actividad, el mejor día, la racha más larga y un sello con fecha en el pasaporte por cada parada alcanzada.
- **Compártelo.** Una postal de dónde estás: guárdala, cópiala o publícala en X, Bluesky, Mastodon, Threads, LinkedIn, Facebook, Telegram o Reddit con el texto listo.
- **También en la terminal.** Una línea en el prompt de tu shell (starship, bash, zsh, fish) y en la barra de estado de Neovim, del mismo núcleo: `🥾 16,0 km · Lago Nesamovyte`.
- **Caminad juntos.** Todos los que hacen commits en un proyecto, en el mismo sendero, con clasificación, el resumen de la semana y una meta del equipo: una ruta que recorréis juntos, sumando los commits de todos.
- **Distancia justa.** Los ficheros lock, el código generado, los cambios solo de espacios y los commits ajenos no cuentan; tras un squash, rebase o amend se recuenta, nunca se cuenta dos veces.
- **Privado por diseño.** No se escribe nada en tus repositorios y nada sobre ti sale de tu ordenador: solo se consulta [commit-hike.dev](https://commit-hike.dev) cuando abres *Rutas del sitio web* o instalas una ruta, y se lee el catálogo público sin enviar nada sobre ti. No se guarda código, mensajes de commit ni nombres de ficheros o repositorios.

## IDE compatibles

| IDE | Dónde conseguirlo |
|---|---|
| IDE de JetBrains 2024.3+: PyCharm, IntelliJ IDEA, GoLand, WebStorm, PhpStorm, RubyMine, CLion, Rider… | [JetBrains Marketplace](https://plugins.jetbrains.com/plugin/34595) |
| VSCodium, Cursor, Windsurf y otros editores que usan Open VSX | [Open VSX](https://open-vsx.org/extension/shchusia/commit-hike) |
| VS Code 1.85+ | pronto en Visual Studio Marketplace; mientras tanto, instala el `.vsix` de Open VSX |
| Neovim 0.10+ y cualquier prompt de shell: starship, bash, zsh, fish | [commit-hike.nvim](https://github.com/Shchusia/commit-hike.nvim), [docs/terminal.md](terminal.md) |
| Zed | a través de su terminal, ver [docs/terminal.md](terminal.md#zed) |

## Instalación

**IDE de JetBrains.** Abre **Settings → Plugins → Marketplace**, busca
*Commit Hike* y pulsa **Install**.

**VSCodium, Cursor, Windsurf.** Abre la vista de extensiones, busca
*Commit Hike* y pulsa **Install**.

**VS Code.** Hasta que Commit Hike esté en Visual Studio Marketplace, descarga
el `.vsix` de la [página de Open VSX](https://open-vsx.org/extension/shchusia/commit-hike)
(**Download**) y en VS Code elige, en la vista de extensiones,
**⋯ → Install from VSIX…**.

**Terminal y Neovim.** Instala el núcleo con un solo comando y añádelo a tu
prompt o a tu barra de estado, ver [docs/terminal.md](terminal.md):

```sh
curl -fsSL https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/scripts/install.sh | sh
```

En Neovim 0.10+, con lazy.nvim, y luego añade `require("commit-hike").statusline` a tu barra de estado:

```lua
{ "Shchusia/commit-hike.nvim", version = "*", opts = {} }
```

## Cómo se usa

Abre la ventana **Commit Hike** (JetBrains) o la vista de Commit Hike en la
barra de actividad (VS Code). La primera vez, una breve configuración pregunta
qué proyectos cuentan y dónde empieza tu viaje. Después, solo haz commits.

Pestañas: **Travesía** (la escena, la distancia de hoy, la historia del lugar),
**Mapa**, **Lugares**, **Equipo** y **Estadísticas**. Todo lo demás —elegir
ruta, rutas propias, el icono del caminante, la dificultad, el idioma— está en
**Tools → Commit Hike** (JetBrains) o en la paleta de comandos, en
*Commit Hike* (VS Code).

**↗ Compartir** bajo la travesía (y en la postal del año, en Estadísticas) guarda o
copia una postal, o abre una publicación con el texto listo; la postal ya está
en el portapapeles para pegarla.

<p><img src="https://raw.githubusercontent.com/Shchusia/Commit-Hike/master/docs/screenshots/es-share.png" width="49%" alt="El menú Compartir: guardar, copiar o publicar la postal"></p>

## Para desarrolladores

La compilación, las pruebas, las versiones, la arquitectura y cómo añadir
rutas están en el [README en inglés](../README.md#development) y en [docs/](./).
Las traducciones de rutas y de la interfaz son bienvenidas: ver
[docs/routes.md](routes.md).
Las versiones, incluido el plugin de Neovim, se describen en [docs/publishing.md](publishing.md).

## Apoya el proyecto

Commit Hike es gratuito y de código abierto. Si disfrutas del camino:

- ⭐ dale una estrella al repositorio: ayuda a que otros lo encuentren;
- valóralo en [JetBrains Marketplace](https://plugins.jetbrains.com/plugin/34595) u [Open VSX](https://open-vsx.org/extension/shchusia/commit-hike/reviews);
- informa de errores y comparte ideas en los [issues](https://github.com/Shchusia/Commit-Hike/issues);
- crea una ruta y compártela: ver [docs/routes.md](routes.md).

## Licencia

[MIT](../LICENSE)

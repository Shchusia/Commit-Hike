# Extras: routes that don't ship with the plugin

These route packs are **not** built into Commit Hike and are never published to
the JetBrains Marketplace, the Visual Studio Marketplace or Open VSX.

`frodo-journey` and `bilbo-journey` follow J. R. R. Tolkien's books: their
names, places and story belong to the Tolkien Estate and Middle-earth
Enterprises, who actively enforce these rights. Shipping them in a public
plugin would get it removed from the marketplaces. They stay here for personal
use only.

Their maps (`assets/middle-earth-map.svg`) are simple schematic maps drawn for
this project, not copies of the books' maps; the trail on them follows the real
road, and every stop is pinned to its place with `x`/`y`.

Use them on your own computer, like any user route:

- all at once: `task routes:install-extras`, then restart the IDE;
- in the IDE: **Tools → Commit Hike → Import a Route…** and choose a folder here;
- to try one without touching your real progress: `task route:try SRC=extras/routes/frodo-journey`.
- Map source: `python3 extras/middle-earth-maps.py` redraws both maps and the paths.

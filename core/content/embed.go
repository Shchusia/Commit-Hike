// Package content embeds the built-in route packs into the binary.
package content

import "embed"

// FS holds routes/<id>/route.json, routes/<id>/locales/*.json and
// series.json (routes walked one after another).
//
//go:embed routes series.json
var FS embed.FS

// Package content embeds the built-in route packs into the binary.
package content

import "embed"

// FS holds routes/<id>/route.json and routes/<id>/locales/*.json.
//
//go:embed routes
var FS embed.FS

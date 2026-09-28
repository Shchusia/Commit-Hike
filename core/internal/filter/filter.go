// Package filter decides which changed files count as "real work".
package filter

import (
	"path"
	"strings"
)

// DefaultIgnore lists lock files, dependency dirs, build output and generated code.
// Pattern rules:
//   - ends with "/"  -> matches a directory segment anywhere in the path
//   - contains "/"   -> glob against the full repo-relative path
//   - otherwise      -> glob against the file name only
var DefaultIgnore = []string{
	// lock files
	"package-lock.json", "yarn.lock", "pnpm-lock.yaml", "bun.lockb",
	"Cargo.lock", "go.sum", "poetry.lock", "Pipfile.lock", "uv.lock",
	"composer.lock", "Gemfile.lock", "Podfile.lock", "pubspec.lock",
	// dependency / build / cache dirs
	"node_modules/", "vendor/", "dist/", "build/", "target/",
	".venv/", "venv/", "__pycache__/", ".next/", ".nuxt/", ".gradle/",
	// generated / minified
	"*.min.js", "*.min.css", "*.map", "*.pb.go", "*_pb2.py", "*_pb2_grpc.py",
	"*.g.dart", "*.freezed.dart", "*.generated.*", "*.snap",
}

// Filter matches paths against ignore patterns. Create it with New.
type Filter struct {
	dirs  []string
	full  []string
	names []string
}

// New builds a filter from the defaults plus user-supplied patterns.
func New(extra []string) *Filter {
	f := &Filter{}
	all := append(append([]string{}, DefaultIgnore...), extra...)
	for _, p := range all {
		p = strings.TrimSpace(p)
		switch {
		case p == "":
		case strings.HasSuffix(p, "/"):
			f.dirs = append(f.dirs, strings.TrimSuffix(p, "/"))
		case strings.Contains(p, "/"):
			f.full = append(f.full, p)
		default:
			f.names = append(f.names, p)
		}
	}
	return f
}

// Ignored reports whether a repo-relative path (forward slashes) is excluded.
func (f *Filter) Ignored(p string) bool {
	segs := strings.Split(p, "/")
	for _, s := range segs[:len(segs)-1] {
		for _, d := range f.dirs {
			if ok, _ := path.Match(d, s); ok {
				return true
			}
		}
	}
	for _, g := range f.full {
		if ok, _ := path.Match(g, p); ok {
			return true
		}
	}
	base := segs[len(segs)-1]
	for _, g := range f.names {
		if ok, _ := path.Match(g, base); ok {
			return true
		}
	}
	return false
}

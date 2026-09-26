package filter

import "testing"

func TestIgnored(t *testing.T) {
	f := New([]string{"docs/generated/*", "*.pdf"})
	ignored := []string{
		"package-lock.json", "web/yarn.lock", "node_modules/x/index.js",
		"a/b/vendor/lib.go", "static/app.min.js", "api/user.pb.go",
		"docs/generated/api.md", "manual.pdf",
	}
	kept := []string{
		"main.go", "src/build.go", "vendor.go", "docs/readme.md", "lockfile.txt",
	}
	for _, p := range ignored {
		if !f.Ignored(p) {
			t.Errorf("%s should be ignored", p)
		}
	}
	for _, p := range kept {
		if f.Ignored(p) {
			t.Errorf("%s should be kept", p)
		}
	}
}

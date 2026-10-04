package cli

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/Shchusia/commit-hike/core/internal/protocol"
)

func call(t *testing.T, dir string, args ...string) (protocol.Envelope, map[string]any, int) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(args, &out, &errOut, "test", dir)
	var raw struct {
		protocol.Envelope
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &raw); err != nil {
		t.Fatalf("stdout is not one JSON document: %v\n%s", err, out.String())
	}
	return raw.Envelope, raw.Data, code
}

func TestEnvelopeAndErrorCodes(t *testing.T) {
	dir := t.TempDir()
	env, _, code := call(t, dir, "status")
	if code != 1 || env.OK || env.API != protocol.Version || env.Error.Code != protocol.CodeNotInitialized {
		t.Fatalf("not initialized: %+v code=%d", env, code)
	}
	env, data, code := call(t, dir, "init", "--email", "me@x.io", "--route", "chornohora-ridge")
	if code != 0 || !env.OK || data["mode"] != "all" {
		t.Fatalf("init: %+v %v", env, data)
	}
	if env, _, _ := call(t, dir, "journey", "--route", "atlantis"); env.Error.Code != protocol.CodeUnknownRoute {
		t.Fatalf("unknown route: %+v", env.Error)
	}
	if env, _, _ := call(t, dir, "bogus"); env.Error.Code != protocol.CodeInvalidArgument {
		t.Fatalf("unknown command: %+v", env.Error)
	}
	if env, _, _ := call(t, dir, "status", "--nope"); env.Error.Code != protocol.CodeInvalidArgument {
		t.Fatalf("bad flag: %+v", env.Error)
	}
	_, data, _ = call(t, dir, "status", "--lang", "uk")
	global := data["global"].(map[string]any)
	if global["route"].(map[string]any)["name"] != "Чорногірський хребет" || data["locale"] != "uk" {
		t.Fatalf("status uk: %v", data)
	}
	_, data, _ = call(t, dir, "locale", "--set", "uk")
	if data["locale"] != "uk" || data["effective"] != "uk" {
		t.Fatalf("locale set: %v", data)
	}
	_, data, _ = call(t, dir, "locale", "--set", "auto", "--lang", "en")
	if data["locale"] != "" || data["effective"] != "en" {
		t.Fatalf("locale auto: %v", data)
	}
	_, data, _ = call(t, dir, "route", "assets", "--id", "seven-lighthouses")
	imgs, _ := data["images"].(map[string]any)
	html, _ := data["html"].(map[string]any)
	if imgs["assets/parchment-map.svg"] == nil || html["assets/aurora.html"] == nil {
		t.Fatalf("assets: %v", data)
	}
	if env, _, _ := call(t, dir, "team", "--repo", t.TempDir()); env.Error.Code != protocol.CodeNotARepo {
		t.Fatalf("team outside a repo: %+v", env.Error)
	}
}

func TestPromptIsPlainTextAndNeverFails(t *testing.T) {
	dir := t.TempDir()
	var out, errOut bytes.Buffer
	// Not set up yet: nothing on stdout, exit 0, so a shell prompt stays clean.
	if code := Run([]string{"prompt", "--repo", t.TempDir()}, &out, &errOut, "test", dir); code != 0 || out.Len() != 0 {
		t.Fatalf("before init: code %d, out %q", code, out.String())
	}
	call(t, dir, "init", "--email", "me@x.io", "--route", "demo-trail")
	out.Reset()
	if code := Run([]string{"prompt", "--repo", t.TempDir(), "--lang", "uk", "--icon", "*"}, &out, &errOut, "test", dir); code != 0 {
		t.Fatalf("code %d", code)
	}
	if got := out.String(); got != "* 0 м · Початок стежки\n" {
		t.Fatalf("prompt = %q", got)
	}
	out.Reset()
	if code := Run([]string{"prompt", "--no-such-flag"}, &out, &errOut, "test", dir); code != 0 || out.Len() != 0 {
		t.Fatalf("a bad flag must not break the prompt: %d %q", code, out.String())
	}
}

func TestProjectGoal(t *testing.T) {
	dir, repo := t.TempDir(), t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"-c", "user.email=me@x.io", "-c", "user.name=Me", "commit", "-q", "--allow-empty", "-m", "first"},
	} {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	call(t, dir, "init", "--email", "me@x.io", "--route", "demo-trail")
	if env, data, _ := call(t, dir, "project", "goal", "--repo", repo, "--route", "camino-frances"); !env.OK || data["goal"] != "camino-frances" {
		t.Fatalf("goal: %+v %v", env.Error, data)
	}
	env, data, _ := call(t, dir, "team", "--repo", repo, "--lang", "es")
	goal, _ := data["goal"].(map[string]any)
	if !env.OK || goal == nil || goal["route_name"] != "Camino Francés" {
		t.Fatalf("team goal: %+v %v", env.Error, data["goal"])
	}
	if env, _, _ := call(t, dir, "project", "goal", "--repo", repo); env.OK || env.Error.Code != protocol.CodeInvalidArgument {
		t.Fatalf("a goal needs a route: %+v", env)
	}
	if env, _, _ := call(t, dir, "project", "goal-off", "--repo", repo); !env.OK {
		t.Fatalf("goal-off: %+v", env.Error)
	}
}

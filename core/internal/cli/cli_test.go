package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/commit-hike/commit-hike/core/internal/protocol"
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
}

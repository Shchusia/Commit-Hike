package i18n

import (
	"reflect"
	"testing"
)

func TestChain(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"uk-UA", []string{"uk-ua", "uk", "en"}},
		{"uk_UA", []string{"uk-ua", "uk", "en"}},
		{"en", []string{"en"}},
		{"", []string{"en"}},
	}
	for _, tt := range tests {
		if got := Chain(tt.in); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Chain(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestBundleFallback(t *testing.T) {
	b := Bundle{
		"en": {"name": "Trail", "text": "Hello"},
		"uk": {"name": "Стежка"},
	}
	chain := Chain("uk")
	if got := b.Text(chain, "name"); got != "Стежка" {
		t.Errorf("got %q", got)
	}
	if got := b.Text(chain, "text"); got != "Hello" { // falls back to English
		t.Errorf("got %q", got)
	}
	if got := b.Missing("en", "uk"); !reflect.DeepEqual(got, []string{"text"}) {
		t.Errorf("missing = %v", got)
	}
}

func TestParseNested(t *testing.T) {
	cat, err := ParseNested([]byte(`{"name":"X","waypoints":{"a":{"name":"A"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	want := Catalog{"name": "X", "waypoints.a.name": "A"}
	if !reflect.DeepEqual(cat, want) {
		t.Errorf("got %v", cat)
	}
	if _, err := ParseNested([]byte(`{"n": 1}`)); err == nil {
		t.Error("numbers must be rejected")
	}
}

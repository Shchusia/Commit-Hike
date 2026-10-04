// Package i18n resolves which language to show and looks up translated text.
//
// The core never builds UI sentences. It returns data plus route content
// (names, stories, achievements) in the requested language; the IDE plugins
// translate their own buttons and labels.
package i18n

import (
	"encoding/json"
	"fmt"
	"strings"
)

// DefaultLocale is the last fallback for every lookup.
const DefaultLocale = "en"

// UILocales are the languages the panel and the IDE plugins are translated
// into. They count as available even before every route is translated:
// route texts fall back to English.
var UILocales = []string{"en", "uk", "pl", "de", "es"}

// Catalog is a flat key -> text map, e.g. "waypoints.trailhead.name".
type Catalog map[string]string

// Chain returns the lookup order for a requested locale:
// "uk-UA" -> ["uk-ua", "uk", "en"]. Duplicates are removed.
func Chain(requested string) []string {
	norm := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(requested), "_", "-"))
	var out []string
	add := func(tag string) {
		if tag == "" {
			return
		}
		for _, t := range out {
			if t == tag {
				return
			}
		}
		out = append(out, tag)
	}
	add(norm)
	if base, _, found := strings.Cut(norm, "-"); found {
		add(base)
	}
	add(DefaultLocale)
	return out
}

// Bundle holds catalogs per locale.
type Bundle map[string]Catalog

// Text returns the first translation found along chain, or "" if none.
func (b Bundle) Text(chain []string, key string) string {
	for _, loc := range chain {
		if s, ok := b[loc][key]; ok && s != "" {
			return s
		}
	}
	return ""
}

// Missing lists keys present in the reference locale but absent in loc.
func (b Bundle) Missing(reference, loc string) []string {
	var out []string
	for k := range b[reference] {
		if _, ok := b[loc][k]; !ok {
			out = append(out, k)
		}
	}
	return out
}

// ParseNested flattens nested JSON objects into dotted keys, so translators
// can edit readable files:
//
//	{"waypoints": {"trailhead": {"name": "Trailhead"}}}  ->  "waypoints.trailhead.name"
func ParseNested(data []byte) (Catalog, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	cat := Catalog{}
	if err := flatten("", raw, cat); err != nil {
		return nil, err
	}
	return cat, nil
}

func flatten(prefix string, v any, out Catalog) error {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			if err := flatten(key, child, out); err != nil {
				return err
			}
		}
	case string:
		out[prefix] = t
	default:
		return fmt.Errorf("key %q: only strings and objects are allowed, got %T", prefix, v)
	}
	return nil
}

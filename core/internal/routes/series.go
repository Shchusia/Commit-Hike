package routes

import (
	"encoding/json"
	"fmt"
	"io/fs"
)

// Series is routes meant to be walked one after another: when one is done,
// the next is offered.
type Series struct {
	ID     string            `json:"id"`
	Routes []string          `json:"routes"`
	Name   map[string]string `json:"name"` // language -> name; "en" is required
}

// LoadSeries reads series.json from fsys and checks it against the known routes.
func LoadSeries(fsys fs.FS, path string, known map[string]*Route) ([]Series, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, err
	}
	var file struct {
		Series []Series `json:"series"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	seen := map[string]string{}
	for _, s := range file.Series {
		if !IDPattern.MatchString(s.ID) || s.Name["en"] == "" || len(s.Routes) < 2 {
			return nil, fmt.Errorf("series %q needs an id, an English name and at least two routes", s.ID)
		}
		for _, r := range s.Routes {
			if known[r] == nil {
				return nil, fmt.Errorf("series %q: unknown route %q", s.ID, r)
			}
			if other, ok := seen[r]; ok {
				return nil, fmt.Errorf("route %q is in two series: %q and %q", r, other, s.ID)
			}
			seen[r] = s.ID
		}
	}
	return file.Series, nil
}

// Next is the route after routeID in its series, and the series; false at
// the end of a series or for a route in none.
func Next(series []Series, routeID string) (string, Series, bool) {
	for _, s := range series {
		for i, r := range s.Routes {
			if r == routeID && i+1 < len(s.Routes) {
				return s.Routes[i+1], s, true
			}
		}
	}
	return "", Series{}, false
}

// SeriesName is the series' name in the first available language of chain.
func SeriesName(s Series, chain []string) string {
	for _, l := range chain {
		if n := s.Name[l]; n != "" {
			return n
		}
	}
	return s.Name["en"]
}

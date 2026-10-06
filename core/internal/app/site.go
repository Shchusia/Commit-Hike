package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/Shchusia/commit-hike/core/internal/i18n"
	"github.com/Shchusia/commit-hike/core/internal/protocol"
	"github.com/Shchusia/commit-hike/core/internal/site"
)

// Routes installed from the site, with their version there: what makes "Update" possible.
const siteRoutesFile = "site-routes.json"

type siteInstall struct {
	Version     int       `json:"version"`
	InstalledAt time.Time `json:"installed_at"`
}

func (s *Service) siteInstalls() map[string]siteInstall {
	out := map[string]siteInstall{}
	if data, err := s.st.ReadFile(siteRoutesFile); err == nil {
		_ = json.Unmarshal(data, &out)
	}
	return out
}

func siteErr(err error) error {
	switch {
	case errors.Is(err, site.ErrUnreachable):
		return fail(protocol.CodeSiteUnreachable, "%s", err)
	case errors.Is(err, site.ErrNotFound):
		return fail(protocol.CodeUnknownRoute, "%s", err)
	}
	return fail(protocol.CodeInvalidArgument, "%s", err)
}

// SiteRoutes searches the routes website and says which routes are installed here.
func (s *Service) SiteRoutes(q site.Query, lang string, version string) (*protocol.SitePage, error) {
	client := site.New(version)
	page, err := client.Routes(context.Background(), q)
	if err != nil {
		return nil, siteErr(err)
	}
	installs := s.siteInstalls()
	chain := i18n.Chain(lang)
	out := &protocol.SitePage{Site: client.Base, Total: page.Total, Page: page.Page, Pages: page.Pages, Routes: []protocol.SiteRoute{}}
	for _, c := range page.Routes {
		r := protocol.SiteRoute{
			ID: c.ID, Title: c.Title, Author: c.Author, LengthM: c.LengthM, Stops: c.Stops, AscentM: c.AscentM,
			Real: c.Real, GPS: c.GPS, Languages: c.Languages, Tags: c.Tags, Downloads: c.Downloads, Rating: c.Rating,
			Ratings: c.Ratings, Version: c.Version, Cover: c.Cover, Page: c.Page,
		}
		for _, l := range chain {
			if t := c.Titles[l]; t != "" {
				r.Title = t
				break
			}
		}
		if existing, ok := s.routes[c.ID]; ok {
			switch inst, fromSite := installs[c.ID]; {
			case existing.Builtin:
				r.Installed = "builtin"
			case fromSite:
				r.Installed, r.InstalledVersion, r.Update = "site", inst.Version, c.Version > inst.Version
			default:
				r.Installed = "local" // the user's own route with the same id
			}
		}
		out.Routes = append(out.Routes, r)
	}
	return out, nil
}

// SiteInstall downloads a route from the site and installs it (or updates it).
// A route of the user's own with the same id is replaced only with replaceLocal.
func (s *Service) SiteInstall(id string, replaceLocal bool, lang, version string) (*protocol.SiteInstalled, error) {
	if existing, ok := s.routes[id]; ok {
		if existing.Builtin {
			return nil, fail(protocol.CodeRouteExists, "%q is a built-in route", id)
		}
		if _, fromSite := s.siteInstalls()[id]; !fromSite && !replaceLocal {
			return nil, fail(protocol.CodeRouteExists, "you already have a route of your own with the id %q; replace it?", id)
		}
	}
	client := site.New(version)
	ctx := context.Background()
	card, err := client.Route(ctx, id)
	if err != nil {
		return nil, siteErr(err)
	}
	pack, err := client.Download(ctx, id)
	if err != nil {
		return nil, siteErr(err)
	}
	tmp, err := os.CreateTemp("", "commit-hike-site-*.zip")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(pack); err != nil {
		_ = tmp.Close()
		return nil, err
	}
	if err := tmp.Close(); err != nil {
		return nil, err
	}
	route, err := s.ImportRoute(tmp.Name(), true, lang) // the same checks as any import
	if err != nil {
		return nil, err
	}
	if route.ID != id {
		_ = s.RemoveRoute(route.ID)
		return nil, fail(protocol.CodeInvalidRoute, "the site sent the route %q instead of %q", route.ID, id)
	}

	unlock, err := s.lock()
	if err != nil {
		return nil, err
	}
	defer unlock()
	installs := s.siteInstalls()
	installs[id] = siteInstall{Version: card.Version, InstalledAt: s.now().UTC()}
	data, err := json.MarshalIndent(installs, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := s.st.WriteFile(siteRoutesFile, data); err != nil {
		return nil, err
	}
	return &protocol.SiteInstalled{Route: *route, Version: card.Version, Page: card.Page}, nil
}

// forgetSiteInstall drops a removed route from the site list.
func (s *Service) forgetSiteInstall(id string) {
	installs := s.siteInstalls()
	if _, ok := installs[id]; !ok {
		return
	}
	delete(installs, id)
	if data, err := json.MarshalIndent(installs, "", "  "); err == nil {
		_ = s.st.WriteFile(siteRoutesFile, data)
	}
}

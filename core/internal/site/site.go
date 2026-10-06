// Package site talks to the routes website (commit-hike.dev): it
// searches its catalogue and downloads route packs. Only public data is
// read, and only when the user asks for it; nothing about the user is sent.
package site

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DefaultURL is the routes website. COMMIT_HIKE_SITE overrides it (tests, a mirror).
const DefaultURL = "https://commit-hike.dev"

const (
	maxJSON  = 4 << 20  // a page of the catalogue is a few kilobytes
	maxPack  = 40 << 20 // the site takes packs up to 20 MB, plus translations
	maxImage = 1 << 20  // covers are 480 px thumbnails, tens of kilobytes
)

var imageTypes = map[string]bool{"image/webp": true, "image/png": true, "image/jpeg": true, "image/gif": true}

var idPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ErrUnreachable means the site couldn't be reached: offline, a proxy, the site is down.
var ErrUnreachable = errors.New("the routes site can't be reached")

// BaseURL is the site to use.
func BaseURL() string {
	if v := strings.TrimRight(strings.TrimSpace(os.Getenv("COMMIT_HIKE_SITE")), "/"); strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
		return v
	}
	return DefaultURL
}

// Card is one route of the catalogue, as the site's API describes it.
type Card struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Titles    map[string]string `json:"titles"`
	Author    string            `json:"author"`
	LengthM   float64           `json:"length_m"`
	Stops     int               `json:"stops"`
	AscentM   float64           `json:"ascent_m"`
	Real      bool              `json:"real"`
	GPS       bool              `json:"gps"`
	Languages []string          `json:"languages"`
	Tags      []string          `json:"tags"`
	Downloads int               `json:"downloads"`
	Rating    float64           `json:"rating"`
	Ratings   int               `json:"ratings"`
	Version   int               `json:"version"`
	Cover     string            `json:"cover"`
	Page      string            `json:"page"`
	Download  string            `json:"download"`
}

// Page is a page of the catalogue.
type Page struct {
	Total  int    `json:"total"`
	Page   int    `json:"page"`
	Pages  int    `json:"pages"`
	Routes []Card `json:"routes"`
}

// Query is a search of the catalogue; empty fields mean "any".
type Query struct {
	Q, Kind, Length, Lang, Sort string
	Tags                        []string
	MinRating, Page, Per        int
	GPS                         bool
}

// Client is a client of the site.
type Client struct {
	Base      string
	HTTP      *http.Client
	UserAgent string
}

// New makes a client of BaseURL().
func New(version string) *Client {
	return &Client{Base: BaseURL(), HTTP: &http.Client{Timeout: 30 * time.Second}, UserAgent: "commit-hike/" + version}
}

func (c *Client) get(ctx context.Context, path string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Base+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w (%s): %w", ErrUnreachable, c.Base, err)
	}
	defer func() { _ = resp.Body.Close() }() // a read-only body: nothing to do if closing fails
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("%w (%s): %w", ErrUnreachable, c.Base, err)
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("the site sent more than %d MB", limit>>20)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w (%s): HTTP %d", ErrUnreachable, c.Base, resp.StatusCode)
	}
	return body, nil
}

// ErrNotFound means the site has no (public) route with that id.
var ErrNotFound = errors.New("the site has no such route")

// Routes searches the catalogue.
func (c *Client) Routes(ctx context.Context, q Query) (*Page, error) {
	v := url.Values{}
	set := func(k, val string) {
		if val != "" {
			v.Set(k, val)
		}
	}
	set("q", q.Q)
	set("kind", q.Kind)
	set("length", q.Length)
	set("lang", q.Lang)
	set("sort", q.Sort)
	for _, t := range q.Tags {
		if t = strings.TrimSpace(t); t != "" {
			v.Add("tag", t)
		}
	}
	if q.MinRating > 0 {
		v.Set("min_rating", strconv.Itoa(q.MinRating))
	}
	if q.Page > 1 {
		v.Set("page", strconv.Itoa(q.Page))
	}
	if q.Per > 0 {
		v.Set("per", strconv.Itoa(q.Per))
	}
	if q.GPS {
		v.Set("gps", "1")
	}
	body, err := c.get(ctx, "/api/routes?"+v.Encode(), maxJSON)
	if err != nil {
		return nil, err
	}
	var p Page
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, fmt.Errorf("the site answered something unexpected: %w", err)
	}
	return &p, nil
}

// Route is one route's card.
func (c *Client) Route(ctx context.Context, id string) (*Card, error) {
	if !idPattern.MatchString(id) {
		return nil, fmt.Errorf("%q isn't a route id", id)
	}
	body, err := c.get(ctx, "/api/routes/"+id, maxJSON)
	if err != nil {
		return nil, err
	}
	var card Card
	if err := json.Unmarshal(body, &card); err != nil {
		return nil, fmt.Errorf("the site answered something unexpected: %w", err)
	}
	return &card, nil
}

// Local puts a link from the catalogue on this site: only its path is kept.
// The catalogue builds its links from the site's own idea of its address
// (BASE_URL), which can be stale after a move; the pages, covers and packs the
// plugins use are always on the site the core talks to.
func (c *Client) Local(link string) string {
	if link == "" {
		return ""
	}
	u, err := url.Parse(link)
	if err != nil || u.Path == "" || !strings.HasPrefix(u.Path, "/") {
		return ""
	}
	out := c.Base + u.EscapedPath()
	if u.RawQuery != "" {
		out += "?" + u.RawQuery
	}
	return out
}

// Image fetches a picture of the site (a route's cover) as a data: URL, so the
// plugins' panels show it without going online themselves. Only pictures on the
// site's own address are fetched, whatever the catalogue says.
func (c *Client) Image(ctx context.Context, link string) (string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", err
	}
	base, err := url.Parse(c.Base)
	if err != nil {
		return "", err
	}
	if u.Scheme != base.Scheme || u.Host != base.Host || u.User != nil {
		return "", fmt.Errorf("%q isn't on %s", link, c.Base)
	}
	body, err := c.get(ctx, u.RequestURI(), maxImage)
	if err != nil {
		return "", err
	}
	// The bytes say what it is, not the header: a server may send WebP as
	// text/plain (Python's own type table doesn't know .webp), or lie.
	typ := http.DetectContentType(body)
	if !imageTypes[typ] {
		return "", fmt.Errorf("%q isn't a picture (%s)", link, typ)
	}
	return "data:" + typ + ";base64," + base64.StdEncoding.EncodeToString(body), nil
}

// Download fetches a route's pack (a zip).
func (c *Client) Download(ctx context.Context, id string) ([]byte, error) {
	if !idPattern.MatchString(id) {
		return nil, fmt.Errorf("%q isn't a route id", id)
	}
	body, err := c.get(ctx, "/routes/"+id+"/download", maxPack)
	if err != nil {
		return nil, err
	}
	if len(body) < 4 || string(body[:2]) != "PK" {
		return nil, errors.New("the site didn't send a route pack")
	}
	return body, nil
}

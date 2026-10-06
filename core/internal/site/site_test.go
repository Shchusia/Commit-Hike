package site

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func serve(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &Client{Base: srv.URL, HTTP: srv.Client(), UserAgent: "commit-hike/test"}
}

func TestBaseURL(t *testing.T) {
	t.Setenv("COMMIT_HIKE_SITE", "")
	if BaseURL() != DefaultURL {
		t.Fatal(BaseURL())
	}
	t.Setenv("COMMIT_HIKE_SITE", " http://127.0.0.1:9000/ ")
	if BaseURL() != "http://127.0.0.1:9000" {
		t.Fatal(BaseURL())
	}
	t.Setenv("COMMIT_HIKE_SITE", "file:///etc")
	if BaseURL() != DefaultURL {
		t.Fatal("only http(s) addresses")
	}
}

func TestTheSearchBecomesAQueryString(t *testing.T) {
	var got url.Values
	var agent string
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		got, agent = r.URL.Query(), r.UserAgent()
		_, _ = w.Write([]byte(`{"total":0,"page":1,"pages":1,"routes":[]}`))
	})
	_, err := c.Routes(context.Background(), Query{
		Q: "lakes", Kind: "real", Length: "m", Lang: "de", Sort: "rating",
		Tags: []string{"alps", " ", "lakes"}, MinRating: 4, Page: 3, Per: 48, GPS: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "gps=1&kind=real&lang=de&length=m&min_rating=4&page=3&per=48&q=lakes&sort=rating&tag=alps&tag=lakes"
	if got.Encode() != want || agent != "commit-hike/test" {
		t.Fatalf("%s %s", got.Encode(), agent)
	}
	if _, err := c.Routes(context.Background(), Query{Page: 1}); err != nil || got.Encode() != "" {
		t.Fatalf("an empty search sends nothing: %q %v", got.Encode(), err)
	}
}

func TestWhatTheSiteSendsIsChecked(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/routes":
			_, _ = w.Write([]byte("<html>maintenance</html>"))
		case "/api/routes/broken":
			_, _ = w.Write([]byte("{"))
		case "/routes/not-a-zip/download":
			_, _ = w.Write([]byte("hello"))
		case "/routes/huge/download":
			_, _ = w.Write([]byte("PK" + strings.Repeat("x", maxPack)))
		case "/routes/down/download":
			w.WriteHeader(http.StatusBadGateway)
		default:
			http.NotFound(w, r)
		}
	})
	ctx := context.Background()
	if _, err := c.Routes(ctx, Query{}); err == nil || !strings.Contains(err.Error(), "unexpected") {
		t.Fatalf("not JSON: %v", err)
	}
	if _, err := c.Route(ctx, "broken"); err == nil || !strings.Contains(err.Error(), "unexpected") {
		t.Fatalf("broken JSON: %v", err)
	}
	if _, err := c.Route(ctx, "nowhere"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("404: %v", err)
	}
	if _, err := c.Download(ctx, "not-a-zip"); err == nil || !strings.Contains(err.Error(), "route pack") {
		t.Fatalf("not a zip: %v", err)
	}
	if _, err := c.Download(ctx, "huge"); err == nil || !strings.Contains(err.Error(), "more than") {
		t.Fatalf("too big: %v", err)
	}
	if _, err := c.Download(ctx, "down"); !errors.Is(err, ErrUnreachable) {
		t.Fatalf("a 502 is the site being unreachable: %v", err)
	}
	for _, bad := range []string{"../etc/passwd", "UPPER", ""} {
		if _, err := c.Download(ctx, bad); err == nil {
			t.Fatalf("%q isn't a route id", bad)
		}
		if _, err := c.Route(ctx, bad); err == nil {
			t.Fatalf("%q isn't a route id", bad)
		}
	}
}

func TestOffline(t *testing.T) {
	c := &Client{Base: "http://127.0.0.1:1", HTTP: http.DefaultClient}
	if _, err := c.Routes(context.Background(), Query{}); !errors.Is(err, ErrUnreachable) {
		t.Fatal(err)
	}
	if New("1.0.0").UserAgent != "commit-hike/1.0.0" {
		t.Fatal("the user agent names the version")
	}
}

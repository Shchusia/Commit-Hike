// Package cli maps command-line arguments to app use cases and writes
// exactly one protocol envelope to stdout. It holds no business logic.
package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/commit-hike/commit-hike/core/internal/app"
	"github.com/commit-hike/commit-hike/core/internal/protocol"
	"github.com/commit-hike/commit-hike/core/internal/store"
)

const usage = `commit-hike: turns commits into a journey. Every command prints one JSON envelope.

  commit-hike init     [--email a@x,b@y] [--mode all|selected] [--route ID] [--from-history=true] [--locale uk]
  commit-hike scan     --repo PATH [--lang uk]
  commit-hike status   [--repo PATH] [--lang uk]
  commit-hike routes   [--lang uk]
  commit-hike route    import --path FOLDER|FILE.zip [--replace] [--lang uk]
  commit-hike route    remove --id ID
  commit-hike route    template --id ID --path FOLDER
  commit-hike avatar   get | set --path ICON.png | reset
  commit-hike journey  --scope global|project [--repo PATH] --route ID|none [--from-history]
  commit-hike project  enable|disable --repo PATH
  commit-hike verify   --repo PATH
  commit-hike render   [--scope global|project] [--repo PATH] [--format svg|scene] [--width 300] [--lang uk]
  commit-hike config
  commit-hike version
`

// Run executes one command and returns the process exit code.
func Run(args []string, stdout, stderr io.Writer, version, dataDir string) int {
	data, err := run(args, stderr, version, dataDir)
	env := protocol.Envelope{API: protocol.Version, OK: err == nil}
	if err != nil {
		env.Error = toError(err) // never both: a failed command has no data
	} else {
		env.Data = data
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if encErr := enc.Encode(env); encErr != nil {
		fmt.Fprintln(stderr, encErr)
		return 2
	}
	if err != nil {
		return 1
	}
	return 0
}

func run(args []string, stderr io.Writer, version, dataDir string) (any, error) {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprint(stderr, usage)
		return nil, invalid("a command is required, see `commit-hike help`")
	}
	cmd, args := args[0], args[1:]
	if cmd == "version" {
		return map[string]any{"version": version, "api": protocol.Version}, nil
	}

	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(io.Discard) // errors are reported inside the envelope
	repo := fs.String("repo", "", "path inside a git repository")
	lang := fs.String("lang", "", "language for route texts, e.g. uk or en")

	// "project" and "route" have a positional sub-command before flags.
	sub := ""
	if (cmd == "project" || cmd == "route" || cmd == "avatar") && len(args) > 0 {
		sub, args = args[0], args[1:]
	}

	var (
		emails, mode, route, locale, scope, format, id, path *string
		fromHistory, replace                                 *bool
		width                                                *float64
	)
	switch cmd {
	case "init":
		emails = fs.String("email", "", "comma-separated author emails that are yours")
		mode = fs.String("mode", "", "all | selected")
		route = fs.String("route", "", "route for the global journey")
		locale = fs.String("locale", "", "fixed language; empty follows the IDE")
		fromHistory = fs.Bool("from-history", true, "count existing history")
	case "journey":
		scope = fs.String("scope", app.ScopeGlobal, "global | project")
		route = fs.String("route", "", "route id, or none")
		fromHistory = fs.Bool("from-history", false, "count existing history")
	case "avatar":
		path = fs.String("path", "", "PNG file, ideally with a transparent background")
	case "route":
		id = fs.String("id", "", "route id")
		path = fs.String("path", "", "route folder or .zip (import), target folder (template)")
		replace = fs.Bool("replace", false, "overwrite an installed user route with the same id")
	case "render":
		scope = fs.String("scope", app.ScopeGlobal, "global | project")
		format = fs.String("format", "svg", "svg | scene")
		width = fs.Float64("width", 300, "SVG width in px")
	}
	if err := fs.Parse(args); err != nil {
		return nil, invalid("%s", err)
	}
	if fs.NArg() > 0 {
		return nil, invalid("unexpected argument %q", fs.Arg(0))
	}

	svc, err := app.New(dataDir)
	if err != nil {
		return nil, err
	}
	for _, w := range svc.RouteWarnings() {
		fmt.Fprintln(stderr, "warning:", w) // stderr: never breaks the JSON on stdout
	}

	switch cmd {
	case "init":
		return svc.Init(app.InitOptions{
			Emails: strings.Split(*emails, ","), Mode: *mode, RouteID: *route,
			FromHistory: *fromHistory, Locale: *locale,
		})
	case "scan":
		return svc.Scan(*repo, *lang)
	case "status":
		return svc.Status(*repo, *lang)
	case "routes":
		return svc.Routes(*lang), nil
	case "journey":
		return svc.SetJourney(*scope, *repo, *route, *fromHistory, *lang)
	case "project":
		if sub != "enable" && sub != "disable" {
			return nil, invalid("usage: commit-hike project enable|disable --repo PATH")
		}
		return map[string]bool{"enabled": sub == "enable"}, svc.SetProjectEnabled(*repo, sub == "enable")
	case "route":
		switch sub {
		case "import":
			return svc.ImportRoute(*path, *replace, *lang)
		case "remove":
			return map[string]string{"removed": *id}, svc.RemoveRoute(*id)
		case "template":
			dir, err := svc.RouteTemplate(*id, *path)
			return map[string]string{"path": dir}, err
		}
		return nil, invalid("usage: commit-hike route import|remove|template, see `commit-hike help`")
	case "avatar":
		switch sub {
		case "get":
			return svc.Avatar()
		case "set":
			return svc.SetAvatar(*path)
		case "reset":
			return map[string]bool{"custom": false}, svc.ResetAvatar()
		}
		return nil, invalid("usage: commit-hike avatar get|set|reset")
	case "verify":
		return svc.Verify(*repo)
	case "render":
		return svc.Render(*scope, *repo, *format, *lang, *width)
	case "config":
		return svc.Config()
	}
	return nil, invalid("unknown command %q", cmd)
}

func invalid(format string, args ...any) error {
	return &app.Error{Code: protocol.CodeInvalidArgument, Err: fmt.Errorf(format, args...)}
}

func toError(err error) *protocol.Error {
	var e *app.Error
	switch {
	case errors.As(err, &e):
		return &protocol.Error{Code: e.Code, Message: e.Error()}
	case errors.Is(err, store.ErrNotInitialized):
		return &protocol.Error{Code: protocol.CodeNotInitialized, Message: err.Error()}
	default:
		return &protocol.Error{Code: protocol.CodeInternal, Message: err.Error()}
	}
}

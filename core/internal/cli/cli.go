// Package cli maps command-line arguments to app use cases and writes
// exactly one protocol envelope to stdout. It holds no business logic.
package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Shchusia/commit-hike/core/internal/app"
	"github.com/Shchusia/commit-hike/core/internal/protocol"
	"github.com/Shchusia/commit-hike/core/internal/site"
	"github.com/Shchusia/commit-hike/core/internal/store"
)

const usage = `commit-hike: turns commits into a journey. Every command prints one JSON envelope.

  commit-hike init     [--difficulty easy|medium|hard] [--email a@x,b@y] [--mode all|selected] [--route ID] [--from-history=true] [--locale uk]
  commit-hike scan     --repo PATH [--prev-head SHA] [--lang uk]
  commit-hike status   [--repo PATH] [--lang uk]
  commit-hike routes   [--lang uk]
  commit-hike route    import --path FOLDER|FILE.zip [--replace] [--lang uk]
  commit-hike route    remove --id ID
  commit-hike route    template --id ID --path FOLDER
  commit-hike route    assets --id ID
  commit-hike avatar   get | set --path ICON.png | reset
  commit-hike journey  --scope global|project [--repo PATH] --route ID|none [--from-history]
  commit-hike project  enable|disable --repo PATH
  commit-hike project  team-on|team-off --repo PATH
  commit-hike project  goal --repo PATH --route ID   # a route the team walks together, from now
  commit-hike project  goal-off --repo PATH
  commit-hike team     --repo PATH
  commit-hike locale   [--set auto|en|uk] [--lang uk]
  commit-hike difficulty [--set easy|medium|hard]
  commit-hike rest-days [--set sat,sun|none]
  commit-hike diagnostics                  # for bug reports: versions, counts, settings; nothing personal
  commit-hike badge    [--repo PATH] [--lang uk]   # an SVG badge for a README
  commit-hike settings [--reduce-motion auto|on|off] [--high-contrast auto|on|off] [--notifications all|milestones|off] [--festive on|off] [--ambient on|off]
  commit-hike backup   export --path FILE.json | import --path FILE.json [--replace]
  commit-hike verify   --repo PATH
  commit-hike render   [--scope global|project] [--repo PATH] [--format svg|scene] [--width 300] [--lang uk]
  commit-hike config
  commit-hike site     routes [--q TEXT] [--kind real|story] [--length s|m|l|xl] [--with-lang de] [--tag a,b] [--min-rating 4] [--gps] [--sort new|popular|rating|short|long] [--page 2] [--per 24] [--lang uk]
  commit-hike site     install --id ID [--replace-local] [--lang uk]   # from the routes website; updates too
  commit-hike version
  commit-hike prompt   [--repo PATH] [--lang uk] [--scan] [--icon 🥾]   # one line of plain text, for shell prompts and status lines
`

// Run executes one command and returns the process exit code.
func Run(args []string, stdout, stderr io.Writer, version, dataDir string) int {
	if len(args) > 0 && args[0] == "prompt" {
		return prompt(args[1:], stdout, stderr, dataDir)
	}
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
	if (cmd == "project" || cmd == "route" || cmd == "avatar" || cmd == "backup" || cmd == "site") && len(args) > 0 {
		sub, args = args[0], args[1:]
	}

	var (
		emails, mode, route, locale, scope, format, id, path, prevHead, setLocale, difficulty, restDays *string
		reduceMotion, highContrast, notifications, festive, ambient                                     *string
		fromHistory, replace                                                                            *bool
		width                                                                                           *float64
		siteQ, siteKind, siteLength, siteLang, siteTags, siteSort                                       *string
		siteRating, sitePage, sitePer                                                                   *int
		siteGPS, replaceLocal                                                                           *bool
	)
	switch cmd {
	case "init":
		emails = fs.String("email", "", "comma-separated author emails that are yours")
		mode = fs.String("mode", "", "all | selected")
		route = fs.String("route", "", "route for the global journey")
		locale = fs.String("locale", "", "fixed language; empty follows the IDE")
		fromHistory = fs.Bool("from-history", true, "count existing history")
		difficulty = fs.String("difficulty", "", "easy | medium | hard")
	case "difficulty":
		difficulty = fs.String("set", "", "easy | medium | hard: applies to commits from now on")
	case "rest-days":
		restDays = fs.String("set", "", "weekdays off, e.g. sat,sun; none for no days off")
	case "settings":
		reduceMotion = fs.String("reduce-motion", "", "auto | on | off")
		highContrast = fs.String("high-contrast", "", "auto | on | off")
		notifications = fs.String("notifications", "", "all | milestones | off")
		festive = fs.String("festive", "", "on | off: holidays in the scene")
		ambient = fs.String("ambient", "", "on | off: weather and wildlife in the scene")
	case "backup":
		path = fs.String("path", "", "backup file")
		replace = fs.Bool("replace", false, "import: replace the progress already on this computer")
	case "scan":
		prevHead = fs.String("prev-head", "", "HEAD before this change; a rewrite triggers a full recount")
	case "project":
		route = fs.String("route", "", "project goal: the route the team walks together")
	case "locale":
		setLocale = fs.String("set", "", "auto | en | uk | …")
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
	case "site":
		id = fs.String("id", "", "route id")
		siteQ = fs.String("q", "", "search text")
		siteKind = fs.String("kind", "", "real | story")
		siteLength = fs.String("length", "", "s | m | l | xl")
		siteLang = fs.String("with-lang", "", "only routes in this language")
		siteTags = fs.String("tag", "", "tags, comma-separated")
		siteRating = fs.Int("min-rating", 0, "1..5")
		siteGPS = fs.Bool("gps", false, "only real GPS tracks")
		siteSort = fs.String("sort", "", "new | popular | rating | short | long")
		sitePage = fs.Int("page", 1, "page")
		sitePer = fs.Int("per", 0, "routes per page (12, 24, 48, 96)")
		replaceLocal = fs.Bool("replace-local", false, "replace your own route with the same id")
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
			FromHistory: *fromHistory, Locale: *locale, Difficulty: *difficulty,
		})
	case "difficulty":
		return svc.Difficulty(*difficulty)
	case "rest-days":
		return svc.RestDays(*restDays)
	case "diagnostics":
		return svc.Diagnostics(version)
	case "badge":
		return svc.Badge(*repo, *lang)
	case "settings":
		return svc.Settings(app.SettingsChange{ReduceMotion: *reduceMotion, HighContrast: *highContrast, Notifications: *notifications, Festive: *festive, Ambient: *ambient})
	case "backup":
		switch sub {
		case "export":
			return svc.ExportBackup(*path, version)
		case "import":
			return svc.ImportBackup(*path, *replace)
		}
		return nil, invalid("usage: commit-hike backup export|import --path FILE")
	case "scan":
		return svc.ScanWith(*repo, *lang, app.ScanOptions{PrevHead: *prevHead})
	case "team":
		return svc.Team(*repo, *lang)
	case "locale":
		var set *string
		fs.Visit(func(f *flag.Flag) {
			if f.Name == "set" {
				set = setLocale
			}
		})
		return svc.Locale(set, *lang)
	case "status":
		return svc.Status(*repo, *lang)
	case "routes":
		return svc.Routes(*lang), nil
	case "journey":
		return svc.SetJourney(*scope, *repo, *route, *fromHistory, *lang)
	case "project":
		switch sub {
		case "enable", "disable":
			return map[string]bool{"enabled": sub == "enable"}, svc.SetProjectEnabled(*repo, sub == "enable")
		case "team-on", "team-off":
			return map[string]bool{"team": sub == "team-on"}, svc.SetTeam(*repo, sub == "team-on")
		case "goal":
			if *route == "" {
				return nil, invalid("usage: commit-hike project goal --repo PATH --route ID")
			}
			return map[string]string{"goal": *route}, svc.SetTeamGoal(*repo, *route)
		case "goal-off":
			return map[string]string{"goal": ""}, svc.SetTeamGoal(*repo, "")
		}
		return nil, invalid("usage: commit-hike project enable|disable|team-on|team-off --repo PATH")
	case "site":
		switch sub {
		case "routes":
			var tags []string
			if *siteTags != "" {
				tags = strings.Split(*siteTags, ",")
			}
			return svc.SiteRoutes(site.Query{
				Q: *siteQ, Kind: *siteKind, Length: *siteLength, Lang: *siteLang, Tags: tags,
				MinRating: *siteRating, GPS: *siteGPS, Sort: *siteSort, Page: *sitePage, Per: *sitePer,
			}, *lang, version)
		case "install":
			return svc.SiteInstall(*id, *replaceLocal, *lang, version)
		}
		return nil, invalid("usage: commit-hike site routes|install, see `commit-hike help`")
	case "route":
		switch sub {
		case "import":
			return svc.ImportRoute(*path, *replace, *lang)
		case "remove":
			return map[string]string{"removed": *id}, svc.RemoveRoute(*id)
		case "template":
			dir, err := svc.RouteTemplate(*id, *path)
			return map[string]string{"path": dir}, err
		case "assets":
			return svc.RouteAssets(*id)
		}
		return nil, invalid("usage: commit-hike route import|remove|template|assets, see `commit-hike help`")
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

// prompt is the one command without a JSON envelope: shell prompts and
// editor status lines print its output as it is. It never fails loudly: on
// any problem it prints nothing (the reason goes to stderr) and exits 0, so a
// prompt never breaks.
func prompt(args []string, stdout, stderr io.Writer, dataDir string) int {
	fs := flag.NewFlagSet("prompt", flag.ContinueOnError)
	fs.SetOutput(stderr)
	repo := fs.String("repo", "", "path inside a git repository (default: the current folder)")
	lang := fs.String("lang", "", "language, e.g. uk (default: from LC_ALL, LC_MESSAGES or LANG)")
	scan := fs.Bool("scan", false, "count new commits first when HEAD moved since the last prompt")
	icon := fs.String("icon", "🥾", "shown before the distance; empty for none")
	if err := fs.Parse(args); err != nil {
		return 0
	}
	if *repo == "" {
		if wd, err := os.Getwd(); err == nil {
			*repo = wd
		}
	}
	if *lang == "" {
		*lang = app.LangFromEnv()
	}
	svc, err := app.New(dataDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 0
	}
	line, err := svc.Prompt(app.PromptOptions{Repo: *repo, Lang: *lang, Icon: *icon, Scan: *scan})
	if err != nil {
		if !errors.Is(err, store.ErrNotInitialized) {
			fmt.Fprintln(stderr, err)
		}
		return 0
	}
	if line != "" {
		fmt.Fprintln(stdout, line)
	}
	return 0
}

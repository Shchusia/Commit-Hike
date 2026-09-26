// Package gitlog reads commit history through the git CLI.
//
// It is strictly read-only: it never writes to the repository, never installs
// hooks and never touches .git/config.
package gitlog

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

// FileChange is one file in a commit, as reported by `git log --numstat`.
type FileChange struct {
	Path    string // new path for renames
	Added   int
	Deleted int
	Binary  bool
}

// Commit is a commit with the fields the core needs.
type Commit struct {
	Hash        string
	AuthorEmail string
	AuthorTime  time.Time
	Parents     int
	Files       []FileChange
}

// Errors returned when a path can't be used as a project.
var (
	ErrNotRepo   = errors.New("not a git repository")
	ErrNoCommits = errors.New("repository has no commits yet")
)

func run(repo string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	// Stable, locale-independent output; never prompt for credentials.
	cmd.Env = append(os.Environ(), "LC_ALL=C", "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// GlobalEmail returns `git config --global user.email`, or "" if unset.
func GlobalEmail() string {
	out, err := exec.Command("git", "config", "--global", "user.email").Output()
	if err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(string(out)))
}

// TopLevel returns the repository root for any path inside a work tree.
func TopLevel(dir string) (string, error) {
	out, err := run(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", ErrNotRepo
	}
	return strings.TrimSpace(string(out)), nil
}

// RootCommit returns a stable project identity: the (lexicographically first)
// root commit hash. It is identical in every clone and never changes.
func RootCommit(repo string) (string, error) {
	// Only HEAD's ancestry: orphan branches (gh-pages etc.) may exist in one
	// clone and not another, which would make the identity unstable.
	out, err := run(repo, "rev-list", "--max-parents=0", "HEAD")
	if err != nil {
		return "", ErrNoCommits
	}
	roots := strings.Fields(string(out))
	if len(roots) == 0 {
		return "", ErrNoCommits
	}
	sort.Strings(roots)
	return roots[0], nil
}

// Log returns non-merge commits reachable from local branches, tags and
// remote-tracking branches (stash is deliberately excluded). since==zero means
// full history; otherwise only commits with committer date >= since.
func Log(repo string, since time.Time) ([]Commit, error) {
	args := []string{
		"log", "-z", "--no-merges", "--numstat",
		"-w",             // whitespace-only changes do not count
		"--find-renames", // pure renames show 0/0
		"--format=%x1e%H%x1f%ae%x1f%at%x1f%P",
		"--branches", "--tags", "--remotes",
	}
	if !since.IsZero() {
		args = append(args, "--since="+strconv.FormatInt(since.Unix(), 10))
	}
	out, err := run(repo, args...)
	if err != nil {
		return nil, err
	}
	return Parse(out)
}

// Parse decodes the output of Log. Exposed for tests.
//
// Layout per commit (with -z):
//
//	\x1e HASH \x1f EMAIL \x1f UNIXTIME \x1f PARENTS \0 \n
//	ADDED \t DELETED \t PATH \0              (regular file)
//	ADDED \t DELETED \t \0 OLD \0 NEW \0     (rename / copy)
func Parse(out []byte) ([]Commit, error) {
	var commits []Commit
	for _, rec := range bytes.Split(out, []byte{0x1e}) {
		if len(bytes.TrimSpace(rec)) == 0 {
			continue
		}
		hdrEnd := bytes.IndexByte(rec, 0)
		if hdrEnd < 0 {
			hdrEnd = len(rec)
		}
		hdr := strings.Split(strings.TrimSpace(string(rec[:hdrEnd])), "\x1f")
		if len(hdr) < 4 {
			return nil, fmt.Errorf("bad commit header %q", rec[:hdrEnd])
		}
		ts, err := strconv.ParseInt(hdr[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("bad timestamp %q", hdr[2])
		}
		c := Commit{
			Hash:        hdr[0],
			AuthorEmail: strings.ToLower(strings.TrimSpace(hdr[1])),
			AuthorTime:  time.Unix(ts, 0).UTC(),
			Parents:     len(strings.Fields(hdr[3])),
		}
		var body []byte
		if hdrEnd < len(rec) {
			body = rec[hdrEnd+1:]
		}
		toks := strings.Split(string(body), "\x00")
		for i := 0; i < len(toks); i++ {
			t := strings.TrimLeft(toks[i], "\n")
			if t == "" {
				continue
			}
			parts := strings.SplitN(t, "\t", 3)
			if len(parts) != 3 {
				continue
			}
			fc := FileChange{Path: parts[2]}
			if parts[0] == "-" || parts[1] == "-" {
				fc.Binary = true
			} else {
				fc.Added, _ = strconv.Atoi(parts[0])
				fc.Deleted, _ = strconv.Atoi(parts[1])
			}
			if fc.Path == "" { // rename: next two tokens are old and new path
				if i+2 < len(toks) {
					fc.Path = toks[i+2]
				}
				i += 2
			}
			c.Files = append(c.Files, fc)
		}
		commits = append(commits, c)
	}
	return commits, nil
}

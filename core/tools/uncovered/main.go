// Command uncovered prints what the tests don't cover: a per-package summary,
// then every uncovered line range with the function it belongs to.
//
//	go test -coverprofile=cover.out ./...
//	go run ./tools/uncovered -profile cover.out [-min 75]
//
// With -min it exits with status 1 when total coverage is below the threshold,
// so it can guard CI.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Block is one coverage block from a Go cover profile.
type Block struct {
	File               string
	StartLine, EndLine int
	Statements, Count  int
}

// Parse reads a cover profile. Blocks that appear several times (profiles of
// several packages) are merged: a block counts as covered if any run hit it.
func Parse(r io.Reader) ([]Block, error) {
	seen := map[string]*Block{}
	var order []string
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}
		// file.go:12.34,15.2 3 1
		colon := strings.LastIndex(line, ":")
		fields := strings.Fields(line[colon+1:])
		if colon < 0 || len(fields) != 3 {
			return nil, fmt.Errorf("bad profile line %q", line)
		}
		pos := strings.Split(fields[0], ",")
		if len(pos) != 2 {
			return nil, fmt.Errorf("bad position in %q", line)
		}
		start, err1 := strconv.Atoi(strings.Split(pos[0], ".")[0])
		end, err2 := strconv.Atoi(strings.Split(pos[1], ".")[0])
		stmts, err3 := strconv.Atoi(fields[1])
		count, err4 := strconv.Atoi(fields[2])
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			return nil, fmt.Errorf("bad numbers in %q", line)
		}
		key := line[:colon] + ":" + fields[0]
		if b, ok := seen[key]; ok {
			b.Count += count
			continue
		}
		seen[key] = &Block{File: line[:colon], StartLine: start, EndLine: end, Statements: stmts, Count: count}
		order = append(order, key)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	out := make([]Block, 0, len(order))
	for _, k := range order {
		out = append(out, *seen[k])
	}
	return out, nil
}

// Range is a run of uncovered lines in one file.
type Range struct {
	File       string
	From, To   int
	Statements int
}

// Uncovered joins adjacent uncovered blocks into line ranges, sorted by file.
func Uncovered(blocks []Block) []Range {
	var zero []Block
	for _, b := range blocks {
		if b.Count == 0 && b.Statements > 0 {
			zero = append(zero, b)
		}
	}
	sort.Slice(zero, func(i, j int) bool {
		if zero[i].File != zero[j].File {
			return zero[i].File < zero[j].File
		}
		return zero[i].StartLine < zero[j].StartLine
	})
	var out []Range
	for _, b := range zero {
		if n := len(out); n > 0 && out[n-1].File == b.File && b.StartLine <= out[n-1].To+1 {
			out[n-1].To = max(out[n-1].To, b.EndLine)
			out[n-1].Statements += b.Statements
			continue
		}
		out = append(out, Range{File: b.File, From: b.StartLine, To: b.EndLine, Statements: b.Statements})
	}
	return out
}

// Summary is statement coverage per package.
type Summary struct {
	Package        string
	Covered, Total int
}

// Percent returns coverage in percent (100 for a package without statements).
func (s Summary) Percent() float64 {
	if s.Total == 0 {
		return 100
	}
	return 100 * float64(s.Covered) / float64(s.Total)
}

// Summarize groups statements by package, least covered first.
func Summarize(blocks []Block) (pkgs []Summary, total Summary) {
	by := map[string]*Summary{}
	for _, b := range blocks {
		p := path.Dir(b.File)
		s := by[p]
		if s == nil {
			s = &Summary{Package: p}
			by[p] = s
		}
		s.Total += b.Statements
		total.Total += b.Statements
		if b.Count > 0 {
			s.Covered += b.Statements
			total.Covered += b.Statements
		}
	}
	for _, s := range by {
		pkgs = append(pkgs, *s)
	}
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].Percent() < pkgs[j].Percent() })
	total.Package = "total"
	return pkgs, total
}

// funcNames maps line numbers to the enclosing function, from source on disk.
func funcNames(file string) func(line int) string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, parser.SkipObjectResolution)
	if err != nil {
		return func(int) string { return "" }
	}
	type span struct {
		from, to int
		name     string
	}
	var spans []span
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok {
			name := fd.Name.Name
			if fd.Recv != nil && len(fd.Recv.List) > 0 {
				t := fd.Recv.List[0].Type
				if st, ok := t.(*ast.StarExpr); ok {
					t = st.X
				}
				if id, ok := t.(*ast.Ident); ok {
					name = id.Name + "." + name
				}
			}
			spans = append(spans, span{fset.Position(fd.Pos()).Line, fset.Position(fd.End()).Line, name})
		}
	}
	return func(line int) string {
		for _, s := range spans {
			if line >= s.from && line <= s.to {
				return s.name
			}
		}
		return ""
	}
}

func main() {
	profile := flag.String("profile", "cover.out", "cover profile from go test -coverprofile")
	module := flag.String("module", "", "module path to strip from file names (read from go.mod by default)")
	minPct := flag.Float64("min", 0, "fail when total coverage is below this percentage")
	brief := flag.Bool("brief", false, "only the summary, not every uncovered line")
	summary := flag.String("summary", "", "also write the total to this JSON file (for the coverage table)")
	flag.Parse()

	f, err := os.Open(*profile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	blocks, err := Parse(f)
	_ = f.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	mod := *module
	if mod == "" {
		mod = modulePath("go.mod")
	}
	rel := func(p string) string { return strings.TrimPrefix(strings.TrimPrefix(p, mod), "/") }

	pkgs, total := Summarize(blocks)
	fmt.Println("Coverage by package (least covered first):")
	for _, s := range pkgs {
		fmt.Printf("  %6.1f%%  %4d/%-4d  %s\n", s.Percent(), s.Covered, s.Total, rel(s.Package))
	}
	fmt.Printf("  %6.1f%%  %4d/%-4d  total\n\n", total.Percent(), total.Covered, total.Total)

	ranges := Uncovered(blocks)
	if *summary != "" {
		line := fmt.Sprintf(`{"part":"Core (Go)","percent":%.1f,"detail":"statements · %d places not covered"}`, total.Percent(), len(ranges))
		if err := os.WriteFile(*summary, []byte(line), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	if *brief {
		fmt.Printf("%d places not covered: task core:cover lists them\n", len(ranges))
		ranges = nil
	} else {
		fmt.Printf("Not covered by tests (%d places):\n", len(ranges))
	}
	names := map[string]func(int) string{}
	for _, r := range ranges {
		local := filepath.FromSlash(rel(r.File))
		if names[local] == nil {
			names[local] = funcNames(local)
		}
		lines := strconv.Itoa(r.From)
		if r.To > r.From {
			lines += "-" + strconv.Itoa(r.To)
		}
		fn := names[local](r.From)
		if fn != "" {
			fn = "  " + fn
		}
		fmt.Printf("  %s:%s%s  (%d stmt)\n", rel(r.File), lines, fn, r.Statements)
	}
	if *minPct > 0 && total.Percent() < *minPct {
		fmt.Printf("\nTotal coverage %.1f%% is below the required %.1f%%\n", total.Percent(), *minPct)
		os.Exit(1)
	}
}

func modulePath(gomod string) string {
	b, err := os.ReadFile(gomod)
	if err != nil {
		return ""
	}
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(l, "module "))
		}
	}
	return ""
}

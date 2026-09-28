package main

import (
	"reflect"
	"strings"
	"testing"
)

const profile = `mode: atomic
m/a/x.go:10.2,12.3 2 1
m/a/x.go:13.2,14.3 1 0
m/a/x.go:15.2,16.3 2 0
m/a/x.go:30.2,31.3 1 0
m/b/y.go:1.2,2.3 4 0
m/b/y.go:1.2,2.3 4 3
`

func TestParseMergesRepeatedBlocks(t *testing.T) {
	blocks, err := Parse(strings.NewReader(profile))
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 5 || blocks[4].Count != 3 {
		t.Fatalf("blocks: %+v", blocks)
	}
	if _, err := Parse(strings.NewReader("m/x.go:bad\n")); err == nil {
		t.Error("bad lines must fail")
	}
}

func TestUncoveredJoinsAdjacentBlocks(t *testing.T) {
	blocks, _ := Parse(strings.NewReader(profile))
	got := Uncovered(blocks)
	want := []Range{
		{File: "m/a/x.go", From: 13, To: 16, Statements: 3},
		{File: "m/a/x.go", From: 30, To: 31, Statements: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
}

func TestSummarize(t *testing.T) {
	blocks, _ := Parse(strings.NewReader(profile))
	pkgs, total := Summarize(blocks)
	if pkgs[0].Package != "m/a" || pkgs[0].Covered != 2 || pkgs[0].Total != 6 {
		t.Fatalf("least covered first: %+v", pkgs)
	}
	if total.Covered != 6 || total.Total != 10 || total.Percent() != 60 {
		t.Fatalf("total: %+v", total)
	}
	if (Summary{}).Percent() != 100 {
		t.Error("an empty package is fully covered")
	}
}

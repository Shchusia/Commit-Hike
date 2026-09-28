package gitlog

import "testing"

func TestParse(t *testing.T) {
	raw := "\x1eaaa\x1fMe@X.io\x1f1790391437\x1fp1\x00\n" +
		"3\t1\tmain.go\x00" +
		"-\t-\timg.png\x00" +
		"0\t0\t\x00old name.go\x00d ir/new name.go\x00" +
		"\x1ebbb\x1fme@x.io\x1f1790391400\x1fp1 p2\x00\n" +
		"\x1eccc\x1fme@x.io\x1f1790391300\x1f\x00\n2\t0\tf\x00"
	cs, err := Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 3 {
		t.Fatalf("got %d commits", len(cs))
	}
	a := cs[0]
	if a.AuthorEmail != "me@x.io" || a.Parents != 1 || len(a.Files) != 3 {
		t.Fatalf("commit a: %+v", a)
	}
	if !a.Files[1].Binary || a.Files[2].Path != "d ir/new name.go" || a.Files[0].Added != 3 {
		t.Fatalf("files: %+v", a.Files)
	}
	if cs[1].Parents != 2 || len(cs[1].Files) != 0 {
		t.Fatalf("merge: %+v", cs[1])
	}
	if cs[2].Parents != 0 || cs[2].Files[0].Added != 2 {
		t.Fatalf("root: %+v", cs[2])
	}
}

func TestParseMailmapIdentity(t *testing.T) {
	raw := "\x1eaaa\x1fold@x.io\x1f1790391437\x1f\x1fAnna Koval\x1fAnna@Team.io\x00\n1\t0\ta\x00" +
		"\x1ebbb\x1fme@x.io\x1f1790391400\x1f\x1f\x1f\x00\n"
	cs, err := Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if cs[0].AuthorEmail != "old@x.io" || cs[0].MailmapMail != "anna@team.io" || cs[0].AuthorName != "Anna Koval" {
		t.Fatalf("mailmap: %+v", cs[0])
	}
	if cs[1].AuthorName != "me@x.io" || cs[1].MailmapMail != "me@x.io" {
		t.Fatalf("fallback: %+v", cs[1])
	}
}
